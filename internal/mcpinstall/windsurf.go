package mcpinstall

import (
	"os"
	"path/filepath"
)

// Windsurf (Codeium) stores MCP config at ~/.codeium/windsurf/mcp_config.json.
// Standard mcpServers shape. User-scope only.
type windsurfAdapter struct{}

func (windsurfAdapter) ID() string                { return "windsurf" }
func (windsurfAdapter) Name() string              { return "Windsurf" }
func (windsurfAdapter) SupportedScopes() []Scope  { return []Scope{ScopeUser} }

func windsurfPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"), nil
}

func (a windsurfAdapter) Detect() DetectionResult {
	p, err := windsurfPath()
	if err != nil {
		return DetectionResult{}
	}
	if _, err := os.Stat(p); err == nil {
		return DetectionResult{Installed: true, ConfigPath: p, Detail: "mcp_config.json found"}
	}
	dir := filepath.Dir(p)
	if _, err := os.Stat(dir); err == nil {
		return DetectionResult{Installed: true, ConfigPath: p, Detail: "windsurf dir present; file will be created"}
	}
	return DetectionResult{Installed: false, ConfigPath: p, Detail: "Windsurf not detected"}
}

func (a windsurfAdapter) ConfigPath(scope Scope) (string, error) {
	if scope != ScopeUser {
		return "", ErrScopeNotSupported{ClientID: a.ID(), Scope: scope}
	}
	return windsurfPath()
}

func (a windsurfAdapter) Install(opts InstallOptions) (Diff, error) {
	return stdInstall(a, opts, standardServerEntry(opts.Command, opts.Args, opts.Env))
}

func (a windsurfAdapter) Uninstall(scope Scope) (Diff, error) {
	return stdUninstall(a, scope)
}

func (a windsurfAdapter) IsInstalled(scope Scope) bool {
	p, err := a.ConfigPath(scope)
	if err != nil {
		return false
	}
	return hasMCPServerEntry(p, ServerName)
}
