package mcpinstall

import (
	"os"
	"path/filepath"
)

// Cursor uses ~/.cursor/mcp.json (user-global) and .cursor/mcp.json
// (project-local). Same mcpServers shape as Claude Desktop / Windsurf.
type cursorAdapter struct{}

func (cursorAdapter) ID() string   { return "cursor" }
func (cursorAdapter) Name() string { return "Cursor" }
func (cursorAdapter) SupportedScopes() []Scope {
	return []Scope{ScopeUser, ScopeProject}
}

func (a cursorAdapter) Detect() DetectionResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return DetectionResult{}
	}
	cursorDir := filepath.Join(home, ".cursor")
	mcpFile := filepath.Join(cursorDir, "mcp.json")

	if _, err := os.Stat(mcpFile); err == nil {
		return DetectionResult{Installed: true, ConfigPath: mcpFile, Detail: "mcp.json found"}
	}
	if _, err := os.Stat(cursorDir); err == nil {
		return DetectionResult{Installed: true, ConfigPath: mcpFile, Detail: "~/.cursor/ present; file will be created"}
	}
	return DetectionResult{Installed: false, ConfigPath: mcpFile, Detail: "Cursor not detected"}
}

func (a cursorAdapter) ConfigPath(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".cursor", "mcp.json"), nil
	case ScopeProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".cursor", "mcp.json"), nil
	}
	return "", ErrScopeNotSupported{ClientID: a.ID(), Scope: scope}
}

func (a cursorAdapter) Install(opts InstallOptions) (Diff, error) {
	return stdInstall(a, opts, standardServerEntry(opts.Command, opts.Args, opts.Env))
}

func (a cursorAdapter) Uninstall(scope Scope) (Diff, error) {
	return stdUninstall(a, scope)
}

func (a cursorAdapter) IsInstalled(scope Scope) bool {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return false
	}
	return hasMCPServerEntry(path, ServerName)
}

// ---- shared install/uninstall for `mcpServers`-flavoured adapters ---------

// stdInstall/stdUninstall factor out the boilerplate shared by clients using
// the { "mcpServers": {...} } shape. Kept here (not in patch.go) so the
// ClientAdapter interface stays the sole public contract.
func stdInstall(a ClientAdapter, opts InstallOptions, entry map[string]any) (Diff, error) {
	path, err := a.ConfigPath(opts.Scope)
	if err != nil {
		return Diff{}, err
	}
	before, err := readJSONObject(path)
	if err != nil {
		return Diff{}, err
	}
	beforeStr := prettyPrint(before)
	after := upsertMCPServer(cloneMap(before), ServerName, entry)
	if err := writeJSONObject(path, after); err != nil {
		return Diff{}, err
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func stdUninstall(a ClientAdapter, scope Scope) (Diff, error) {
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
