package mcpinstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSON patch helpers shared by adapters whose config uses the standard
// { "mcpServers": { <name>: {command, args, env} } } shape. This covers
// Claude Code, Claude Desktop, Cursor, Windsurf. VS Code and Zed use
// different shapes and patch inline.

// readJSONObject reads path as a JSON object. Missing file returns an empty
// map (not an error) so first-install is a clean write. Malformed JSON is
// surfaced so we never silently overwrite the user's existing config.
func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w (refusing to overwrite malformed config)", path, err)
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func writeJSONObject(path string, v map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// prettyPrint returns the indented JSON for a map, used to build Diff.Before /
// Diff.After without a trailing newline.
func prettyPrint(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	return string(b)
}

// upsertMCPServer sets mcpServers[name] = serverEntry in `raw`, preserving
// all other keys. Used by the standard-format adapters.
func upsertMCPServer(raw map[string]any, name string, serverEntry map[string]any) map[string]any {
	servers, _ := raw["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[name] = serverEntry
	raw["mcpServers"] = servers
	return raw
}

// removeMCPServer deletes mcpServers[name]. Empties the mcpServers map if it
// becomes empty after removal; removes the top-level key if no other servers
// remain, so uninstall leaves no residue.
func removeMCPServer(raw map[string]any, name string) map[string]any {
	servers, _ := raw["mcpServers"].(map[string]any)
	if servers == nil {
		return raw
	}
	delete(servers, name)
	if len(servers) == 0 {
		delete(raw, "mcpServers")
	} else {
		raw["mcpServers"] = servers
	}
	return raw
}

// standardServerEntry builds the {command, args, env} shape most clients share.
func standardServerEntry(command string, args []string, env map[string]string) map[string]any {
	entry := map[string]any{
		"command": command,
		"args":    args,
	}
	if len(env) > 0 {
		// env should be stable for diff-ability — convert to an ordered-ish
		// map. JSON serialization will sort keys alphabetically.
		entry["env"] = env
	}
	return entry
}
