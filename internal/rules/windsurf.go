package rules

import (
	"fmt"
	"os"
	"path/filepath"
)

// WindsurfInstaller manages Windsurf's user-scope (global) rules file at
// `~/.codeium/windsurf/memories/global_rules.md`. We inject the protocol
// inside a managed block so the user's other notes around it are preserved.
type WindsurfInstaller struct {
	// path overrides ~/.codeium/.../global_rules.md for tests.
	path string
}

// NewWindsurfInstaller returns an installer pointed at the default path.
func NewWindsurfInstaller() *WindsurfInstaller { return &WindsurfInstaller{} }

// NewWindsurfInstallerAt returns an installer rooted at an explicit file.
func NewWindsurfInstallerAt(path string) *WindsurfInstaller {
	return &WindsurfInstaller{path: path}
}

// Name implements Installer.
func (w *WindsurfInstaller) Name() string { return "windsurf" }

// RulesPath is the global_rules.md file we mutate.
func (w *WindsurfInstaller) RulesPath() string {
	if w.path != "" {
		return w.path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codeium", "windsurf", "memories", "global_rules.md")
}

// Detect returns true when the Windsurf config root exists. We probe the
// memories directory (created on first Windsurf launch); the rules file
// itself need not pre-exist.
func (w *WindsurfInstaller) Detect() bool {
	path := w.RulesPath()
	if path == "" {
		return false
	}
	if _, err := os.Stat(filepath.Dir(path)); err == nil {
		return true
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// Install upserts the managed block into global_rules.md. If the file
// doesn't exist yet it's created with only the managed block; if it does,
// the existing block (if any) is replaced; otherwise the block is appended
// after the existing content. Idempotent across repeated calls.
func (w *WindsurfInstaller) Install(text string) error {
	path := w.RulesPath()
	if path == "" {
		return fmt.Errorf("windsurf: cannot resolve rules path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing, err := readOrEmpty(path)
	if err != nil {
		return err
	}
	updated := upsertManagedBlock(existing, text)
	return os.WriteFile(path, []byte(updated), 0o644)
}

// Uninstall strips the managed block; non-Ramorie content is preserved.
func (w *WindsurfInstaller) Uninstall() error {
	path := w.RulesPath()
	if path == "" {
		return nil
	}
	existing, err := readOrEmpty(path)
	if err != nil {
		return err
	}
	if existing == "" {
		return nil
	}
	updated := removeManagedBlock(existing)
	return os.WriteFile(path, []byte(updated), 0o644)
}

// Status returns whether a managed block exists and the version marker
// captured from its start tag.
func (w *WindsurfInstaller) Status() (bool, string, error) {
	path := w.RulesPath()
	if path == "" {
		return false, "", nil
	}
	existing, err := readOrEmpty(path)
	if err != nil {
		return false, "", err
	}
	if !managedBlockRegexp.MatchString(existing) {
		return false, "", nil
	}
	return true, extractVersion(existing), nil
}

// readOrEmpty returns file content or empty string if the file is missing.
func readOrEmpty(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
