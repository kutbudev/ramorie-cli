# Ramorie MCP — Cursor Rules

> Save this file as `.cursorrules` in the root of your project.

## MCP Server Config

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"]
    }
  }
}
```

Then authenticate the CLI (MCP reuses the same credentials):

```bash
ramorie setup login    # writes ~/.ramorie/config.json
```

> **v5.0.0:** 14 unified tools. Legacy tools (`create_task`, `add_memory`,
> `search_memories`, `create_decision`, `activate_context_pack`, `skill`,
> `decision`, `get_active_task`, `manage_focus`) no longer exist. Restart
> Cursor after upgrading the CLI.

---

## Core Principles

### 1. Context-First
Always start by initializing the session:
```
setup_agent                → initialize session, auto-detect project from cwd
task(action="list")        → see pending work
```

### 2. Task-Driven Development
Every piece of work should be tracked via the unified `task` tool:
```
task(action="create", description="...")  → start new work
task(action="start",  id="...")           → begin working
task(action="note",   id="...", note="...")→ log progress
task(action="complete", id="...")         → finish work
```

### 3. Knowledge Persistence
Save everything valuable:
```
remember("content")        → store a memory (type auto-detected)
find("query")              → hybrid semantic + lexical search
recall("term")             → FTS search (pass precision:true for find backend)
```

Decisions are just memories — `remember` auto-tags them. There is no
`create_decision` tool in v5.0.0.

---

## Workflow Rules

### Starting New Work
```
1. setup_agent                     → confirm project context
2. task(action="create", ...)      → create task with clear description
3. task(action="start", id=...)    → mark active
4. task(action="note", id=..., note="initial plan: ...")
```

### During Development
```
- task(action="note", ...)              → log progress at each milestone
- task(action="progress", progress=N)   → 0-100 percentage
- remember("...")                       → save reusable info
- remember("chose X over Y because ...")→ architectural decisions
```

### Completing Work
```
1. task(action="note", id=..., note="summary")
2. task(action="complete", id=...)
3. remember("...")                → (optional) capture key learnings
```

---

## Memory Bank Usage

### When to save memory
- Code patterns that work
- Config snippets
- API endpoint usage
- Error solutions
- Performance optimizations
- Architectural decisions (just `remember` — no separate tool)

### Memory format
```
Good: "PostgreSQL JSONB indexing: CREATE INDEX idx_data ON table USING GIN (data jsonb_path_ops); improves query performance ~10x for JSON searches."
Bad:  "db index stuff"
```

### Searching
Before asking the user:
```
find("relevant keywords")  → hybrid search (preferred for fuzzy)
recall("exact term")       → FTS (fast, lexical)
```

---

## Context Packs

Context packs group related memories + tasks. Manage them via the CLI
(no MCP tool for activation in v5.0.0 — pass `project` field on tool args
to scope queries):

```bash
ramorie context-pack list
ramorie context-pack create <name>
ramorie context-pack delete <id>
```

---

## Quick Reference (14 tools)

| Goal | Tool + Args |
|------|-------------|
| Initialize session | `setup_agent` |
| List projects | `list_projects` |
| Create project | `create_project(name="...")` |
| List tasks | `task(action="list")` |
| Create task | `task(action="create", description="...")` |
| Start task | `task(action="start", id="...")` |
| Note progress | `task(action="note", id="...", note="...")` |
| Update progress | `task(action="progress", id="...", progress=N)` |
| Complete task | `task(action="complete", id="...")` |
| Save info | `remember("...")` |
| Search (fuzzy) | `find("...")` |
| Search (exact) | `recall("...")` |
| List memories | `memory(action="list")` |
| Generate skill | `memory(action="generate", goal="...")` |
| Subtasks | `manage_subtasks(action="...", ...)` |
| Task stats | `get_stats()` |
| Agent timeline | `get_agent_activity()` |
| Context surface | `surface_context(file="...")` |
| Graph ops | `entity(action="...", ...)` |
| Admin ops | `admin(action="consolidate" / "cleanup" / ...)` |

---

## Anti-Patterns

- Don't work without calling `setup_agent` first.
- Don't go long stretches without `task(action="note", ...)`.
- Don't use `remember` for actionable work — use `task(action="create")` (or prefix remember content with `todo:`).
- Don't invent tool names. The 14 above are the complete set.
- Don't store implementation details as memories — attach them to the task via `task(action="note", ...)`.

---

## Progress Tracking

| % | Stage |
|---|-------|
| 0 | Not started |
| 25 | Planning / research |
| 50 | Implementation |
| 75 | Testing |
| 100 | Complete |

---

## Example Session

```
# Start
setup_agent                          → session ready, project auto-detected
task(action="list")                  → no pending work

# New work
task(action="create", description="Implement password reset flow", priority="H")
task(action="start", id="abc123")
task(action="note", id="abc123", note="Plan: 1) email service 2) token gen 3) reset endpoint")

# Progress
task(action="note", id="abc123", note="SendGrid integration done")
task(action="progress", id="abc123", progress=25)
remember("SendGrid: POST /v3/mail/send with API key in Authorization: Bearer header")

# Decision (just a memory)
remember("Chose SendGrid for transactional email: best deliverability, ~$15/mo, trade-off: vendor lock-in on templates")

# Finish
task(action="note", id="abc123", note="Password reset shipped with tests")
task(action="progress", id="abc123", progress=100)
task(action="complete", id="abc123")
```

---

*Ramorie MCP v5.0.0 — 14 tools.*
