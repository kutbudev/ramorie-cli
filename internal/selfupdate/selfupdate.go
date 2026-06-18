// Package selfupdate implements the `ramorie update` self-upgrade flow and the
// passive "a new version is available" notice.
//
// The CLI ships three ways (Homebrew, npm, and a direct GoReleaser binary), so
// updating is HYBRID: when the running binary lives inside a Homebrew Cellar or
// an npm global tree we delegate to that package manager; otherwise we download
// the matching release asset from GitHub and atomically replace the binary in
// place — the codex/claude-code experience.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	repo          = "kutbudev/ramorie-cli"
	checkInterval = 24 * time.Hour
	userAgent     = "ramorie-cli-selfupdate"
)

// ---- version comparison ----------------------------------------------------

// Newer reports whether latest is a strictly higher version than current.
// Both may carry a leading "v"; pre-release suffixes (e.g. "-rc1") are ignored
// for the numeric comparison and only break ties in favor of the release.
func Newer(current, latest string) bool {
	return compareVersions(latest, current) > 0
}

// compareVersions returns >0 if a>b, <0 if a<b, 0 if equal (numeric, dotted).
func compareVersions(a, b string) int {
	an := splitVersion(a)
	bn := splitVersion(b)
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			if an[i] > bn[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func splitVersion(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 { // drop pre-release / build metadata
		v = v[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		out[i], _ = strconv.Atoi(strings.TrimSpace(part))
	}
	return out
}

// ---- GitHub release lookup -------------------------------------------------

type ghRelease struct {
	TagName string `json:"tag_name"`
}

// LatestVersion returns the newest published release version (without a leading
// "v"). The context bounds the network call.
func LatestVersion(ctx context.Context) (string, error) {
	url := "https://api.github.com/repos/" + repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github releases API returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return "", err
	}
	tag := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	if tag == "" {
		return "", fmt.Errorf("github release has empty tag")
	}
	return tag, nil
}

// ---- passive update notice -------------------------------------------------

type updateCache struct {
	LastCheck time.Time `json:"last_check"`
	Latest    string    `json:"latest_version"`
}

func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ramorie", "update-check.json"), nil
}

func loadCache() updateCache {
	var c updateCache
	p, err := cachePath()
	if err != nil {
		return c
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

func saveCache(c updateCache) {
	p, err := cachePath()
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

// MaybeNotify prints a one-line upgrade hint to w when a newer version is
// known. It is best-effort and never blocks meaningfully: the latest version
// is cached for checkInterval, so only the first run each day pays a short
// network call. It stays silent when w is not a terminal (so the MCP stdio
// server and piped output are never polluted) or when RAMORIE_NO_UPDATE_CHECK
// is set.
func MaybeNotify(w io.Writer, current string) {
	if os.Getenv("RAMORIE_NO_UPDATE_CHECK") != "" {
		return
	}
	if f, ok := w.(*os.File); !ok || !term.IsTerminal(int(f.Fd())) {
		return
	}

	c := loadCache()
	if time.Since(c.LastCheck) > checkInterval {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		if latest, err := LatestVersion(ctx); err == nil {
			c = updateCache{LastCheck: time.Now(), Latest: latest}
			saveCache(c)
		}
	}

	if c.Latest != "" && Newer(current, c.Latest) {
		fmt.Fprintf(w, "\n\033[2m─\033[0m A new version of ramorie is available: \033[1m%s\033[0m → \033[1;38;2;138;135;255m%s\033[0m\n  Run \033[1mramorie update\033[0m to upgrade.\n", current, c.Latest)
	}
}

// ---- install-method detection ----------------------------------------------

// Method is how the running binary was installed.
type Method int

const (
	MethodBinary Method = iota // direct GoReleaser binary — self-replace
	MethodBrew                 // Homebrew Cellar — delegate to brew
	MethodNpm                  // npm global tree — delegate to npm
)

func (m Method) String() string {
	switch m {
	case MethodBrew:
		return "homebrew"
	case MethodNpm:
		return "npm"
	default:
		return "binary"
	}
}

// DetectMethod resolves the running executable (following symlinks) and infers
// how it was installed. It returns the method and the resolved path.
func DetectMethod() (Method, string) {
	exe, err := os.Executable()
	if err != nil {
		return MethodBinary, exe
	}
	resolved := exe
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		resolved = r
	}
	lower := strings.ToLower(resolved)
	switch {
	case strings.Contains(lower, "/node_modules/"):
		return MethodNpm, resolved
	case strings.Contains(lower, "/cellar/") || strings.Contains(lower, "/homebrew/"):
		return MethodBrew, resolved
	}
	return MethodBinary, resolved
}

// ---- update ----------------------------------------------------------------

// Update upgrades the CLI to the latest release. When force is false and the
// binary is already current, it returns without doing work. Progress is
// written to w.
func Update(ctx context.Context, current string, force bool, w io.Writer) error {
	fmt.Fprintln(w, "Checking for the latest release…")
	latest, err := LatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("could not reach GitHub releases: %w", err)
	}
	if !force && !Newer(current, latest) {
		fmt.Fprintf(w, "✓ ramorie is already up to date (%s).\n", current)
		return nil
	}
	fmt.Fprintf(w, "Updating ramorie %s → %s\n", current, latest)

	method, exePath := DetectMethod()
	switch method {
	case MethodBrew:
		return runManager(ctx, w, "homebrew", "brew", "upgrade", "kutbudev/tap/ramorie")
	case MethodNpm:
		return runManager(ctx, w, "npm", "npm", "install", "-g", "ramorie@latest")
	default:
		return selfReplace(ctx, w, latest, exePath)
	}
}

// runManager shells out to a package manager (brew/npm) and streams its output.
func runManager(ctx context.Context, w io.Writer, label, bin string, args ...string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("installed via %s but %q is not on PATH — run: %s %s",
			label, bin, bin, strings.Join(args, " "))
	}
	fmt.Fprintf(w, "Detected a %s install — delegating: %s %s\n", label, bin, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s upgrade failed: %w", label, err)
	}
	fmt.Fprintln(w, "✓ Upgrade complete.")
	return nil
}

// selfReplace downloads the release asset matching the current OS/arch and
// atomically swaps it in for the running executable.
func selfReplace(ctx context.Context, w io.Writer, version, exePath string) error {
	goos := runtime.GOOS
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	asset := fmt.Sprintf("ramorie_%s_%s_%s.%s", version, goos, runtime.GOARCH, ext)
	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", repo, version, asset)

	fmt.Fprintf(w, "Downloading %s…\n", asset)
	bin, err := downloadBinary(ctx, url, ext)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, ".ramorie-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (try: sudo ramorie update): %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		// A running .exe can't be overwritten directly; move it aside first.
		_ = os.Rename(exePath, exePath+".old")
	}
	if err := os.Rename(tmpName, exePath); err != nil {
		return fmt.Errorf("could not replace %s (try: sudo ramorie update): %w", exePath, err)
	}
	fmt.Fprintf(w, "✓ Updated to %s.\n", version)
	return nil
}

// downloadBinary fetches the archive at url and returns the extracted `ramorie`
// executable bytes.
func downloadBinary(ctx context.Context, url, ext string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s (%s)", resp.Status, url)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return nil, err
	}
	if ext == "zip" {
		return extractZip(data)
	}
	return extractTarGz(data)
}

func extractTarGz(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == "ramorie" && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("ramorie binary not found in archive")
}

func extractZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == "ramorie.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("ramorie.exe not found in archive")
}
