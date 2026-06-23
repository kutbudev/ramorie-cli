package selfupdate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"9.1.0", "9.2.0", true},
		{"9.1.0", "9.1.1", true},
		{"9.1.0", "10.0.0", true},
		{"v9.1.0", "v9.2.0", true},
		{"9.2.0", "9.2.0", false},
		{"9.2.0", "9.1.9", false},
		{"9.2.0", "9.1.10", false},
		{"10.0.0", "9.9.9", false},
		{"9.2.0", "9.2.0-rc1", false}, // pre-release of same version is not newer
		{"9.1.0", "9.2.0-rc1", true},
		{"0.0.0", "9.1.0", true},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestMethodString(t *testing.T) {
	for m, want := range map[Method]string{
		MethodBinary: "binary",
		MethodBrew:   "homebrew",
		MethodNpm:    "npm",
	} {
		if got := m.String(); got != want {
			t.Errorf("Method(%d).String() = %q, want %q", m, got, want)
		}
	}
}

func TestHookRefreshBinary_PrefersPathRamorie(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "ramorie")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got := hookRefreshBinary("/fallback/ramorie")
	if got != fake {
		t.Fatalf("hookRefreshBinary = %q, want PATH binary %q", got, fake)
	}
}

func TestHookRefreshBinary_FallsBackToExePath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	got := hookRefreshBinary("/fallback/ramorie")
	if got != "/fallback/ramorie" {
		t.Fatalf("hookRefreshBinary fallback = %q", got)
	}
}

func TestRefreshProtocolHooks_RunsSetupHooksInstall(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	fake := filepath.Join(dir, "ramorie")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellArg(argsFile) + "\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	var out bytes.Buffer
	refreshProtocolHooks(context.Background(), &out, "")

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("expected fake ramorie to run: %v\noutput:\n%s", err, out.String())
	}
	got := strings.TrimSpace(string(data))
	want := strings.Join([]string{"setup-hooks", "install", "--client", "all"}, "\n")
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
	if !strings.Contains(out.String(), "Protocol hooks refreshed") {
		t.Fatalf("expected success output, got:\n%s", out.String())
	}
}

func TestRefreshProtocolHooks_SkipEnv(t *testing.T) {
	t.Setenv(skipHookRefreshEnv, "1")
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer
	refreshProtocolHooks(context.Background(), &out, "/does/not/exist")

	if !strings.Contains(out.String(), "Skipping protocol hook refresh") {
		t.Fatalf("expected skip message, got:\n%s", out.String())
	}
}

func shellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
