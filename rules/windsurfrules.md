# Ramorie MCP — Windsurf Rules

> Save this file as `.windsurfrules` in the root of your project.

## MCP Server Config

Add to `~/.codeium/windsurf/mcp_config.json` (or the project-level
`.windsurf/mcp.json`):

```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"],
      "env": {}
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
> `decision`, `get_active_task`, `manage_focus`) have been removed. Restart
> Windsurf after upgrading the CLI.

---

## Agent Rules

### Core Principles

1. **Initialize context each session**
   - First call: `setup_agent` (auto-detects project from cwd)

2. **Task-driven work**
   - New work → `task(action="create")` + `task(action="start")`
   - Track progress → `task(action="note")`
   - When done → `task(action="complete")`

3. **Persist knowledge**
   - Save learnings → `remember("...")`
   - Search before asking → `find("...")` or `recall("...")`

4. **Record decisions as memories**
   - `remember("chose X over Y because ...")` — auto-tagged by server
   - No separate `create_decision` tool in v5.0.0

---

## Task Management

### Starting a task
```
1. setup_agent                    → confirm context
2. task(action="create",
        description="clear title",
        priority="H"|"M"|"L")
3. task(action="start", id=...)
4. task(action="note",  id=..., note="initial plan")
```

### While working
```
- Each meaningful step  → task(action="note", ...)
- Every ~25% milestone  → task(action="progress", progress=N)
- Learned something     → remember("...")
- Made a decision       → remember("chose ... because ...")
```

### Completing
```
1. task(action="note",     id=..., note="summary")
2. task(action="complete", id=...)
3. remember("...")            → optional key learning
```

---

## Memory

### Use `remember` for
- Reusable technical info
- Error solutions
- Architectural decisions
- Performance notes

### Memory format
```
Good:
"PostgreSQL connection pooling: max_connections=100, pool_size=20. Use pgbouncer for throughput > 2k rps."

Bad:
"db settings"
```

### Search before asking the user
```
find("pg connection pool")     → hybrid semantic + lexical
recall("pool_size")            → FTS for exact terms
```

---

## Context Packs

Context packs group related memories + tasks. Managed via the CLI only
(no MCP tool to activate them in v5.0.0 — pass `project` on tool args to
scope):

```bash
ramorie context-pack list
ramorie context-pack create "feature-auth"
ramorie context-pack delete <id>
```

---

## Quick Reference (14 tools)

| Action | Tool |
|--------|------|
| Initialize | `setup_agent` |
| Projects | `list_projects`, `create_project` |
| Task ops | `task(action=list/get/create/start/complete/stop/progress/note/move)` |
| Subtasks | `manage_subtasks` |
| Save memory | `remember` |
| List memories | `memory(action="list")` |
| Generate skill | `memory(action="generate", goal="...")` |
| Search (fuzzy) | `find` |
| Search (exact) | `recall` |
| Stats | `get_stats` |
| Timeline | `get_agent_activity` |
| Context | `surface_context` |
| Graph | `entity` |
| Admin | `admin` |

---

## Anti-Patterns

1. Starting work without `setup_agent`
2. Long stretches without `task(action="note", ...)`
3. Skipping decisions — just `remember("chose ...")` it
4. Using `remember` for actionable work (use `task(action="create")`)
5. Inventing tool names — 14 tools are the complete set

---

*Ramorie MCP v5.0.0 — 14 tools.*
