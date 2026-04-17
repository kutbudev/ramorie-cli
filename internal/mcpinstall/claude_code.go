package mcpinstall

import (
	"fmt"
	"os"
	"path/filepath"
)

// Claude Code stores MCP servers in ~/.claude.json under the "mcpServers"
// key (user-global). Project scope is driven by a .mcp.json at the repo
// root, using the same mcpServers shape.
type claudeCodeAdapter struct{}

func (claudeCodeAdapter) ID() string   { return "claude-code" }
func (claudeCodeAdapter) Name() string { return "Claude Code" }
func (claudeCodeAdapter) SupportedScopes() []Scope {
	return []Scope{ScopeUser, ScopeProject}
}

func (a claudeCodeAdapter) Detect() DetectionResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return DetectionResult{}
	}
	cfg := filepath.Join(home, ".claude.json")
	if _, err := os.Stat(cfg); err == nil {
		return DetectionResult{Installed: true, ConfigPath: cfg, Detail: "~/.claude.json found"}
	}
	// Fallback: .claude directory exists even if .claude.json doesn't (fresh
	// Claude Code install writes it lazily).
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		return DetectionResult{Installed: true, ConfigPath: cfg, Detail: "~/.claude/ dir present; will create ~/.claude.json"}
	}
	return DetectionResult{Installed: false, ConfigPath: cfg, Detail: "Claude Code not detected"}
}

func (a claudeCodeAdapter) ConfigPath(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude.json"), nil
	case ScopeProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".mcp.json"), nil
	}
	return "", ErrScopeNotSupported{ClientID: a.ID(), Scope: scope}
}

func (a claudeCodeAdapter) Install(opts InstallOptions) (Diff, error) {
	path, err := a.ConfigPath(opts.Scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)

	after := upsertMCPServer(cloneMap(before), ServerName, standardServerEntry(opts.Command, opts.Args, opts.Env))

	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, fmt.Errorf("write %s: %w", path, err)
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a claudeCodeAdapter) Uninstall(scope Scope) (Diff, error) {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)

	after := removeMCPServer(cloneMap(before), ServerName)
	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, err
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a claudeCodeAdapter) IsInstalled(scope Scope) bool {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return false
	}
	return hasMCPServerEntry(path, ServerName)
}

// cloneMap does a shallow clone so we can build `before` and `after` without
// aliasing — important for the Diff preview.
func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if inner, ok := v.(map[string]any); ok {
			out[k] = cloneMap(inner)
		} else {
			out[k] = v
		}
	}
	return out
}

func hasMCPServerEntry(path, name string) bool {
	raw, err := readJSONObject(path)
	if err != nil {
		return false
	}
	servers, _ := raw["mcpServers"].(map[string]any)
	_, ok := servers[name]
	return ok
}
