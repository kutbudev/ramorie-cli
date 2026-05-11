package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestClaudeCodeInstaller_Idempotent asserts that running Install three
// times back-to-back produces byte-identical settings files. Idempotency is
// the property we rely on most: `ramorie init` is expected to be safe to
// re-run after upgrades, and a non-idempotent installer would gradually
// duplicate hook entries on every invocation.
func TestClaudeCodeInstaller_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	inst := NewClaudeCodeInstallerAt(path)

	entries := DefaultEntries()

	for i := 0; i < 3; i++ {
		if err := inst.Install(entries); err != nil {
			t.Fatalf("install round %d failed: %v", i, err)
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	// Re-install one more time into a fresh file and diff: must match.
	other := filepath.Join(dir, "other.json")
	if err := NewClaudeCodeInstallerAt(other).Install(entries); err != nil {
		t.Fatalf("install other: %v", err)
	}
	wanted, err := os.ReadFile(other)
	if err != nil {
		t.Fatalf("read other: %v", err)
	}
	if string(got) != string(wanted) {
		t.Fatalf("idempotent settings diverged:\n--- single install ---\n%s\n--- triple install ---\n%s", wanted, got)
	}

	// Sanity: settings must contain exactly len(entries) Ramorie hook IDs.
	var raw map[string]interface{}
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	count := countRamorieIDs(raw)
	if count != len(entries) {
		t.Fatalf("expected %d ramorie IDs after 3 installs, got %d", len(entries), count)
	}
}

// TestClaudeCodeInstaller_UninstallRemovesOnlyRamorieEntries verifies that
// uninstall scrubs ramorie-* IDs but leaves foreign hooks alone — this is
// the contract that lets users `hook uninstall` without breaking their own
// integrations.
func TestClaudeCodeInstaller_UninstallRemovesOnlyRamorieEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	// Seed with a foreign hook the user installed manually.
	foreign := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Agent",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "my-team-tool log",
							"id":      "team-custom-tool",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(foreign, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	inst := NewClaudeCodeInstallerAt(path)
	entries := DefaultEntries()

	if err := inst.Install(entries); err != nil {
		t.Fatalf("install: %v", err)
	}

	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	if err := inst.Uninstall(ids); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	raw, err := loadJSONFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if countRamorieIDs(raw) != 0 {
		t.Fatalf("expected zero ramorie IDs after uninstall, got %d", countRamorieIDs(raw))
	}

	// Foreign entry must still be there.
	if !containsID(raw, "team-custom-tool") {
		t.Fatalf("foreign hook 'team-custom-tool' got nuked by uninstall — foreign entries must survive")
	}
}

// TestClaudeCodeInstaller_Status reads back installed entries and confirms
// the round-trip captures event + matcher + ID.
func TestClaudeCodeInstaller_Status(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	inst := NewClaudeCodeInstallerAt(path)
	entries := DefaultEntries()
	if err := inst.Install(entries); err != nil {
		t.Fatal(err)
	}
	got, err := inst.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(entries) {
		t.Fatalf("Status returned %d entries, want %d", len(got), len(entries))
	}
	// Spot-check the Agent matcher survived the round-trip.
	foundAgent := false
	for _, e := range got {
		if e.Matcher == "Agent" && e.Event == PostToolUse {
			foundAgent = true
		}
	}
	if !foundAgent {
		t.Errorf("Status missed PostToolUse:Agent entry")
	}
}

// TestCodexInstaller_Detect ensures Codex detection keys off config.toml
// (not the hooks.json we write).
func TestCodexInstaller_Detect(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	hooksPath := filepath.Join(dir, "hooks.json")

	inst := NewCodexInstallerAt(hooksPath, configPath)
	if inst.Detect() {
		t.Fatal("Detect should be false when config.toml is missing")
	}
	if err := os.WriteFile(configPath, []byte("# codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !inst.Detect() {
		t.Fatal("Detect should be true after config.toml is created")
	}
}

// TestCodexInstaller_InstallWritesSeparateFile makes sure the Codex
// installer writes to hooks.json — never to the user's config.toml.
func TestCodexInstaller_InstallWritesSeparateFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	hooksPath := filepath.Join(dir, "hooks.json")
	if err := os.WriteFile(configPath, []byte("# codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := NewCodexInstallerAt(hooksPath, configPath)
	if err := inst.Install(DefaultEntries()); err != nil {
		t.Fatalf("install: %v", err)
	}

	if _, err := os.Stat(hooksPath); err != nil {
		t.Fatalf("hooks.json not created: %v", err)
	}

	// config.toml content must be unchanged.
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# codex\n" {
		t.Fatalf("config.toml was modified, want untouched. got: %q", string(got))
	}
}

// countRamorieIDs walks a settings map and counts ramorie-* hook IDs.
func countRamorieIDs(raw map[string]interface{}) int {
	hooks, _ := raw["hooks"].(map[string]interface{})
	count := 0
	for _, groupsRaw := range hooks {
		groups, _ := groupsRaw.([]interface{})
		for _, g := range groups {
			gm, ok := g.(map[string]interface{})
			if !ok {
				continue
			}
			inner, _ := gm["hooks"].([]interface{})
			for _, h := range inner {
				hm, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				if id, _ := hm["id"].(string); IsRamorieID(id) {
					count++
				}
			}
		}
	}
	return count
}

// containsID reports whether any hook in the settings map carries `want`.
func containsID(raw map[string]interface{}, want string) bool {
	hooks, _ := raw["hooks"].(map[string]interface{})
	for _, groupsRaw := range hooks {
		groups, _ := groupsRaw.([]interface{})
		for _, g := range groups {
			gm, ok := g.(map[string]interface{})
			if !ok {
				continue
			}
			inner, _ := gm["hooks"].([]interface{})
			for _, h := range inner {
				hm, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				if id, _ := hm["id"].(string); id == want {
					return true
				}
			}
		}
	}
	return false
}
