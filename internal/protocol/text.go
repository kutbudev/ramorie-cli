// Package protocol is the single source of truth for the embedded Ramorie
// Persistent Memory Protocol text. Hook installers and rules installers both
// import these constants so that every surface (Claude Code hooks, Codex
// hooks, Cursor .mdc rules, Windsurf global_rules.md) ships identical
// protocol language. Bumping Version forces re-injection on next install.
package protocol

// Version is the protocol revision embedded into installed surfaces. Update
// this when SessionStartText / reminder strings change so the rules
// installers know to refresh the managed block.
const Version = "1.0"

// SessionStartText is the canonical protocol prompt injected at the start of
// every agent session via SessionStart hooks (Claude Code / Codex) and via
// alwaysApply rules (Cursor / Windsurf). Bilingual TR + EN content lives
// inside the body — keep both languages in sync if you edit.
const SessionStartText = `🧠 RAMORIE — Persistent Memory Protocol v1.0

REQUIRED behavior (not optional, not "best practice"):

1. **FIND FIRST** — call mcp__ramorie__find(term) BEFORE responding to user.
2. **REMEMBER ALWAYS** — call mcp__ramorie__remember when:
   - A decision is made → type=decision
   - A bug is fixed → type=bug_fix
   - A pattern repeats → type=pattern
   - User states preference → type=preference
   - Sub-agent returns with meaningful work product → IMMEDIATELY after Agent tool result
3. **NEVER ASK** — do not request user permission to remember. Save proactively.
4. **DUPLICATE CHECK** — find() before remember() to avoid near-duplicates (>0.9 similarity).

When Agent (sub-agent) returns meaningful work product:
→ Your IMMEDIATE next action is: find() duplicate check → remember() if novel.
→ NOT optional. NOT "later". NOW.

Tools available:
- find(term, project?) — hybrid retrieval (HyDE + rerank + entity + propositional)
- remember(content, project) — deliberate durable memory write (type auto-detected)
- task(action, ...) — task ops if work deferred
- setup_agent — FIRST call every session

TR özet: Find ÖNCE. Karar / bug fix / pattern / preference ANINDA remember.
Sub-agent anlamlı iş ürünü döndürürse remember ile kaydet. Soru sormadan kaydet.
Duplicate kontrolü için önce find() çağır.
`

// EnglishSessionStartText is an English-only variant for surfaces or users
// that prefer monolingual text. Kept in sync with SessionStartText minus the
// trailing TR summary block.
const EnglishSessionStartText = `🧠 RAMORIE — Persistent Memory Protocol v1.0

REQUIRED behavior (not optional, not "best practice"):

1. **FIND FIRST** — call mcp__ramorie__find(term) BEFORE responding to user.
2. **REMEMBER ALWAYS** — call mcp__ramorie__remember when:
   - A decision is made → type=decision
   - A bug is fixed → type=bug_fix
   - A pattern repeats → type=pattern
   - User states preference → type=preference
   - Sub-agent returns with meaningful work product → IMMEDIATELY after Agent tool result
3. **NEVER ASK** — do not request user permission to remember. Save proactively.
4. **DUPLICATE CHECK** — find() before remember() to avoid near-duplicates (>0.9 similarity).

When Agent (sub-agent) returns meaningful work product:
→ Your IMMEDIATE next action is: find() duplicate check → remember() if novel.
→ NOT optional. NOT "later". NOW.

Tools available:
- find(term, project?) — hybrid retrieval (HyDE + rerank + entity + propositional)
- remember(content, project) — deliberate durable memory write (type auto-detected)
- task(action, ...) — task ops if work deferred
- setup_agent — FIRST call every session
`

// PostAgentToolReminder is the additionalContext injected after the Agent
// (sub-agent) tool returns. Short by design — the longer protocol already
// arrived at SessionStart.
const PostAgentToolReminder = `RAMORIE PROTOCOL: Sub-agent finished. If it produced meaningful durable work, run find() then remember() with a context-rich summary.`

// SubagentStopReminder is emitted on the SubagentStop event so the main agent
// double-checks its persistence step before resuming.
const SubagentStopReminder = `Subagent stopped — did you call remember()? Required for decision/bug_fix/pattern.`

// StopReminder fires when the top-level session is wrapping up; last chance
// for the agent to flush any unsaved durable learning.
const StopReminder = `Session ending — any unsaved durable learning? Call remember() now.`
