# Ramorie MCP — Global User Rules

> Paste this into your IDE's global user rules / memory section.

---

## Ramorie MCP Usage Guide

Agent reference for the Ramorie MCP server (v5.0.0, 14 tools).

---

## What It Does

Ramorie MCP is a task + knowledge management server with:

- Task tracking (unified `task` tool)
- Memory bank (`remember`, `find`, `recall`, `memory`)
- Subtasks (`manage_subtasks`)
- Knowledge graph (`entity`)
- Agent timeline + stats (`get_agent_activity`, `get_stats`)
- Multi-project + multi-org support (`list_projects`, `create_project`)

Decisions are memories — the server auto-detects and tags them. There is
no separate `decision` / `create_decision` tool in v5.0.0.

---

## Core Rules

### Session start
```
1. setup_agent             → initialize session, auto-detect project
2. task(action="list")     → see pending work
```

### Starting new work
```
1. task(action="create", description="clear title", priority="H"|"M"|"L")
2. task(action="start",  id=...)
3. task(action="note",   id=..., note="initial plan")
```

### While working
```
- task(action="note",     id=..., note="progress step")
- task(action="progress", id=..., progress=0-100)
- remember("learning / decision")
```

### Completing
```
1. task(action="note",     id=..., note="summary")
2. task(action="complete", id=...)
```

---

## Memory Bank

### When to save
- Working code patterns
- Config snippets
- API endpoint usage
- Error solutions
- Performance optimizations
- Architectural decisions

### How to search
```
find("fuzzy query")        → hybrid semantic + lexical (preferred)
recall("exact term")       → FTS
```

**Rule:** Before asking the user, search existing memory.

---

## Decisions (ADRs)

No separate tool. Store as memories — the server auto-detects:

```
remember("Chose PostgreSQL over MongoDB: need ACID for ledger. Trade-off: ops complexity on geo-replication.")
```

Recommended content pattern:
- Title-ish first sentence
- `because ...` / `context ...` / `trade-off ...`

Later, query with `find("database choice")` or `recall`.

---

## Context Management

Context packs group related memories + tasks. Managed via the CLI only
(no MCP activation tool in v5.0.0). Scope tool calls by passing the
`project` field instead:

```bash
ramorie context-pack list
ramorie context-pack create <name>
```

---

## Quick Reference (14 tools)

| Goal | Tool |
|------|------|
| Initialize | `setup_agent` |
| List projects | `list_projects` |
| Create project | `create_project` |
| Task ops | `task` (list/get/create/start/complete/stop/progress/note/move) |
| Subtasks | `manage_subtasks` |
| Save memory | `remember` |
| Memory ops | `memory` (list/get/generate) |
| Fuzzy search | `find` |
| Exact search | `recall` |
| Stats | `get_stats` |
| Timeline | `get_agent_activity` |
| File context | `surface_context` |
| Graph | `entity` |
| Admin | `admin` |

---

## Anti-Patterns

1. Working without `setup_agent`
2. Long stretches without `task(action="note", ...)`
3. Using `remember` for actionable work (use `task(action="create")`)
4. Inventing tool names — 14 above are the complete set
5. Asking the user before running `find` / `recall` first

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

## MCP Setup

### Install the CLI (Homebrew)
```bash
brew tap kutbudev/homebrew-tap
brew install ramorie
ramorie setup login   # email + password → ~/.ramorie/config.json
```

### MCP Config
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

Verify: `ramorie mcp tools` lists all 14 tools.

---

*Ramorie MCP v5.0.0 — 14 tools. Restart your MCP client after upgrading from v4.x.*
