package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

// NewHookCommand exposes `ramorie hook install|uninstall|status|context`.
// It drives Claude Code PreToolUse integration so that editing a file
// automatically surfaces relevant memories/decisions into the model's
// context — no manual recall() needed.
func NewHookCommand() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "Manage Claude Code integration (PreToolUse hook)",
		Subcommands: []*cli.Command{
			{
				Name:   "install",
				Usage:  "Install the PreToolUse hook into ~/.claude/settings.json",
				Action: hookInstall,
			},
			{
				Name:   "uninstall",
				Usage:  "Remove the Ramorie hook from ~/.claude/settings.json",
				Action: hookUninstall,
			},
			{
				Name:   "status",
				Usage:  "Check whether the hook is installed and wired correctly",
				Action: hookStatus,
			},
			{
				Name:   "context",
				Usage:  "Hook shim: read PreToolUse JSON from stdin, emit system-reminder",
				Hidden: true, // called from the shim, not humans
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "budget", Value: 500},
					&cli.IntFlag{Name: "limit", Value: 2},
				},
				Action: hookContext,
			},
		},
	}
}

const (
	hookMatcher      = "Edit|Write|Read"
	hookIdentifier   = "ramorie-autocontext"
	hookCooldownSecs = 30
)

type claudeSettings struct {
	Hooks map[string][]hookGroup `json:"hooks,omitempty"`
	// Preserve unknown fields so we don't clobber user config.
	Rest map[string]json.RawMessage `json:"-"`
}

type hookGroup struct {
	Matcher string     `json:"matcher,omitempty"`
	Hooks   []hookSpec `json:"hooks"`
}

type hookSpec struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	// Marker field we set so we can identify our own entries on uninstall.
	ID string `json:"id,omitempty"`
}

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func ramorieBinary() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "ramorie"
}

func loadSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if raw == nil {
		return map[string]interface{}{}, nil
	}
	return raw, nil
}

func saveSettings(path string, raw map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func hookInstall(c *cli.Context) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	raw, err := loadSettings(path)
	if err != nil {
		return err
	}

	bin := ramorieBinary()
	entry := map[string]interface{}{
		"type":    "command",
		"command": fmt.Sprintf("%s hook context --budget 500 --limit 2", bin),
		"id":      hookIdentifier,
	}

	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}
	preUse, _ := hooks["PreToolUse"].([]interface{})

	// Remove any prior ramorie entry so reinstalling is idempotent.
	preUse = pruneHookEntries(preUse, hookIdentifier)

	preUse = append(preUse, map[string]interface{}{
		"matcher": hookMatcher,
		"hooks":   []interface{}{entry},
	})
	hooks["PreToolUse"] = preUse
	raw["hooks"] = hooks

	if err := saveSettings(path, raw); err != nil {
		return err
	}
	fmt.Printf("✅ Installed PreToolUse hook into %s\n", path)
	fmt.Printf("   Matcher: %s\n", hookMatcher)
	fmt.Printf("   Command: %s hook context ...\n", bin)
	fmt.Println("   Restart Claude Code to activate.")
	return nil
}

func hookUninstall(c *cli.Context) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	raw, err := loadSettings(path)
	if err != nil {
		return err
	}
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		fmt.Println("No hooks configured — nothing to remove.")
		return nil
	}
	preUse, _ := hooks["PreToolUse"].([]interface{})
	before := len(preUse)
	preUse = pruneHookEntries(preUse, hookIdentifier)
	if len(preUse) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = preUse
	}
	if len(hooks) == 0 {
		delete(raw, "hooks")
	} else {
		raw["hooks"] = hooks
	}
	if err := saveSettings(path, raw); err != nil {
		return err
	}
	removed := before - len(preUse)
	fmt.Printf("✅ Removed %d hook entrie(s) from %s\n", removed, path)
	return nil
}

func hookStatus(c *cli.Context) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	raw, err := loadSettings(path)
	if err != nil {
		return err
	}
	hooks, _ := raw["hooks"].(map[string]interface{})
	preUse, _ := hooks["PreToolUse"].([]interface{})
	found := false
	for _, entry := range preUse {
		group, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		inner, _ := group["hooks"].([]interface{})
		for _, h := range inner {
			hmap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if hmap["id"] == hookIdentifier {
				found = true
				fmt.Printf("✅ Ramorie hook installed\n")
				fmt.Printf("   Path:    %s\n", path)
				fmt.Printf("   Matcher: %v\n", group["matcher"])
				fmt.Printf("   Command: %v\n", hmap["command"])
			}
		}
	}
	if !found {
		fmt.Printf("❌ Ramorie hook not installed.\n")
		fmt.Println("   Run: ramorie hook install")
	}
	return nil
}

// pruneHookEntries removes any PreToolUse groups whose inner hook matches
// the given identifier. Preserves foreign (non-ramorie) entries untouched.
func pruneHookEntries(preUse []interface{}, identifier string) []interface{} {
	filtered := make([]interface{}, 0, len(preUse))
	for _, entry := range preUse {
		group, ok := entry.(map[string]interface{})
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		inner, _ := group["hooks"].([]interface{})
		kept := make([]interface{}, 0, len(inner))
		for _, h := range inner {
			hmap, ok := h.(map[string]interface{})
			if !ok {
				kept = append(kept, h)
				continue
			}
			if hmap["id"] == identifier {
				continue // drop ramorie-owned entry
			}
			kept = append(kept, h)
		}
		if len(kept) == 0 {
			continue // drop group if it becomes empty
		}
		group["hooks"] = kept
		filtered = append(filtered, group)
	}
	return filtered
}

// hookContext is invoked by Claude Code as a PreToolUse shim. Reads the
// tool-call JSON from stdin, extracts a file path, calls the backend surface
// endpoint and writes a Claude Code compatible hook response on stdout.
//
// Output schema (per Claude Code hooks spec):
//
//	{"hookSpecificOutput": {"hookEventName":"PreToolUse","additionalContext":"..."}}
//
// Any failure is silent (exit 0, empty output) so hook errors never block
// the user's tool call.
func hookContext(c *cli.Context) error {
	payload := map[string]interface{}{}
	dec := json.NewDecoder(os.Stdin)
	_ = dec.Decode(&payload) // non-fatal; empty stdin is fine

	filePath := extractFilePathFromPayload(payload)
	if filePath == "" {
		return nil
	}

	// Cooldown: don't repeat the same file within 30s. File mtime is cheap.
	if wasRecentlyProcessed(filePath) {
		return nil
	}
	markProcessed(filePath)

	// Shell out to `ramorie hook-context-call` via the find-related helper so
	// this function stays focused on I/O shape.
	budget := c.Int("budget")
	limit := c.Int("limit")
	cmd := exec.Command(ramorieBinary(), "find-related",
		"--file", filePath,
		"--budget", fmt.Sprintf("%d", budget),
		"--limit", fmt.Sprintf("%d", limit))
	out, err := cmd.Output()
	if err != nil {
		return nil // silent
	}
	additional := strings.TrimSpace(string(out))
	if additional == "" {
		return nil
	}

	resp := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "PreToolUse",
			"additionalContext": additional,
		},
	}
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(resp)
	return nil
}

// extractFilePathFromPayload tries the common shapes Claude Code sends.
// Examples:
//
//	{"tool_name":"Edit","tool_input":{"file_path":"/abs/path.go"}}
//	{"tool_name":"Read","tool_input":{"file_path":"/abs/path.go"}}
func extractFilePathFromPayload(p map[string]interface{}) string {
	ti, _ := p["tool_input"].(map[string]interface{})
	if ti == nil {
		return ""
	}
	if fp, ok := ti["file_path"].(string); ok {
		return fp
	}
	// Some tools use `path`
	if fp, ok := ti["path"].(string); ok {
		return fp
	}
	return ""
}
