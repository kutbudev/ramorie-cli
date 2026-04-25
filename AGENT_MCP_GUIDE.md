# Ramorie MCP Agent Guide

> Quick reference for AI agents using the Ramorie MCP server (v5.0.0, 14 tools).
> For full CLI docs, see https://ramorie.com/docs/cli or run `ramorie mcp tools`.

## Setup

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

Authenticate the CLI first so the MCP server has an API key:

```bash
ramorie setup login     # interactive email + password, writes ~/.ramorie/config.json
ramorie setup status    # verify
```

> **v5.0.0 upgrade:** Old tools (`create_task`, `add_memory`, `search_memories`,
> `create_decision`, `activate_context_pack`, `get_active_task`, `manage_focus`,
> `skill`, `decision`) are removed. Use the 14 tools below instead. Restart your
> MCP client (Claude Code / Cursor / Windsurf) after upgrading.

---

## The 14 Tools

### Core (always use)

| # | Tool | Purpose |
|---|------|---------|
| 1 | `setup_agent` | Initialize session, auto-detect project from cwd. **Call this first.** |
| 2 | `list_projects` | List personal + org projects. |
| 3 | `remember` | Store a memory. Auto-detects type. Prefix with `todo:` to create a task instead. |
| 4 | `find` | Hybrid semantic + lexical search (HyDE + rerank). Best for fuzzy queries. |
| 5 | `recall` | FTS search; pass `precision: true` to route to `find`. |
| 6 | `task` | Unified task ops. `action`: `list` / `get` / `create` / `start` / `complete` / `stop` / `progress` / `note` / `move`. |

### Common

| # | Tool | Purpose |
|---|------|---------|
| 7 | `memory` | Unified memory ops (`list`, `get`), or skill generation via `goal` param. |
| 8 | `get_stats` | Task counts per project / status. |
| 9 | `get_agent_activity` | Agent timeline query. |
| 10 | `surface_context` | File/domain-scoped context surfacing for the active task. |

### Advanced

| # | Tool | Purpose |
|---|------|---------|
| 11 | `create_project` | Create a project. |
| 12 | `manage_subtasks` | Subtask CRUD. |
| 13 | `entity` | Knowledge graph (10 actions: `create`, `link`, `list`, `search`, etc.). |
| 14 | `admin` | `consolidate` / `cleanup` / `orgs` / `export` / `import` / `plan` / `analyze`. |

---

## Canonical Workflow

```
# 1. Session start
setup_agent → gets active project from cwd + recent activity

# 2. See pending work
task(action="list", status="TODO")

# 3. Start a new unit of work
task(action="create", description="Clear, imperative title", priority="H")
task(action="start", id="<task-id>")
task(action="note", id="<task-id>", note="Initial plan: 1)... 2)... 3)...")

# 4. Work loop
task(action="progress", id="<task-id>", progress=50)
remember("API rate limit is 1000 req/min; use exponential backoff on 429")
find("jwt refresh token pattern")

# 5. Finish
task(action="note", id="<task-id>", note="Summary of what shipped")
task(action="complete", id="<task-id>")
```

---

## Decision Records

The legacy `decision` / `create_decision` tool is gone. Record decisions as
memories — the server auto-detects and tags them:

```
remember("Chose PostgreSQL over MongoDB: need ACID for financial ledger. Trade-off: ops complexity for geo-replication.")
```

Query decisions later with `find("database choice")` or `recall`.

---

## Skills (procedural memory)

Skills are stored via `memory(action="generate", goal="...")`, which produces
a reusable procedure from prior activity, or via plain `remember` with a
step-by-step content body.

```
remember("When deploying to prod: 1) yarn test --ci 2) yarn build 3) verify bundle < 500KB 4) yarn deploy:prod 5) curl /health → 200")
```

Retrieve with `find("deploy production")`.

---

## Context Packs

Context packs group related memories/tasks. Manage them via the CLI:

```bash
ramorie context list
ramorie context create "feature-auth"
```

There is no MCP tool for pack activation in v6.0.0 — the agent passes
project scope through tool args (`project:` field on `remember` / `task` /
`find`).

---

## Anti-Patterns

- Don't call `task(action="create")` for knowledge — use `remember`.
- Don't call `remember` for actionable work — use `task(action="create")` (or prefix content with `todo:`).
- Don't skip `setup_agent` at session start.
- Don't invent tool names. The list above is exhaustive.

---

## CLI Commands (for humans)

| Goal | Command |
|------|---------|
| Login / setup | `ramorie setup` |
| Auth status | `ramorie setup status` |
| Create project | `ramorie project create <name>` |
| List projects | `ramorie project list` |
| Create task | `ramorie task create "<desc>"` |
| Start task | `ramorie task start <id>` |
| Complete task | `ramorie task complete <id>` |
| Kanban view | `ramorie kanban` |
| Stats | `ramorie stats` |
| Activity / burndown | `ramorie activity [--burndown]` |
| Save memory | `ramorie remember "<content>"` |
| Search memories | `ramorie find "<term>"` |
| List MCP tools | `ramorie mcp tools` |
| MCP config snippet | `ramorie mcp config` |

---

*Ramorie MCP v6.0.0. Source: https://github.com/kutbudev/ramorie-cli*
