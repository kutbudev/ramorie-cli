// Package hooks installs the Ramorie Persistent Memory Protocol into client
// CLIs that support shell-command hooks (Claude Code, Codex). The installer
// API is intentionally narrow: each client implementation knows how to read
// its own settings file, merge our managed hook entries idempotently, and
// remove them on uninstall without disturbing foreign entries.
package hooks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/protocol"
)

// HookEvent enumerates the lifecycle hook points Ramorie cares about.
// SessionStart fires once at agent boot; PostToolUse with matcher "Agent"
// fires after each sub-agent call; SubagentStop and Stop wrap up nested and
// top-level sessions respectively.
type HookEvent string

const (
	// SessionStart fires once when the agent session begins.
	SessionStart HookEvent = "SessionStart"
	// PostToolUse fires after a tool call; pair with Matcher to scope.
	PostToolUse HookEvent = "PostToolUse"
	// SubagentStop fires when a sub-agent finishes.
	SubagentStop HookEvent = "SubagentStop"
	// Stop fires when the top-level session ends.
	Stop HookEvent = "Stop"
)

// HookEntry is the installer-level description of one managed hook. The
// concrete settings.json shape is constructed by each Installer.
type HookEntry struct {
	// Event is the lifecycle phase this hook listens to.
	Event HookEvent
	// Matcher narrows PostToolUse to a specific tool name (e.g. "Agent").
	// Empty matcher means "all" — most events leave this blank.
	Matcher string
	// Command is the shell command emitted into settings.json. Must produce
	// the JSON shape the client expects on stdout (hookSpecificOutput) for
	// events that consume additionalContext.
	Command string
	// ID is a stable, ramorie-prefixed identifier used for idempotent
	// uninstall — entries carrying this ID are pruned before re-install.
	ID string
}

// Installer is the contract every client integration implements. Detect
// gates auto-install on missing CLIs; SettingsPath is surfaced for user
// debugging; Install/Uninstall/Status drive the actual write.
type Installer interface {
	// Name returns a stable, human-readable identifier ("claude-code").
	Name() string
	// Detect reports whether this client is present on the system.
	Detect() bool
	// SettingsPath is the absolute path to the file we mutate.
	SettingsPath() string
	// Install merges entries into the settings file. Must be idempotent —
	// running Install 3x must produce identical output.
	Install(entries []HookEntry) error
	// Uninstall removes any entry whose ID is in ids. Foreign entries
	// MUST be left untouched.
	Uninstall(ids []string) error
	// Status returns the currently-installed managed entries.
	Status() ([]HookEntry, error)
}

// DefaultEntries is the canonical set of hooks Ramorie installs. Order is
// stable so installers can rely on positional reasoning if needed.
func DefaultEntries() []HookEntry {
	return []HookEntry{
		{
			Event:   SessionStart,
			Command: makeHookCmd(protocol.SessionStartText, "SessionStart"),
			ID:      "ramorie-protocol-session-start-v1",
		},
		{
			Event:   PostToolUse,
			Matcher: "Agent",
			Command: makeHookCmd(protocol.PostAgentToolReminder, "PostToolUse"),
			ID:      "ramorie-protocol-post-agent-v1",
		},
		{
			Event:   SubagentStop,
			Command: makeSystemMsg(protocol.SubagentStopReminder),
			ID:      "ramorie-protocol-subagent-stop-v1",
		},
		{
			Event:   Stop,
			Command: makeSystemMsg(protocol.StopReminder),
			ID:      "ramorie-protocol-stop-v1",
		},
	}
}

// makeHookCmd builds a `cat <<'EOF' … EOF` shell command that emits the
// Claude Code / Codex hookSpecificOutput JSON. We pre-encode the additional
// context with json.Marshal so embedded quotes / newlines survive intact.
func makeHookCmd(additionalContext, hookEventName string) string {
	payload := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     hookEventName,
			"additionalContext": additionalContext,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		// Should be impossible with a static struct, but stay defensive.
		return ""
	}
	// Use a sentinel heredoc tag so embedded $ and ` survive unchanged.
	return fmt.Sprintf("cat <<'RAMORIE_EOF'\n%s\nRAMORIE_EOF", string(b))
}

// makeSystemMsg builds a shell command that echoes a short reminder via the
// `systemMessage` channel — used for events (SubagentStop / Stop) where we
// want a visible nudge rather than silent context injection.
func makeSystemMsg(msg string) string {
	payload := map[string]interface{}{
		"systemMessage": msg,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("cat <<'RAMORIE_EOF'\n%s\nRAMORIE_EOF", string(b))
}

// IsRamorieID reports whether an arbitrary hook id was installed by Ramorie.
// Used by Uninstall to bulk-prune without listing every ID.
func IsRamorieID(id string) bool {
	return strings.HasPrefix(id, "ramorie-protocol-")
}
