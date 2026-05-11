package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/protocol"
)

// CursorInstaller writes a project-scoped Cursor rule file at
// `.cursor/rules/ramorie-memory-protocol.mdc`. The .mdc format is markdown
// with a YAML-ish frontmatter block; we set `alwaysApply: true` so the
// protocol injects into every Cursor chat for the project.
//
// User-scope ("global") cursor rules don't exist as a first-class concept,
// so when no project root is available we surface a TODO message rather
// than silently writing to an unexpected location.
type CursorInstaller struct {
	// projectRoot overrides cwd. Empty = use os.Getwd().
	projectRoot string
}

// NewCursorInstaller constructs the default project-root cursor installer.
func NewCursorInstaller() *CursorInstaller { return &CursorInstaller{} }

// NewCursorInstallerAt constructs an installer rooted at an explicit dir.
func NewCursorInstallerAt(root string) *CursorInstaller {
	return &CursorInstaller{projectRoot: root}
}

// Name implements Installer.
func (c *CursorInstaller) Name() string { return "cursor" }

// projectDir resolves the project root from the override or cwd.
func (c *CursorInstaller) projectDir() string {
	if c.projectRoot != "" {
		return c.projectRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

// RulesPath is the .mdc file we write inside the resolved project root.
func (c *CursorInstaller) RulesPath() string {
	root := c.projectDir()
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".cursor", "rules", "ramorie-memory-protocol.mdc")
}

// Detect returns true if a Cursor config root exists alongside the project.
// We probe for any of: `.cursor/` dir at project root, or the presence of
// an existing rules file (re-install case). Absence is not fatal — we'll
// happily create the directory on Install.
func (c *CursorInstaller) Detect() bool {
	root := c.projectDir()
	if root == "" {
		return false
	}
	// Treat any of these as "Cursor in play" — Cursor itself uses .cursor/.
	if _, err := os.Stat(filepath.Join(root, ".cursor")); err == nil {
		return true
	}
	if _, err := os.Stat(c.RulesPath()); err == nil {
		return true
	}
	return false
}

// Install writes the .mdc file. Text is wrapped in alwaysApply frontmatter
// so Cursor picks it up automatically. Idempotent: same input → same bytes.
// `text` is typically protocol.SessionStartText but callers may override.
func (c *CursorInstaller) Install(text string) error {
	path := c.RulesPath()
	if path == "" {
		return fmt.Errorf("cursor: no project root resolved (TODO: user-scope rule nudge)")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := buildCursorMDC(text)
	return os.WriteFile(path, []byte(content), 0o644)
}

// Uninstall removes the file if present.
func (c *CursorInstaller) Uninstall() error {
	path := c.RulesPath()
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Status checks whether a managed file exists at the expected path and
// returns the protocol Version embedded in its frontmatter.
func (c *CursorInstaller) Status() (bool, string, error) {
	path := c.RulesPath()
	if path == "" {
		return false, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	v := extractCursorVersion(string(data))
	return true, v, nil
}

// buildCursorMDC composes the .mdc payload: frontmatter + body. The
// `description`, `globs`, and `alwaysApply` keys are the Cursor-recognised
// rule metadata; the `ramorie_version` key is our own bookkeeping so
// Status() can report it.
func buildCursorMDC(body string) string {
	frontmatter := strings.Join([]string{
		"---",
		"description: Ramorie Persistent Memory Protocol",
		"globs:",
		"alwaysApply: true",
		"ramorie_version: " + protocol.Version,
		"---",
		"",
	}, "\n")
	return frontmatter + strings.TrimSpace(body) + "\n"
}

// extractCursorVersion pulls `ramorie_version: X` from the frontmatter.
// Returns "" when no marker line is present (e.g. user-edited file).
func extractCursorVersion(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ramorie_version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ramorie_version:"))
		}
		// Stop scanning once we leave the frontmatter.
		if line == "---" {
			continue
		}
	}
	return ""
}
