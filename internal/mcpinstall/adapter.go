// Package mcpinstall adds Ramorie's MCP server to the configs of popular
// AI coding tools (Claude Code, Claude Desktop, Cursor, Windsurf, VS Code
// Copilot, Zed). Each adapter knows how to:
//
//   - detect whether the client is installed on the machine
//   - locate its config file for a given scope (user-global vs project)
//   - patch the config idempotently, preserving foreign config
//   - remove the Ramorie entry cleanly
//   - verify our entry is currently present
//
// The TUI in tui.go drives all six adapters through this interface.
package mcpinstall

import "fmt"

// Scope determines where the config lives.
type Scope string

const (
	ScopeUser    Scope = "user"    // user-global config (e.g. ~/.cursor/mcp.json)
	ScopeProject Scope = "project" // project-local (e.g. .cursor/mcp.json in cwd)
)

// ServerName is the key our entry lives under. Consumers can override per
// install if they already have a "ramorie" entry they want to keep separate.
const ServerName = "ramorie"

// DetectionResult describes whether a client is usable on this machine.
type DetectionResult struct {
	Installed   bool   // true if the client's config dir exists or binary was found
	ConfigPath  string // absolute path the adapter would write to (user scope)
	Detail      string // human-readable note shown in TUI (e.g. "settings.json found")
	ProjectOnly bool   // true for clients that only support project-local config
}

// Diff is returned by Install/Uninstall for the preview step. Before/After
// are JSON-pretty-printed blobs and show what will change. Empty Before means
// the file doesn't exist yet (will be created).
type Diff struct {
	Path   string
	Before string
	After  string
}

// InstallOptions configures what we write. Command is the absolute binary
// path; Args are passed verbatim (default: ["mcp", "serve"]).
type InstallOptions struct {
	Scope   Scope
	Command string
	Args    []string
	Env     map[string]string
}

// ClientAdapter is the per-client contract.
type ClientAdapter interface {
	ID() string                              // stable identifier, e.g. "claude-code"
	Name() string                            // display name, e.g. "Claude Code"
	SupportedScopes() []Scope                // which scopes this client understands
	Detect() DetectionResult                 // user scope detection (project scope is cwd-relative)
	ConfigPath(scope Scope) (string, error)  // resolve config path for scope; error if scope unsupported
	Install(opts InstallOptions) (Diff, error)
	Uninstall(scope Scope) (Diff, error)
	IsInstalled(scope Scope) bool // does our entry currently exist in that scope?
}

// Registry is the canonical ordering shown in the TUI.
func Registry() []ClientAdapter {
	return []ClientAdapter{
		&claudeCodeAdapter{},
		&claudeDesktopAdapter{},
		&cursorAdapter{},
		&windsurfAdapter{},
		&vscodeAdapter{},
		&zedAdapter{},
	}
}

// ErrScopeNotSupported is returned by adapters whose client doesn't support a
// given scope (e.g. VS Code workspace-only).
type ErrScopeNotSupported struct {
	ClientID string
	Scope    Scope
}

func (e ErrScopeNotSupported) Error() string {
	return fmt.Sprintf("%s does not support scope=%s", e.ClientID, e.Scope)
}
