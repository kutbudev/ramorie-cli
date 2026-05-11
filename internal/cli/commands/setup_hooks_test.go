package commands

import (
	"testing"
)

// TestFilterHookInstallers_AllReturnsEverything ensures the default ("all"
// or "") path doesn't drop any known integration. If a new hook installer
// is added without updating allHookInstallers this test will silently pass,
// which is acceptable — it guards the negative case (filter returns 0).
func TestFilterHookInstallers_AllReturnsEverything(t *testing.T) {
	for _, val := range []string{"", "all", "ALL"} {
		got := filterHookInstallers(val)
		if len(got) != len(allHookInstallers()) {
			t.Errorf("filter %q dropped installers: got %d, want %d",
				val, len(got), len(allHookInstallers()))
		}
	}
}

// TestFilterHookInstallers_SpecificClient narrows to a single integration.
// Used by `--client claude-code` to avoid touching Codex.
func TestFilterHookInstallers_SpecificClient(t *testing.T) {
	got := filterHookInstallers("claude-code")
	if len(got) != 1 {
		t.Fatalf("claude-code filter: got %d, want 1", len(got))
	}
	if got[0].Name() != "claude-code" {
		t.Errorf("wrong installer returned: %q", got[0].Name())
	}
}

// TestFilterHookInstallers_UnknownClientReturnsEmpty so the caller can
// distinguish "no clients selected" from "all clients selected".
func TestFilterHookInstallers_UnknownClientReturnsEmpty(t *testing.T) {
	got := filterHookInstallers("not-a-real-editor")
	if len(got) != 0 {
		t.Errorf("unknown client should return empty, got %d", len(got))
	}
}

// TestFilterRulesInstallers_All mirrors the hooks-side test.
func TestFilterRulesInstallers_All(t *testing.T) {
	got := filterRulesInstallers("all")
	if len(got) != len(allRulesInstallers()) {
		t.Errorf("rules all: got %d, want %d", len(got), len(allRulesInstallers()))
	}
}

func TestFilterRulesInstallers_SpecificClient(t *testing.T) {
	got := filterRulesInstallers("windsurf")
	if len(got) != 1 || got[0].Name() != "windsurf" {
		t.Fatalf("windsurf filter wrong: %+v", got)
	}
}
