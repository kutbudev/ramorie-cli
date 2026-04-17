package mcpinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// adapterTestEnv isolates HOME and cwd so tests never touch the user's real
// ~/.claude.json, ~/.cursor/, etc. Call at the start of every test.
func adapterTestEnv(t *testing.T) (home string, cwd string) {
	t.Helper()
	home = t.TempDir()
	cwd = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config")) // Zed + Claude Desktop linux
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	origCwd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	return home, cwd
}

func readJSON(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal %s: %v — %s", p, err, b)
	}
	return out
}

// standardOpts returns an InstallOptions that every adapter accepts.
func standardOpts(scope Scope) InstallOptions {
	return InstallOptions{
		Scope:   scope,
		Command: "/usr/local/bin/ramorie",
		Args:    []string{"mcp", "serve"},
	}
}

// ---- Registry tests ---------------------------------------------------------

func TestRegistry_HasAllSixAdaptersInStableOrder(t *testing.T) {
	got := Registry()
	want := []string{"claude-code", "claude-desktop", "cursor", "windsurf", "vscode", "zed"}
	if len(got) != len(want) {
		t.Fatalf("registry size: got %d want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID() != id {
			t.Errorf("registry[%d]: got %q want %q — order matters for deterministic TUI", i, got[i].ID(), id)
		}
	}
}

// ---- Claude Code ------------------------------------------------------------

func TestClaudeCode_UserScope_RoundTripPreservesForeignConfig(t *testing.T) {
	home, _ := adapterTestEnv(t)
	// Seed ~/.claude.json with foreign top-level config we must preserve.
	cfg := filepath.Join(home, ".claude.json")
	_ = os.WriteFile(cfg, []byte(`{"theme":"dark","mcpServers":{"other":{"command":"other"}}}`), 0o644)

	a := &claudeCodeAdapter{}
	if a.IsInstalled(ScopeUser) {
		t.Fatal("pre-install state should not report installed")
	}

	diff, err := a.Install(standardOpts(ScopeUser))
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if diff.Path != cfg {
		t.Errorf("Diff.Path mismatch: got %q want %q", diff.Path, cfg)
	}
	if !a.IsInstalled(ScopeUser) {
		t.Fatal("IsInstalled should be true after Install")
	}

	got := readJSON(t, cfg)
	if got["theme"] != "dark" {
		t.Error("foreign top-level key `theme` must survive installs")
	}
	servers := got["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Error("foreign mcpServers entry must survive")
	}
	entry := servers["ramorie"].(map[string]any)
	if entry["command"] != "/usr/local/bin/ramorie" {
		t.Errorf("ramorie command not written: %v", entry)
	}
	args, _ := entry["args"].([]any)
	if len(args) != 2 || args[0] != "mcp" || args[1] != "serve" {
		t.Errorf("ramorie args not written correctly: %v", entry["args"])
	}

	// Re-install should be idempotent (no crash, still one entry).
	if _, err := a.Install(standardOpts(ScopeUser)); err != nil {
		t.Fatalf("re-install: %v", err)
	}
	got = readJSON(t, cfg)
	servers = got["mcpServers"].(map[string]any)
	if len(servers) != 2 {
		t.Errorf("re-install should keep 2 entries (other + ramorie), got %d", len(servers))
	}

	// Uninstall
	if _, err := a.Uninstall(ScopeUser); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if a.IsInstalled(ScopeUser) {
		t.Error("IsInstalled should be false after Uninstall")
	}
	got = readJSON(t, cfg)
	servers = got["mcpServers"].(map[string]any)
	if _, ramoriePresent := servers["ramorie"]; ramoriePresent {
		t.Error("ramorie entry should be gone after Uninstall")
	}
	if _, otherPresent := servers["other"]; !otherPresent {
		t.Error("foreign `other` server must survive Uninstall")
	}
	if got["theme"] != "dark" {
		t.Error("foreign theme must survive Uninstall")
	}
}

func TestClaudeCode_ProjectScope_WritesToCwdMcpJson(t *testing.T) {
	_, cwd := adapterTestEnv(t)
	a := &claudeCodeAdapter{}

	if _, err := a.Install(standardOpts(ScopeProject)); err != nil {
		t.Fatalf("Install project scope: %v", err)
	}
	expected := filepath.Join(cwd, ".mcp.json")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf(".mcp.json should have been created at cwd; stat: %v", err)
	}
	if !a.IsInstalled(ScopeProject) {
		t.Error("IsInstalled(project) should be true")
	}
	if a.IsInstalled(ScopeUser) {
		t.Error("installing project scope must NOT also flag user scope as installed")
	}
}

// ---- Cursor -----------------------------------------------------------------

func TestCursor_UserScope_CreatesCursorDirAndFile(t *testing.T) {
	home, _ := adapterTestEnv(t)
	a := &cursorAdapter{}
	if _, err := a.Install(standardOpts(ScopeUser)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cfg := filepath.Join(home, ".cursor", "mcp.json")
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("~/.cursor/mcp.json should exist: %v", err)
	}
	got := readJSON(t, cfg)
	if _, ok := got["mcpServers"].(map[string]any)["ramorie"]; !ok {
		t.Error("ramorie entry missing from cursor mcp.json")
	}
}

// ---- Claude Desktop -------------------------------------------------------

func TestClaudeDesktop_UserScope_PathIsPlatformCorrect(t *testing.T) {
	adapterTestEnv(t)
	a := &claudeDesktopAdapter{}
	if _, err := a.Install(standardOpts(ScopeUser)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	p, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("platform-resolved config path should exist after Install: %s err=%v", p, err)
	}
}

func TestClaudeDesktop_ProjectScopeUnsupported(t *testing.T) {
	adapterTestEnv(t)
	a := &claudeDesktopAdapter{}
	if _, err := a.ConfigPath(ScopeProject); err == nil {
		t.Error("Claude Desktop is user-only; project scope must error")
	}
	if _, err := a.Install(InstallOptions{Scope: ScopeProject, Command: "x"}); err == nil {
		t.Error("Install with unsupported scope must error, not silently succeed")
	}
}

// ---- Windsurf --------------------------------------------------------------

func TestWindsurf_UserScope_RoundTrip(t *testing.T) {
	home, _ := adapterTestEnv(t)
	a := &windsurfAdapter{}
	if _, err := a.Install(standardOpts(ScopeUser)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cfg := filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("windsurf config should exist: %v", err)
	}
	if !a.IsInstalled(ScopeUser) {
		t.Error("IsInstalled should report true after Install")
	}
	if _, err := a.Uninstall(ScopeUser); err != nil {
		t.Fatal(err)
	}
	if a.IsInstalled(ScopeUser) {
		t.Error("IsInstalled should be false after Uninstall")
	}
}

// ---- VS Code — uses `servers` key with {type:"stdio"} shape ----------------

func TestVSCode_ProjectScope_UsesServersKeyAndStdioType(t *testing.T) {
	_, cwd := adapterTestEnv(t)
	a := &vscodeAdapter{}
	if _, err := a.Install(standardOpts(ScopeProject)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cfg := filepath.Join(cwd, ".vscode", "mcp.json")
	got := readJSON(t, cfg)
	// VS Code expects "servers" NOT "mcpServers".
	if _, wrongKey := got["mcpServers"]; wrongKey {
		t.Error("VS Code config must use `servers` — `mcpServers` would be ignored")
	}
	servers, ok := got["servers"].(map[string]any)
	if !ok {
		t.Fatalf("servers key missing: %v", got)
	}
	entry, ok := servers["ramorie"].(map[string]any)
	if !ok {
		t.Fatalf("ramorie entry missing: %v", servers)
	}
	if entry["type"] != "stdio" {
		t.Errorf("VS Code entry MUST include type:stdio — got %v", entry["type"])
	}
	if entry["command"] != "/usr/local/bin/ramorie" {
		t.Errorf("command wrong: %v", entry["command"])
	}
}

func TestVSCode_UserScopeUnsupported(t *testing.T) {
	adapterTestEnv(t)
	a := &vscodeAdapter{}
	if _, err := a.ConfigPath(ScopeUser); err == nil {
		t.Error("VS Code is project-only in our impl; user scope must error")
	}
}

// ---- Zed — uses nested `command.path` / `command.args` shape ---------------

func TestZed_UserScope_UsesContextServersKeyAndNestedCommand(t *testing.T) {
	home, _ := adapterTestEnv(t)
	a := &zedAdapter{}
	if _, err := a.Install(standardOpts(ScopeUser)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cfg := filepath.Join(home, ".config", "zed", "settings.json")
	got := readJSON(t, cfg)
	if _, wrongKey := got["mcpServers"]; wrongKey {
		t.Error("Zed config must use context_servers — mcpServers would be ignored")
	}
	servers, _ := got["context_servers"].(map[string]any)
	entry, ok := servers["ramorie"].(map[string]any)
	if !ok {
		t.Fatalf("ramorie entry missing from context_servers: %v", servers)
	}
	cmd, ok := entry["command"].(map[string]any)
	if !ok {
		t.Fatalf("Zed entry must nest command as an object (path+args), got: %v", entry["command"])
	}
	if cmd["path"] != "/usr/local/bin/ramorie" {
		t.Errorf("command.path wrong: %v", cmd["path"])
	}
	args, _ := cmd["args"].([]any)
	if len(args) != 2 {
		t.Errorf("command.args should be 2 elements, got %v", args)
	}
}

// ---- Preview predictor -----------------------------------------------------

func TestPredictAfter_MatchesActualInstallAcrossClientShapes(t *testing.T) {
	// For each adapter, predictAfter must produce the same JSON as a real
	// Install. This test guards against the TUI preview drifting from what
	// actually gets written.
	_, cwd := adapterTestEnv(t)
	_ = cwd
	for _, a := range Registry() {
		t.Run(a.ID(), func(t *testing.T) {
			adapterTestEnv(t) // fresh tmp HOME per subtest
			scope := a.SupportedScopes()[0]
			before := map[string]any{}
			// Predicted.
			predicted := predictAfter(a.ID(), cloneMap(before), "/usr/local/bin/ramorie", []string{"mcp", "serve"})
			// Actual.
			if _, err := a.Install(standardOpts(scope)); err != nil {
				t.Fatalf("Install: %v", err)
			}
			path, _ := a.ConfigPath(scope)
			actual := readJSON(t, path)

			predJSON, _ := json.Marshal(predicted)
			actJSON, _ := json.Marshal(actual)
			if string(predJSON) != string(actJSON) {
				t.Errorf("predictAfter drifted from actual Install output:\npredict=%s\nactual =%s", predJSON, actJSON)
			}
		})
	}
}

// ---- Patch helpers ---------------------------------------------------------

func TestReadJSONObject_ReturnsEmptyForMissingFile(t *testing.T) {
	raw, err := readJSONObject(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(raw) != 0 {
		t.Errorf("missing file should yield empty map, got %v", raw)
	}
}

func TestReadJSONObject_ErrorsOnMalformedJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(p, []byte("{ not json"), 0o644)
	if _, err := readJSONObject(p); err == nil {
		t.Fatal("malformed JSON must surface as error — refuse to overwrite")
	}
}

func TestCloneMap_NestedIndependence(t *testing.T) {
	src := map[string]any{
		"top": "a",
		"nested": map[string]any{
			"inner": "b",
		},
	}
	cp := cloneMap(src)
	cp["top"] = "changed"
	cp["nested"].(map[string]any)["inner"] = "changed"
	if src["top"] != "a" {
		t.Error("cloneMap must not alias top-level values")
	}
	if src["nested"].(map[string]any)["inner"] != "b" {
		t.Error("cloneMap must recursively clone nested maps to protect originals")
	}
}

func TestRemoveMCPServer_DropsEmptyParent(t *testing.T) {
	raw := map[string]any{
		"other": "keep",
		"mcpServers": map[string]any{
			"ramorie": map[string]any{"command": "x"},
		},
	}
	raw = removeMCPServer(raw, "ramorie")
	if _, ok := raw["mcpServers"]; ok {
		t.Error("empty mcpServers map should be dropped to keep config clean")
	}
	if raw["other"] != "keep" {
		t.Error("sibling keys must survive removal")
	}
}

func TestRemoveMCPServer_KeepsNonEmptyParent(t *testing.T) {
	raw := map[string]any{
		"mcpServers": map[string]any{
			"ramorie": map[string]any{"command": "x"},
			"other":   map[string]any{"command": "y"},
		},
	}
	raw = removeMCPServer(raw, "ramorie")
	servers, ok := raw["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers should remain when other servers exist")
	}
	if _, present := servers["ramorie"]; present {
		t.Error("ramorie must be removed")
	}
	if _, present := servers["other"]; !present {
		t.Error("other server must survive")
	}
}
