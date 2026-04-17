package mcpinstall

import (
	"os"
	"path/filepath"
	"runtime"
)

// Claude Desktop uses claude_desktop_config.json under the platform-appropriate
// application-support directory. User-scope only.
type claudeDesktopAdapter struct{}

func (claudeDesktopAdapter) ID() string   { return "claude-desktop" }
func (claudeDesktopAdapter) Name() string { return "Claude Desktop" }
func (claudeDesktopAdapter) SupportedScopes() []Scope {
	return []Scope{ScopeUser}
}

func claudeDesktopConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		// %APPDATA% is typically ~/AppData/Roaming on recent Windows.
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "Claude", "claude_desktop_config.json"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json"), nil
	default:
		// Linux — XDG config.
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Claude", "claude_desktop_config.json"), nil
		}
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
	}
}

func (a claudeDesktopAdapter) Detect() DetectionResult {
	path, err := claudeDesktopConfigPath()
	if err != nil {
		return DetectionResult{}
	}
	if _, err := os.Stat(path); err == nil {
		return DetectionResult{Installed: true, ConfigPath: path, Detail: "config file found"}
	}
	// Config dir without file — treat as installed (Claude Desktop creates
	// the file only after first-launch).
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err == nil {
		return DetectionResult{Installed: true, ConfigPath: path, Detail: "app dir present; file will be created"}
	}
	return DetectionResult{Installed: false, ConfigPath: path, Detail: "Claude Desktop not detected"}
}

func (a claudeDesktopAdapter) ConfigPath(scope Scope) (string, error) {
	if scope != ScopeUser {
		return "", ErrScopeNotSupported{ClientID: a.ID(), Scope: scope}
	}
	return claudeDesktopConfigPath()
}

func (a claudeDesktopAdapter) Install(opts InstallOptions) (Diff, error) {
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
		return Diff{}, err
	}
	return Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}, nil
}

func (a claudeDesktopAdapter) Uninstall(scope Scope) (Diff, error) {
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

func (a claudeDesktopAdapter) IsInstalled(scope Scope) bool {
	path, err := a.ConfigPath(scope)
	if err != nil {
		return false
	}
	return hasMCPServerEntry(path, ServerName)
}
