package mcpinstall

import (
	"os"
	"path/filepath"
)

// Zed stores MCP-style servers in ~/.config/zed/settings.json under the
// "context_servers" key, with a nested { "command": { "path": ..., "args": [...] } }
// shape rather than a flat command/args.
type zedAdapter struct{}

func (zedAdapter) ID() string                { return "zed" }
func (zedAdapter) Name() string              { return "Zed" }
func (zedAdapter) SupportedScopes() []Scope  { return []Scope{ScopeUser} }

func zedPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zed", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zed", "settings.json"), nil
}

func (a zedAdapter) Detect() DetectionResult {
	p, err := zedPath()
	if err != nil {
		return DetectionResult{}
	}
	if _, err := os.Stat(p); err == nil {
		return DetectionResult{Installed: true, ConfigPath: p, Detail: "settings.json found"}
	}
	dir := filepath.Dir(p)
	if _, err := os.Stat(dir); err == nil {
		return DetectionResult{Installed: true, ConfigPath: p, Detail: "zed dir present; file will be created"}
	}
	return DetectionResult{Installed: false, ConfigPath: p, Detail: "Zed not detected"}
}

func (a zedAdapter) ConfigPath(scope Scope) (string, error) {
	if scope != ScopeUser {
		return "", ErrScopeNotSupported{ClientID: a.ID(), Scope: scope}
	}
	return zedPath()
}

func zedServerEntry(cmd string, args []string, env map[string]string) map[string]any {
	command := map[string]any{
		"path": cmd,
		"args": args,
	}
	if len(env) > 0 {
		command["env"] = env
	}
	return map[string]any{"command": command}
}

func upsertZedServer(raw map[string]any, name string, entry map[string]any) map[string]any {
	servers, _ := raw["context_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[name] = entry
	raw["context_servers"] = servers
	return raw
}

func removeZedServer(raw map[string]any, name string) map[string]any {
	servers, _ := raw["context_servers"].(map[string]any)
	if servers == nil {
		return raw
	}
	delete(servers, name)
	if len(servers) == 0 {
		delete(raw, "context_servers")
	} else {
		raw["context_servers"] = servers
	}
	return raw
}

func (a zedAdapter) Install(opts InstallOptions) (Diff, error) {
	path, err := a.ConfigPath(opts.Scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)
	after := upsertZedServer(cloneMap(before), ServerName, zedServerEntry(opts.Command, opts.Args, opts.Env))
	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, err
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a zedAdapter) Uninstall(scope Scope) (Diff, error) {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)
	after := removeZedServer(cloneMap(before), ServerName)
	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, err
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a zedAdapter) IsInstalled(scope Scope) bool {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return false
	}
	raw, err := readJSONObject(path)
	if err != nil {
		return false
	}
	servers, _ := raw["context_servers"].(map[string]any)
	_, ok := servers[ServerName]
	return ok
}
