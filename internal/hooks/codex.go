package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

// CodexInstaller installs Ramorie protocol hooks into Codex CLI. Codex's
// primary config is `~/.codex/config.toml`, but mixing TOML and JSON edits
// is fragile — so we keep hooks in a sibling JSON file `~/.codex/hooks.json`
// that shares the same schema as Claude Code's settings.json hooks block.
//
// Detection key: presence of `~/.codex/config.toml` (proof of Codex install).
// Mutation target: `~/.codex/hooks.json` (separate file, JSON, our own).
type CodexInstaller struct {
	hooksPath  string // override for tests; empty = ~/.codex/hooks.json
	configPath string // override for tests; empty = ~/.codex/config.toml
}

// NewCodexInstaller constructs the default installer.
func NewCodexInstaller() *CodexInstaller {
	return &CodexInstaller{}
}

// NewCodexInstallerAt constructs an installer with explicit paths — both
// the detection probe and the hooks file are overridable for tests.
func NewCodexInstallerAt(hooksPath, configPath string) *CodexInstaller {
	return &CodexInstaller{hooksPath: hooksPath, configPath: configPath}
}

// Name implements Installer.
func (c *CodexInstaller) Name() string { return "codex" }

// SettingsPath returns the hooks.json file we actually write to.
func (c *CodexInstaller) SettingsPath() string {
	if c.hooksPath != "" {
		return c.hooksPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "hooks.json")
}

// configProbe is the file we check to decide whether Codex is installed.
func (c *CodexInstaller) configProbe() string {
	if c.configPath != "" {
		return c.configPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "config.toml")
}

// Detect returns true when the Codex config.toml exists. We DO NOT write to
// config.toml — only check it as a presence signal.
func (c *CodexInstaller) Detect() bool {
	p := c.configProbe()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// Install reuses the Claude Code JSON merge logic — same schema, separate
// file. By construction the call is idempotent.
func (c *CodexInstaller) Install(entries []HookEntry) error {
	path := c.SettingsPath()
	if path == "" {
		return fmt.Errorf("codex: cannot resolve hooks path")
	}
	// Delegate to the shared JSON merger by piggy-backing on a temporary
	// ClaudeCodeInstaller pointed at the codex hooks path.
	delegate := NewClaudeCodeInstallerAt(path)
	return delegate.Install(entries)
}

// Uninstall delegates to the shared JSON merger.
func (c *CodexInstaller) Uninstall(ids []string) error {
	path := c.SettingsPath()
	if path == "" {
		return fmt.Errorf("codex: cannot resolve hooks path")
	}
	delegate := NewClaudeCodeInstallerAt(path)
	return delegate.Uninstall(ids)
}

// Status delegates to the shared JSON reader.
func (c *CodexInstaller) Status() ([]HookEntry, error) {
	path := c.SettingsPath()
	if path == "" {
		return nil, nil
	}
	delegate := NewClaudeCodeInstallerAt(path)
	return delegate.Status()
}
