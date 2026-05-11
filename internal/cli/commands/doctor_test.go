package commands

import (
	"strings"
	"testing"

	"github.com/kutbudev/ramorie-cli/internal/config"
)

// TestDoctorResult_Symbol confirms the glyph mapping is stable so external
// log scrapers can rely on the markers.
func TestDoctorResult_Symbol(t *testing.T) {
	cases := []struct {
		status doctorStatus
		want   string
	}{
		{doctorOK, "✓"},
		{doctorWarn, "⚠"},
		{doctorFail, "✗"},
		{doctorStatus(99), "·"},
	}
	for _, c := range cases {
		got := doctorResult{Status: c.status}.symbol()
		if got != c.want {
			t.Errorf("status %d → %q, want %q", c.status, got, c.want)
		}
	}
}

// TestRunDoctorScope_UnknownScopeDegrades verifies a typo doesn't silently
// skip the whole battery — we still run "all" and prepend a Warn note.
func TestRunDoctorScope_UnknownScopeDegrades(t *testing.T) {
	results := runDoctorScope("not-a-scope")
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}
	if results[0].Status != doctorWarn {
		t.Errorf("first result should be Warn, got %v", results[0])
	}
	if !strings.Contains(results[0].Message, "unknown scope") {
		t.Errorf("first message should mention scope, got %q", results[0].Message)
	}
}

// TestCheckConfig_NoAuth must surface a Fail with a usable remedy when no
// API key is configured. We isolate via a temp HOME so the test never
// touches the developer's real config.
func TestCheckConfig_NoAuth(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Sanity check: LoadConfig should return empty.
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.APIKey != "" {
		t.Skip("temp HOME isolation failed; skipping")
	}

	results := checkConfig()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != doctorFail {
		t.Errorf("expected Fail when not configured, got %v", results[0])
	}
	if results[0].Remedy == "" {
		t.Error("Fail should carry a remedy command")
	}
}

// TestCheckHooks_NoClientsDetected returns a single OK with the
// "nothing to check" message when nothing is installed locally.
func TestCheckHooks_NoClientsDetected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	results := checkHooks()
	if len(results) != 1 {
		t.Fatalf("expected 1 fallback result, got %d", len(results))
	}
	if results[0].Status != doctorOK {
		t.Errorf("fallback should be OK, got %v", results[0])
	}
}

// TestCheckRules_NoClientsDetected mirrors the hooks fallback test.
func TestCheckRules_NoClientsDetected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Also redirect cwd so CursorInstaller doesn't see a `.cursor/` in this
	// repo. We can't os.Chdir easily without races, but the temp HOME alone
	// is enough for Windsurf; Cursor will fall through to its empty Detect
	// when the project root has no .cursor dir. The repo we run from might
	// have one — accept up to 2 results and assert none are Fail.
	results := checkRules()
	for _, r := range results {
		if r.Status == doctorFail {
			t.Errorf("rules check should not Fail in detection sweep, got %+v", r)
		}
	}
}
