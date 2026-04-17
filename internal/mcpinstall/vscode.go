package mcpinstall

import (
	"fmt"
	"os"
	"path/filepath"
)

// VS Code Copilot MCP integration uses `.vscode/mcp.json` at the workspace
// level with a DIFFERENT key ("servers" not "mcpServers") and per-server
// "type":"stdio". Project scope only.
type vscodeAdapter struct{}

func (vscodeAdapter) ID() string                { return "vscode" }
func (vscodeAdapter) Name() string              { return "VS Code (Copilot)" }
func (vscodeAdapter) SupportedScopes() []Scope  { return []Scope{ScopeProject} }

func (a vscodeAdapter) Detect() DetectionResult {
	cwd, err := os.Getwd()
	if err != nil {
		return DetectionResult{}
	}
	cfg := filepath.Join(cwd, ".vscode", "mcp.json")
	// Detection is looser for VS Code — any .vscode dir counts as "VS Code
	// workspace" even if mcp.json doesn't exist yet.
	if _, err := os.Stat(filepath.Join(cwd, ".vscode")); err == nil {
		return DetectionResult{Installed: true, ConfigPath: cfg, Detail: ".vscode/ found", ProjectOnly: true}
	}
	// No .vscode dir → user may still want to install; show as available.
	return DetectionResult{Installed: false, ConfigPath: cfg, Detail: "no .vscode/ in cwd", ProjectOnly: true}
}

func (a vscodeAdapter) ConfigPath(scope Scope) (string, error) {
	if scope != ScopeProject {
		return "", ErrScopeNotSupported{ClientID: a.ID(), Scope: scope}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".vscode", "mcp.json"), nil
}

// vscodeServerEntry emits VS Code's expected shape:
//   { "type": "stdio", "command": ..., "args": [...], "env": {...} }
func vscodeServerEntry(cmd string, args []string, env map[string]string) map[string]any {
	entry := map[string]any{
		"type":    "stdio",
		"command": cmd,
		"args":    args,
	}
	if len(env) > 0 {
		entry["env"] = env
	}
	return entry
}

// upsertVSCodeServer patches the "servers" key instead of "mcpServers".
func upsertVSCodeServer(raw map[string]any, name string, entry map[string]any) map[string]any {
	servers, _ := raw["servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[name] = entry
	raw["servers"] = servers
	return raw
}

func removeVSCodeServer(raw map[string]any, name string) map[string]any {
	servers, _ := raw["servers"].(map[string]any)
	if servers == nil {
		return raw
	}
	delete(servers, name)
	if len(servers) == 0 {
		delete(raw, "servers")
	} else {
		raw["servers"] = servers
	}
	return raw
}

func (a vscodeAdapter) Install(opts InstallOptions) (Diff, error) {
	path, err := a.ConfigPath(opts.Scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)
	after := upsertVSCodeServer(cloneMap(before), ServerName, vscodeServerEntry(opts.Command, opts.Args, opts.Env))
	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, fmt.Errorf("write %s: %w", path, err)
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a vscodeAdapter) Uninstall(scope Scope) (Diff, error) {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)
	after := removeVSCodeServer(cloneMap(before), ServerName)
	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, err
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a vscodeAdapter) IsInstalled(scope Scope) bool {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return false
	}
	raw, err := readJSONObject(path)
	if err != nil {
		return false
	}
	servers, _ := raw["servers"].(map[string]any)
	_, ok := servers[ServerName]
	return ok
}
