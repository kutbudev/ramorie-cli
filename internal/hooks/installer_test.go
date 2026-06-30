package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestDefaultEntries_SessionStartUsesDynamicHookCommand(t *testing.T) {
	entries := DefaultEntries()
	if len(entries) == 0 {
		t.Fatal("DefaultEntries returned no hooks")
	}
	session := entries[0]
	if session.Event != SessionStart {
		t.Fatalf("first hook event = %s, want %s", session.Event, SessionStart)
	}
	if !strings.Contains(session.Command, " hook session-start") {
		t.Fatalf("SessionStart command must call the dynamic hook shim, got %q", session.Command)
	}
	if strings.Contains(session.Command, "cat <<") {
		t.Fatalf("SessionStart command must not be a static heredoc anymore: %q", session.Command)
	}
}

func TestDefaultEntries_IncludeBeforeActionRunbookHook(t *testing.T) {
	entries := DefaultEntries()
	var found bool
	for _, entry := range entries {
		if entry.ID != "ramorie-protocol-before-action-v1" {
			continue
		}
		found = true
		if entry.Event != PreToolUse {
			t.Fatalf("before-action event = %s, want %s", entry.Event, PreToolUse)
		}
		if entry.Matcher != "Bash|Shell" {
			t.Fatalf("before-action matcher = %q, want Bash|Shell", entry.Matcher)
		}
		if !strings.Contains(entry.Command, " hook before-action") {
			t.Fatalf("before-action command must call hook shim, got %q", entry.Command)
		}
	}
	if !found {
		t.Fatalf("DefaultEntries missing before-action runbook hook")
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
	// Spot-check the Task matcher survived the round-trip. Claude Code
	// dispatches sub-agents via the Task tool, so PostToolUse must scope to
	// "Task" (not the old, never-firing "Agent" matcher).
	foundTask := false
	for _, e := range got {
		if e.Matcher == "Task" && e.Event == PostToolUse {
			foundTask = true
		}
	}
	if !foundTask {
		t.Errorf("Status missed PostToolUse:Task entry")
	}
}

// TestDefaultEntries_PostToolUseMatchesTaskTool guards the matcher fix: the
// post-subagent reminder must scope to the Task tool. Claude Code has no tool
// literally named "Agent", so the old matcher meant the hook never fired.
func TestDefaultEntries_PostToolUseMatchesTaskTool(t *testing.T) {
	for _, e := range DefaultEntries() {
		if e.ID != "ramorie-protocol-post-agent-v1" {
			continue
		}
		if e.Event != PostToolUse {
			t.Fatalf("post-agent event = %s, want %s", e.Event, PostToolUse)
		}
		if e.Matcher != "Task" {
			t.Fatalf("post-agent matcher = %q, want Task", e.Matcher)
		}
		return
	}
	t.Fatal("DefaultEntries missing post-agent reminder hook")
}

// TestDefaultEntries_IncludeFileContextHook locks the Edit|Write|Read per-file
// context hook into the canonical installer set (it previously shipped only via
// the deprecated `ramorie hook install` path).
func TestDefaultEntries_IncludeFileContextHook(t *testing.T) {
	for _, e := range DefaultEntries() {
		if e.ID != "ramorie-protocol-file-context-v1" {
			continue
		}
		if e.Event != PreToolUse {
			t.Fatalf("file-context event = %s, want %s", e.Event, PreToolUse)
		}
		if e.Matcher != "Edit|Write|Read" {
			t.Fatalf("file-context matcher = %q, want Edit|Write|Read", e.Matcher)
		}
		if !strings.Contains(e.Command, " hook context") {
			t.Fatalf("file-context command must call the context shim, got %q", e.Command)
		}
		return
	}
	t.Fatal("DefaultEntries missing Edit|Write|Read file-context hook")
}

// TestDefaultEntries_SessionStartRequestsFullPayload verifies SessionStart asks
// for --full so recent_memories / in_progress_tasks / last_session surface at
// boot rather than staying behind the compact path.
func TestDefaultEntries_SessionStartRequestsFullPayload(t *testing.T) {
	cmd := DefaultEntries()[0].Command
	if !strings.Contains(cmd, " hook session-start") {
		t.Fatalf("first entry is not session-start: %q", cmd)
	}
	if !strings.Contains(cmd, "--full") {
		t.Fatalf("session-start command must pass --full, got %q", cmd)
	}
}

// TestDefaultEntries_IncludePromptSubmitHook locks the UserPromptSubmit hook
// (T1.4) into the canonical installer set: each user prompt is used to inject
// prompt-relevant memories. It is a non-tool event, so it must carry no matcher.
func TestDefaultEntries_IncludePromptSubmitHook(t *testing.T) {
	for _, e := range DefaultEntries() {
		if e.ID != "ramorie-protocol-prompt-submit-v1" {
			continue
		}
		if e.Event != UserPromptSubmit {
			t.Fatalf("prompt-submit event = %s, want %s", e.Event, UserPromptSubmit)
		}
		if e.Matcher != "" {
			t.Fatalf("prompt-submit must have no matcher (not a tool event), got %q", e.Matcher)
		}
		if !strings.Contains(e.Command, " hook prompt-submit") {
			t.Fatalf("prompt-submit command must call the shim, got %q", e.Command)
		}
		return
	}
	t.Fatal("DefaultEntries missing UserPromptSubmit prompt-submit hook")
}

// TestDiffEntries_BinaryPathMoveNotStale verifies DiffEntries treats two hook
// commands that differ only by embedded binary path as identical — a moved
// binary must not flag every entry as stale.
func TestDiffEntries_BinaryPathMoveNotStale(t *testing.T) {
	expected := []HookEntry{
		{Event: SessionStart, Command: "'/usr/local/bin/ramorie' hook session-start --full", ID: "ramorie-protocol-session-start-v1"},
	}
	installed := []HookEntry{
		{Event: SessionStart, Command: "'/opt/homebrew/bin/ramorie' hook session-start --full", ID: "ramorie-protocol-session-start-v1"},
	}
	missing, stale := DiffEntries(expected, installed)
	if len(missing) != 0 {
		t.Fatalf("expected no missing, got %d", len(missing))
	}
	if len(stale) != 0 {
		t.Fatalf("binary path move must not be stale, got %d stale", len(stale))
	}

	// But a real argument change (missing --full) MUST still be stale.
	installedOld := []HookEntry{
		{Event: SessionStart, Command: "'/opt/homebrew/bin/ramorie' hook session-start", ID: "ramorie-protocol-session-start-v1"},
	}
	if _, stale := DiffEntries(expected, installedOld); len(stale) != 1 {
		t.Fatalf("argument change must be stale, got %d", len(stale))
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
