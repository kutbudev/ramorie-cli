package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestPruneHookEntries_RemovesOwnLeavesForeignAlone ensures uninstall never
// clobbers hooks installed by other tools that happen to share the matcher.
func TestPruneHookEntries_RemovesOwnLeavesForeignAlone(t *testing.T) {
	before := []interface{}{
		map[string]interface{}{
			"matcher": "Edit",
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": "ramorie hook context", "id": hookIdentifier},
			},
		},
		map[string]interface{}{
			"matcher": "Edit",
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": "foreign-tool run"},
			},
		},
		map[string]interface{}{
			"matcher": "Write",
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": "ramorie hook context", "id": hookIdentifier},
				map[string]interface{}{"type": "command", "command": "other"},
			},
		},
	}

	after := pruneHookEntries(before, hookIdentifier)

	if len(after) != 2 {
		t.Fatalf("expected 2 remaining groups, got %d", len(after))
	}

	// First surviving group must be the foreign Edit entry.
	first, _ := after[0].(map[string]interface{})
	if first["matcher"] != "Edit" {
		t.Errorf("expected foreign Edit group, got %v", first)
	}
	if inner, _ := first["hooks"].([]interface{}); len(inner) != 1 {
		t.Errorf("foreign Edit should keep exactly 1 hook, got %d", len(inner))
	}

	// Second group (Write) should have its ramorie entry dropped but keep the other.
	second, _ := after[1].(map[string]interface{})
	inner, _ := second["hooks"].([]interface{})
	if len(inner) != 1 {
		t.Errorf("Write group should have 1 hook after prune, got %d", len(inner))
	}
	if h, _ := inner[0].(map[string]interface{}); h["command"] != "other" {
		t.Errorf("surviving hook should be the foreign one, got %v", h)
	}
}

func TestPruneHookEntries_EmptyInput(t *testing.T) {
	out := pruneHookEntries(nil, hookIdentifier)
	if len(out) != 0 {
		t.Fatalf("nil input should produce empty slice, got %v", out)
	}
}

func TestPruneHookEntries_GroupBecomesEmptyIsDropped(t *testing.T) {
	before := []interface{}{
		map[string]interface{}{
			"matcher": "Edit",
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "id": hookIdentifier},
			},
		},
	}
	after := pruneHookEntries(before, hookIdentifier)
	if len(after) != 0 {
		t.Fatalf("group emptied by prune should be dropped entirely, got %d", len(after))
	}
}

func TestExtractFilePathFromPayload_ToolInputShape(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
		want    string
	}{
		{
			name: "file_path in tool_input",
			payload: map[string]interface{}{
				"tool_name":  "Edit",
				"tool_input": map[string]interface{}{"file_path": "/abs/foo.go"},
			},
			want: "/abs/foo.go",
		},
		{
			name: "path as alternate field",
			payload: map[string]interface{}{
				"tool_input": map[string]interface{}{"path": "/abs/bar.ts"},
			},
			want: "/abs/bar.ts",
		},
		{
			name:    "missing tool_input → empty",
			payload: map[string]interface{}{"tool_name": "Edit"},
			want:    "",
		},
		{
			name: "no file_path or path fields",
			payload: map[string]interface{}{
				"tool_input": map[string]interface{}{"unrelated": 1},
			},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractFilePathFromPayload(tc.payload)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestInstallUninstall_Roundtrip runs the real install/uninstall paths
// against a temp HOME so we don't touch the user's ~/.claude/settings.json.
// Verifies settings shape preservation — foreign top-level keys must survive.
func TestInstallUninstall_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	settingsPath := filepath.Join(tmp, ".claude", "settings.json")
	_ = os.MkdirAll(filepath.Dir(settingsPath), 0o755)

	// Seed with foreign config we must preserve end-to-end.
	foreign := map[string]interface{}{
		"theme": "dark",
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Edit",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "foreign-tool"},
					},
				},
			},
		},
	}
	foreignJSON, _ := json.Marshal(foreign)
	if err := os.WriteFile(settingsPath, foreignJSON, 0o644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	// Install
	if err := hookInstall(nil); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	raw, err := loadSettings(settingsPath)
	if err != nil {
		t.Fatalf("post-install load: %v", err)
	}
	if raw["theme"] != "dark" {
		t.Error("foreign top-level key `theme` must be preserved")
	}
	preUse, _ := raw["hooks"].(map[string]interface{})["PreToolUse"].([]interface{})
	if len(preUse) != 2 {
		t.Fatalf("after install expected 2 PreToolUse groups (foreign + ramorie), got %d", len(preUse))
	}

	// Install again → must stay at 2 (idempotent, not duplicated).
	if err := hookInstall(nil); err != nil {
		t.Fatalf("re-install failed: %v", err)
	}
	raw2, _ := loadSettings(settingsPath)
	preUse2, _ := raw2["hooks"].(map[string]interface{})["PreToolUse"].([]interface{})
	if len(preUse2) != 2 {
		t.Fatalf("re-install must be idempotent; got %d groups", len(preUse2))
	}

	// Uninstall
	if err := hookUninstall(nil); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}
	raw3, _ := loadSettings(settingsPath)
	if raw3["theme"] != "dark" {
		t.Error("uninstall must not clobber foreign config")
	}
	hooksOut, ok := raw3["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("foreign hooks map should still exist after uninstall")
	}
	surviving, _ := hooksOut["PreToolUse"].([]interface{})
	if len(surviving) != 1 {
		t.Fatalf("foreign group must survive; got %d remaining", len(surviving))
	}
	// The survivor must be the foreign one, not ours.
	group, _ := surviving[0].(map[string]interface{})
	inner, _ := group["hooks"].([]interface{})
	h, _ := inner[0].(map[string]interface{})
	if h["command"] != "foreign-tool" {
		t.Errorf("survivor should be foreign-tool hook, got %v", h)
	}
}

func TestLoadSettings_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.json")
	raw, err := loadSettings(path)
	if err != nil {
		t.Fatalf("missing file should not error (returns empty), got %v", err)
	}
	if !reflect.DeepEqual(raw, map[string]interface{}{}) {
		t.Fatalf("missing file should produce empty map, got %+v", raw)
	}
}

func TestLoadSettings_InvalidJSONErrors(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "broken.json")
	if err := os.WriteFile(path, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSettings(path); err == nil {
		t.Fatal("malformed JSON must surface an error so we never overwrite user config silently")
	}
}
