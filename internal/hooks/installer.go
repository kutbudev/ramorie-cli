// Package hooks installs the Ramorie Persistent Memory Protocol into client
// CLIs that support shell-command hooks (Claude Code, Codex). The installer
// API is intentionally narrow: each client implementation knows how to read
// its own settings file, merge our managed hook entries idempotently, and
// remove them on uninstall without disturbing foreign entries.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/protocol"
)

// HookEvent enumerates the lifecycle hook points Ramorie cares about.
// SessionStart fires once at agent boot; PostToolUse with matcher "Task"
// fires after each sub-agent call (the Task tool is Claude Code's sub-agent
// dispatcher); SubagentStop and Stop wrap up nested and top-level sessions
// respectively.
type HookEvent string

const (
	// SessionStart fires once when the agent session begins.
	SessionStart HookEvent = "SessionStart"
	// UserPromptSubmit fires each time the user submits a prompt, before the
	// model sees it — lets us inject prompt-relevant memories as context. It is
	// not a tool event, so it carries no Matcher.
	UserPromptSubmit HookEvent = "UserPromptSubmit"
	// PreToolUse fires before a tool call; pair with Matcher to scope.
	PreToolUse HookEvent = "PreToolUse"
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
	// Matcher narrows PostToolUse to a specific tool name (e.g. "Task", the
	// Claude Code sub-agent dispatcher). Empty matcher means "all" — most
	// events leave this blank.
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
			Command: makeSessionStartCmd(),
			ID:      "ramorie-protocol-session-start-v1",
		},
		{
			// Prompt-relevant memory injection: each user prompt is used as a
			// retrieval query so active preferences + related decisions/skills
			// surface as context before the model answers. Silent on trivial
			// prompts (see `hook prompt-submit`).
			Event:   UserPromptSubmit,
			Command: makePromptSubmitCmd(),
			ID:      "ramorie-protocol-prompt-submit-v1",
		},
		{
			Event:   PreToolUse,
			Matcher: "Bash|Shell",
			Command: makeBeforeActionCmd(),
			ID:      "ramorie-protocol-before-action-v1",
		},
		{
			// Per-file context injection: opening/editing a file surfaces the
			// decisions + bug_fix + pattern memories tied to that module. The
			// legacy `ramorie hook install` wired this only on the deprecated
			// single-client path; the canonical installer now ships it too.
			Event:   PreToolUse,
			Matcher: "Edit|Write|Read",
			Command: makeFileContextCmd(),
			ID:      "ramorie-protocol-file-context-v1",
		},
		{
			// Matcher is "Task" — Claude Code dispatches sub-agents through the
			// Task tool, not a tool literally named "Agent". A mismatched matcher
			// here means the post-subagent remember() reminder never fires.
			Event:   PostToolUse,
			Matcher: "Task",
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

func makeSessionStartCmd() string {
	// --full opts into the richer startup payload (recent_memories,
	// in_progress_tasks, last_session). Without it those fields stay behind the
	// compact path and no installer ever surfaces them at session start.
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		return shellQuote(exe) + " hook session-start --full"
	}
	return "ramorie hook session-start --full"
}

func makeBeforeActionCmd() string {
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		return shellQuote(exe) + " hook before-action --budget 1200 --limit 3"
	}
	return "ramorie hook before-action --budget 1200 --limit 3"
}

func makeFileContextCmd() string {
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		return shellQuote(exe) + " hook context --budget 500 --limit 2"
	}
	return "ramorie hook context --budget 500 --limit 2"
}

func makePromptSubmitCmd() string {
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		return shellQuote(exe) + " hook prompt-submit --budget 700 --limit 4"
	}
	return "ramorie hook prompt-submit --budget 700 --limit 4"
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
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

// DiffEntries compares the canonical hook set with what a client currently
// reports. Missing means the canonical ID is absent. Stale means the ID exists
// but event, matcher, or command differs from the current binary's template.
func DiffEntries(expected, installed []HookEntry) (missing []HookEntry, stale []HookEntry) {
	byID := make(map[string]HookEntry, len(installed))
	for _, entry := range installed {
		if entry.ID == "" {
			continue
		}
		byID[entry.ID] = entry
	}
	for _, want := range expected {
		got, ok := byID[want.ID]
		if !ok {
			missing = append(missing, want)
			continue
		}
		if got.Event != want.Event || got.Matcher != want.Matcher ||
			normalizeHookCommand(got.Command) != normalizeHookCommand(want.Command) {
			stale = append(stale, want)
		}
	}
	return missing, stale
}

// normalizeHookCommand strips the absolute binary path that the hook shim
// commands embed (e.g. "'/usr/local/bin/ramorie' hook session-start --full").
// The path varies by install location, so a literal command comparison would
// flag every entry as stale whenever the binary moved — even though the shim
// invocation is identical. Heredoc commands (cat <<'RAMORIE_EOF' ...) carry no
// path and pass through unchanged.
func normalizeHookCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if idx := strings.Index(cmd, " hook "); idx != -1 {
		return "ramorie" + cmd[idx:]
	}
	if strings.HasSuffix(cmd, " hook") {
		return "ramorie hook"
	}
	return cmd
}
