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

## Context Packs (the "Gemini Gem" pattern, v6.6.0+)

Context packs group related memories + tasks + contexts into a curated
bundle. The agent loads the entire bundle in **one tool call** instead
of issuing 5–10 ad-hoc `find()` queries — same idea as Gemini Gems or
ChatGPT custom GPTs, but built from your own Ramorie data.

**MCP tools (v6.6.0+):**
- `load_context_pack(pack_id, format?, budget_tokens?, sections?)` — 🔵 ESSENTIAL.
  Loads the assembled, token-budgeted bundle into the agent context. Use
  at session start with a known scope.
- `list_context_packs(...)` — discover packs by type/status/query.
- `manage_context_pack(action, ...)` — create / update / link / clone.
  Bulk actions: `link_memories`, `link_tasks`, `unlink_memories`,
  `unlink_tasks`, `clone`, `set_active`.
- `get_context_pack(packId)` — pack details (members + metadata).
- `export_context_pack` / `import_context_pack` — portable JSON bundles.

**Resource templates (sidebar-aware clients):**
- `ramorie://context-packs/{id}` — pack details (JSON)
- `ramorie://context-packs/{id}/assembled` — XML bundle ready for context

**Workflow — when to use `load_context_pack`:**
1. **Session start with a known scope** — `load_context_pack(pack_id="auth-refactor")` once. The bundle replaces 5–10 individual `find()` calls.
2. **Switching scopes** — `manage_context_pack(action="set_active", ...)` then `load_context_pack` again.
3. **After remembering important facts** — `manage_context_pack(action="link_memories", memoryIds=[...])` to keep the pack current.
4. **Don't** — call `load_context_pack` mid-task for unrelated topics; use `find()` for ad-hoc queries.

**CLI:**
```bash
ramorie pack list
ramorie pack create "feature-auth"
ramorie pack use auth-refactor                    # assemble + stdout
ramorie pack add <pack> --memory id1 id2 --task id3
ramorie pack remove <pack> --memory id1
ramorie pack clone <pack> --name "billing-flow"
ramorie pack render <pack> --format json --budget 8000
```

**Encryption:** server never decrypts (zero-knowledge). Encrypted
memories/tasks come back as envelopes (id + `kind="encrypted"`) so the
agent knows the row exists; the CLI/client decrypts via vault key when
needed. `_meta.items_skipped_encrypted` reports the skip count.

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
| Unlock vault | `ramorie unlock` (alias of `setup vault unlock`, since v6.3.5) |
| Lock vault | `ramorie lock` (alias of `setup vault lock`, since v6.3.5) |
| Vault status | `ramorie setup vault status` |
| Create project | `ramorie project create <name>` |
| List projects | `ramorie project list` |
| Create task | `ramorie task create "<desc>"` |
| Start task | `ramorie task start <id>` |
| Complete task | `ramorie task complete <id>` |
| Kanban view | `ramorie kanban` |
| Stats | `ramorie stats` (auto-JSON when piped) |
| Activity / burndown | `ramorie activity [--burndown]` (auto-JSON when piped) |
| Save memory (positional) | `ramorie remember "<content>" -p "<project name>"` |
| Save memory (stdin pipe) | `echo "<content>" \| ramorie remember -p "<project>"` |
| Save memory (JSON + tags) | `cat memo.md \| ramorie remember -p ramorie-cli -t cli,docs --json` |
| Search memories | `ramorie find "<term>"` |
| List MCP tools | `ramorie mcp tools` |
| MCP config snippet | `ramorie mcp config` |

> `ramorie remember` (v6.4.0+) accepts content via positional args **or**
> piped stdin, takes a project **name** (not just UUID), supports
> comma-separated `-t/--tags`, and prints structured JSON with `--json`
> for agents/scripts.

---

*Ramorie MCP v6.0.0. Source: https://github.com/kutbudev/ramorie-cli*
