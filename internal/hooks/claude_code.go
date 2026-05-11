package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeCodeInstaller installs Ramorie protocol hooks into Claude Code's
// global settings.json (~/.claude/settings.json). The JSON shape is:
//
//	{ "hooks": { "<EventName>": [ { "matcher": "...", "hooks": [ { "type":"command", "command":"...", "id":"..." } ] } ] } }
//
// We always preserve foreign hook entries — only Ramorie-owned IDs are pruned.
type ClaudeCodeInstaller struct {
	// path overrides ~/.claude/settings.json for tests. Empty = use default.
	path string
}

// NewClaudeCodeInstaller returns an installer using the default user-scope
// settings path. Tests should construct the struct directly with path set.
func NewClaudeCodeInstaller() *ClaudeCodeInstaller {
	return &ClaudeCodeInstaller{}
}

// NewClaudeCodeInstallerAt returns an installer rooted at a custom settings
// file path — used by tests to isolate from the developer's real config.
func NewClaudeCodeInstallerAt(path string) *ClaudeCodeInstaller {
	return &ClaudeCodeInstaller{path: path}
}

// Name implements Installer.
func (c *ClaudeCodeInstaller) Name() string { return "claude-code" }

// SettingsPath returns the file we mutate; falls back to user homedir if no
// override is set. Errors surface as the literal "" so Detect() can react.
func (c *ClaudeCodeInstaller) SettingsPath() string {
	if c.path != "" {
		return c.path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "settings.json")
}

// Detect returns true when the settings file already exists. Claude Code
// creates it on first run, so absence usually means "not installed".
func (c *ClaudeCodeInstaller) Detect() bool {
	p := c.SettingsPath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// Install merges entries into the settings file. Three-step algorithm:
//  1. Load raw map[string]interface{} (preserving unknown fields).
//  2. For each unique event in entries, prune any pre-existing Ramorie-owned
//     hook entry, then append the new group.
//  3. Write back with stable 2-space indent.
//
// Running Install with the same input N times produces byte-identical files.
func (c *ClaudeCodeInstaller) Install(entries []HookEntry) error {
	path := c.SettingsPath()
	if path == "" {
		return fmt.Errorf("claude-code: cannot resolve settings path")
	}

	raw, err := loadJSONFile(path)
	if err != nil {
		return err
	}

	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}

	// Group entries by event so we can write one matcher-group per (event,
	// matcher) pair. PostToolUse with matcher "Agent" stays distinct from a
	// matcher-less PostToolUse, mirroring Claude Code's own structure.
	byEvent := map[string][]HookEntry{}
	for _, e := range entries {
		byEvent[string(e.Event)] = append(byEvent[string(e.Event)], e)
	}

	for event, evEntries := range byEvent {
		existing, _ := hooks[event].([]interface{})

		// Drop any prior ramorie-owned entries for this event.
		existing = prunePolicyEntries(existing)

		for _, ent := range evEntries {
			hookSpec := map[string]interface{}{
				"type":    "command",
				"command": ent.Command,
				"id":      ent.ID,
			}
			group := map[string]interface{}{
				"hooks": []interface{}{hookSpec},
			}
			if ent.Matcher != "" {
				group["matcher"] = ent.Matcher
			}
			existing = append(existing, group)
		}

		hooks[event] = existing
	}

	raw["hooks"] = hooks
	return writeJSONFile(path, raw)
}

// Uninstall removes every hook entry whose id matches one of `ids`. If a
// matcher-group becomes empty after pruning it is dropped; if an event's
// list becomes empty the event key is removed. The "hooks" key itself is
// kept (even if empty) to avoid surprising the user with disappearing keys.
func (c *ClaudeCodeInstaller) Uninstall(ids []string) error {
	path := c.SettingsPath()
	if path == "" {
		return fmt.Errorf("claude-code: cannot resolve settings path")
	}
	raw, err := loadJSONFile(path)
	if err != nil {
		return err
	}
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil
	}

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	for event, groupsRaw := range hooks {
		groups, _ := groupsRaw.([]interface{})
		filtered := pruneByIDSet(groups, idSet)
		if len(filtered) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = filtered
		}
	}

	raw["hooks"] = hooks
	return writeJSONFile(path, raw)
}

// Status reads back the installed Ramorie-owned hook entries. Foreign
// entries are ignored so the caller sees only what `ramorie hook install`
// would manage.
func (c *ClaudeCodeInstaller) Status() ([]HookEntry, error) {
	path := c.SettingsPath()
	if path == "" {
		return nil, nil
	}
	raw, err := loadJSONFile(path)
	if err != nil {
		return nil, err
	}
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil, nil
	}
	var out []HookEntry
	for event, groupsRaw := range hooks {
		groups, _ := groupsRaw.([]interface{})
		for _, g := range groups {
			gm, ok := g.(map[string]interface{})
			if !ok {
				continue
			}
			matcher, _ := gm["matcher"].(string)
			inner, _ := gm["hooks"].([]interface{})
			for _, h := range inner {
				hm, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				id, _ := hm["id"].(string)
				if !IsRamorieID(id) {
					continue
				}
				cmd, _ := hm["command"].(string)
				out = append(out, HookEntry{
					Event:   HookEvent(event),
					Matcher: matcher,
					Command: cmd,
					ID:      id,
				})
			}
		}
	}
	return out, nil
}

// loadJSONFile reads a JSON object map. Missing or empty file returns an
// empty map so the caller can treat it as "fresh install" without branching.
func loadJSONFile(path string) (map[string]interface{}, error) {
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

// writeJSONFile writes JSON with 2-space indent and creates the parent dir
// (mode 0755) if needed. File permissions default to 0644.
func writeJSONFile(path string, raw map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// prunePolicyEntries drops any group whose inner hooks list contains a
// Ramorie-owned ID. Foreign hook entries sharing a matcher are preserved.
// This is the close cousin of cli/commands/hook.go's pruneHookEntries but
// scoped to the IsRamorieID prefix check rather than a single identifier.
func prunePolicyEntries(groups []interface{}) []interface{} {
	filtered := make([]interface{}, 0, len(groups))
	for _, g := range groups {
		gm, ok := g.(map[string]interface{})
		if !ok {
			filtered = append(filtered, g)
			continue
		}
		inner, _ := gm["hooks"].([]interface{})
		kept := make([]interface{}, 0, len(inner))
		for _, h := range inner {
			hm, ok := h.(map[string]interface{})
			if !ok {
				kept = append(kept, h)
				continue
			}
			if id, _ := hm["id"].(string); IsRamorieID(id) {
				continue
			}
			kept = append(kept, h)
		}
		if len(kept) == 0 {
			continue
		}
		gm["hooks"] = kept
		filtered = append(filtered, gm)
	}
	return filtered
}

// pruneByIDSet removes hooks whose ID is in idSet. Same group-collapse rule
// as prunePolicyEntries.
func pruneByIDSet(groups []interface{}, idSet map[string]struct{}) []interface{} {
	filtered := make([]interface{}, 0, len(groups))
	for _, g := range groups {
		gm, ok := g.(map[string]interface{})
		if !ok {
			filtered = append(filtered, g)
			continue
		}
		inner, _ := gm["hooks"].([]interface{})
		kept := make([]interface{}, 0, len(inner))
		for _, h := range inner {
			hm, ok := h.(map[string]interface{})
			if !ok {
				kept = append(kept, h)
				continue
			}
			id, _ := hm["id"].(string)
			if _, drop := idSet[id]; drop {
				continue
			}
			kept = append(kept, h)
		}
		if len(kept) == 0 {
			continue
		}
		gm["hooks"] = kept
		filtered = append(filtered, gm)
	}
	return filtered
}
