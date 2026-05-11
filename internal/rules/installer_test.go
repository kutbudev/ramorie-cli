package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kutbudev/ramorie-cli/internal/protocol"
)

// TestUpsertManagedBlock_AppendsWhenAbsent verifies a clean append on a
// file that has no marker pair yet — the block should be separated from
// existing content by exactly one blank line.
func TestUpsertManagedBlock_AppendsWhenAbsent(t *testing.T) {
	existing := "# My Global Rules\n\nUse Tailwind everywhere.\n"
	got := upsertManagedBlock(existing, "RAMORIE PROTOCOL BODY")
	if !strings.Contains(got, MarkerStart) {
		t.Fatal("expected start marker present")
	}
	if !strings.Contains(got, MarkerEnd) {
		t.Fatal("expected end marker present")
	}
	if !strings.Contains(got, "Use Tailwind everywhere.") {
		t.Fatal("original content was clobbered")
	}
	// Single, not duplicated.
	if strings.Count(got, MarkerStart) != 1 {
		t.Fatalf("expected exactly one start marker, got %d", strings.Count(got, MarkerStart))
	}
}

// TestUpsertManagedBlock_ReplacesWhenPresent is the idempotency check —
// three rounds of upsert must converge on identical content.
func TestUpsertManagedBlock_ReplacesWhenPresent(t *testing.T) {
	existing := "# Notes\n"
	out := existing
	for i := 0; i < 3; i++ {
		out = upsertManagedBlock(out, "BODY v1")
	}
	if strings.Count(out, MarkerStart) != 1 {
		t.Fatalf("3x upsert produced %d start markers, want 1", strings.Count(out, MarkerStart))
	}
	if strings.Count(out, "BODY v1") != 1 {
		t.Fatalf("expected body to appear once, got %d", strings.Count(out, "BODY v1"))
	}
}

// TestUpsertManagedBlock_BodyUpdates makes sure that when the body text
// changes between calls, the new body wins and the old one is discarded.
func TestUpsertManagedBlock_BodyUpdates(t *testing.T) {
	out := upsertManagedBlock("", "OLD")
	out = upsertManagedBlock(out, "NEW")
	if strings.Contains(out, "OLD") {
		t.Fatal("stale body 'OLD' survived an upsert")
	}
	if !strings.Contains(out, "NEW") {
		t.Fatal("fresh body 'NEW' missing after upsert")
	}
}

// TestRemoveManagedBlock_LeavesForeignContentIntact ensures Uninstall is
// non-destructive against user-authored text.
func TestRemoveManagedBlock_LeavesForeignContentIntact(t *testing.T) {
	original := "# Header\n\nKeep me.\n"
	withBlock := upsertManagedBlock(original, "protocol body")
	stripped := removeManagedBlock(withBlock)
	if !strings.Contains(stripped, "Keep me.") {
		t.Fatal("foreign content was removed alongside managed block")
	}
	if strings.Contains(stripped, MarkerStart) {
		t.Fatal("managed block not removed")
	}
}

// TestCursorInstaller_RoundTrip exercises Install → Status → Uninstall in
// a temp project root. Frontmatter must carry the live protocol.Version.
func TestCursorInstaller_RoundTrip(t *testing.T) {
	root := t.TempDir()
	inst := NewCursorInstallerAt(root)

	if err := inst.Install(protocol.SessionStartText); err != nil {
		t.Fatalf("install: %v", err)
	}

	installed, version, err := inst.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Fatal("Status reports not installed right after Install")
	}
	if version != protocol.Version {
		t.Fatalf("version mismatch: got %q want %q", version, protocol.Version)
	}

	// File should be at the expected path with the frontmatter intact.
	expectedPath := filepath.Join(root, ".cursor", "rules", "ramorie-memory-protocol.mdc")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "---\n") {
		t.Fatal("missing frontmatter opener")
	}
	if !strings.Contains(string(data), "alwaysApply: true") {
		t.Fatal("missing alwaysApply: true")
	}

	if err := inst.Uninstall(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Fatalf("file should be gone after uninstall, stat err = %v", err)
	}
}

// TestCursorInstaller_InstallIsIdempotent — 3 installs, same bytes.
func TestCursorInstaller_InstallIsIdempotent(t *testing.T) {
	root := t.TempDir()
	inst := NewCursorInstallerAt(root)
	path := inst.RulesPath()

	for i := 0; i < 3; i++ {
		if err := inst.Install(protocol.SessionStartText); err != nil {
			t.Fatalf("install round %d: %v", i, err)
		}
	}
	got, _ := os.ReadFile(path)

	// Compare to a fresh single install in another dir.
	other := t.TempDir()
	otherInst := NewCursorInstallerAt(other)
	if err := otherInst.Install(protocol.SessionStartText); err != nil {
		t.Fatal(err)
	}
	wanted, _ := os.ReadFile(otherInst.RulesPath())

	if string(got) != string(wanted) {
		t.Fatalf("not idempotent:\n--- 1x ---\n%s\n--- 3x ---\n%s", wanted, got)
	}
}

// TestWindsurfInstaller_RoundTrip runs Install → Status → Uninstall against
// a temp global_rules.md file that already has user content.
func TestWindsurfInstaller_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "global_rules.md")
	if err := os.WriteFile(path, []byte("# Foreign Header\n\nUser notes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := NewWindsurfInstallerAt(path)
	if err := inst.Install(protocol.SessionStartText); err != nil {
		t.Fatalf("install: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Foreign Header") {
		t.Fatal("user content was clobbered")
	}
	if !strings.Contains(string(data), MarkerStart) {
		t.Fatal("managed block not injected")
	}

	installed, version, err := inst.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !installed || version != "1" {
		t.Fatalf("status mismatch: installed=%v version=%q", installed, version)
	}

	if err := inst.Uninstall(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), MarkerStart) {
		t.Fatal("managed block survived uninstall")
	}
	if !strings.Contains(string(data), "Foreign Header") {
		t.Fatal("user content was removed during uninstall")
	}
}

// TestWindsurfInstaller_IdempotentThreeRounds — re-running install three
// times must NOT duplicate the managed block.
func TestWindsurfInstaller_IdempotentThreeRounds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "global_rules.md")
	inst := NewWindsurfInstallerAt(path)

	for i := 0; i < 3; i++ {
		if err := inst.Install(protocol.SessionStartText); err != nil {
			t.Fatalf("install round %d: %v", i, err)
		}
	}
	data, _ := os.ReadFile(path)
	if got := strings.Count(string(data), MarkerStart); got != 1 {
		t.Fatalf("3x install produced %d managed blocks, want 1", got)
	}
}

// TestExtractCursorVersion_ParsesFrontmatter sanity-checks the helper that
// powers Status() on the cursor installer.
func TestExtractCursorVersion_ParsesFrontmatter(t *testing.T) {
	body := "---\nalwaysApply: true\nramorie_version: 1.0\n---\nrest"
	if got := extractCursorVersion(body); got != "1.0" {
		t.Fatalf("extractCursorVersion = %q, want 1.0", got)
	}
	if got := extractCursorVersion("no frontmatter here"); got != "" {
		t.Fatalf("expected empty version on plain file, got %q", got)
	}
}
