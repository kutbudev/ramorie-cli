# Ramorie MCP Agent Guide

> Quick reference for AI agents using the Ramorie MCP server (v7.0.0, 16 tools).
> For full CLI docs, see https://ramorie.com/docs/cli or run `ramorie mcp tools`.

## Setup

The fastest path is the one-command installer — it handles auth, MCP config
for every detected client (Claude Code, Codex, Cursor, Windsurf, VS Code,
Zed), and the new Persistent Memory Protocol hooks/rules in a single step:

```bash
ramorie setup               # full install: auth + MCP + hooks + rules + vault + doctor
ramorie doctor              # re-run the health check at any time
ramorie setup-hooks status  # see which clients have the protocol installed
# v7.1.0: `ramorie setup hooks status` works too — either command form is supported.
```

The legacy interactive picker is still available via `ramorie setup --legacy`.

Manual MCP config (if you'd rather wire it yourself):

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

### v7.0.0 — Persistent Memory Protocol hardening

PR10 introduces a stronger protocol surface so agents stop skipping
`remember()` after sub-agent returns:

- **`auto_remember` tool** (#16) — atomic `find()`+`remember()` with
  similarity gating. Use this instead of separate `find`+`remember` calls
  whenever a sub-agent returns or a decision/bug-fix/preference lands.
- **`_meta.protocol_reminder`** — every tool response now carries an
  op-specific nudge so the protocol fires mid-turn, not just at session
  start.
- **Hook installer** — `ramorie setup-hooks install` writes 4 hooks into
  Claude Code (`~/.claude/settings.json`) and Codex (`~/.codex/hooks.json`):
  `SessionStart`, `PostToolUse(Agent)`, `SubagentStop`, `Stop`.
- **Rules-file installer** — for editors without hook support (Cursor,
  Windsurf), the same protocol text is written into a managed markdown
  block inside `.cursor/rules/ramorie-memory-protocol.mdc` and
  `~/.codeium/windsurf/memories/global_rules.md`.
- **`ramorie doctor`** — health check for config / vault / MCP / hooks /
  rules surfaces. Exits 1 on any ✗ result so it's CI-friendly.

> **v6.x:** v5.0.0'dan beri legacy tools (`create_task`, `add_memory`, `search_memories`,
> `create_decision`, `activate_context_pack`, `get_active_task`, `manage_focus`,
> `skill`, `decision`) kaldırıldı. Use the 15 tools below. Restart your MCP client
> (Claude Code / Cursor / Windsurf) after upgrading from v5.x.
>
> **v6.9.0 (PR6):** New `load_skill` tool — Claude Code-format skill
> rendering on demand. Mirrors `load_context_pack`: one call returns
> the procedural memory as ready-to-apply markdown.

---

## The 15 Tools

### Core (always use)

| # | Tool | Purpose |
|---|------|---------|
| 1 | `setup_agent` | Initialize session, auto-detect project from cwd. **Call this first.** |
| 2 | `list_projects` | List personal + org projects. |
| 3 | `remember` | Store a memory. Auto-detects type. Prefix with `todo:` to create a task instead. |
| 4 | `find` | Hybrid semantic + lexical search (HyDE + rerank). Best for fuzzy queries. |
| 5 | `recall` | FTS search; pass `precision: true` to route to `find`. |
| 6 | `task` | Unified task ops. `action`: `list` / `get` / `create` / `start` / `complete` / `stop` / `progress` / `note` / `move`. |
| 7 | `load_skill` | Load a skill into context (frontmatter + markdown body, ready to apply). |

### Common

| # | Tool | Purpose |
|---|------|---------|
| 8 | `memory` | Unified memory ops (`list`, `get`), or skill generation via `goal` param. |
| 9 | `get_stats` | Task counts per project / status. |
| 10 | `get_agent_activity` | Agent timeline query. |
| 11 | `surface_context` | File/domain-scoped context surfacing for the active task. |

### Advanced

| # | Tool | Purpose |
|---|------|---------|
| 12 | `create_project` | Create a project. |
| 13 | `manage_subtasks` | Subtask CRUD. |
| 14 | `entity` | Knowledge graph (10 actions: `create`, `link`, `list`, `search`, etc.). |
| 15 | `admin` | `consolidate` / `cleanup` / `orgs` / `export` / `import` / `plan` / `analyze`. |

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

### Loading a skill into context (v6.9.0+)

Use `load_skill(skill_id)` when the agent needs the **full procedural
markdown** ready to apply — same shape Claude Code expects from a
`.claude/skills/<name>/SKILL.md` file. Counterpart to
`load_context_pack`: one tool call, instruction inlined.

```
load_skill(skill_id="deploy-prod")
# returns: (1) markdown body (frontmatter + steps) — read as instruction
#          (2) JSON envelope { skill, source, _meta } for tooling
```

`skill_id` accepts a UUID or a unique skill name. Ambiguous names
error out so the agent can disambiguate via `recall(type="skill")`.

**Workflow:**
1. `recall(query="deploy", type="skill")` or `find("deploy prod")` → discover skill ids
2. `load_skill(skill_id="<id-or-name>")` → render the skill into context
3. Apply the steps verbatim. Skip `load_skill` for trivia queries — use `find()`.

**CLI:**
```bash
ramorie skill use deploy-prod                # markdown body to stdout
ramorie skill use deploy-prod --json         # full response JSON
ramorie skill use deploy-prod > SKILL.md     # snapshot to file
ramorie skill use deploy-prod | pbcopy       # paste into chat
```

### Filesystem sync (v6.9.0+)

Round-trip skills between Ramorie and `~/.claude/skills/`. Ramorie is the
source of truth; filesystem mirrors it. Hash-based idempotency means
re-running these commands is safe.

```bash
ramorie skill upload ~/.claude/skills/foo/SKILL.md     # single file → Ramorie
ramorie skill upload ./bar.md --overwrite              # force replace on name collision
ramorie skill sync                                     # bulk push ~/.claude/skills/
ramorie skill sync --dir .claude/skills/ --overwrite   # project-level skills
ramorie skill pull                                     # Ramorie → ~/.claude/skills/
ramorie skill pull --dry-run                           # plan-only (no writes)
ramorie skill diff                                     # +/- / ~ / = report
ramorie skill diff -v                                  # also show in-sync rows
```

---

## Context Packs (the "Gemini Gem" pattern, v6.6.0+ / manifest mode v6.7.0+)

Context packs group related memories + tasks + contexts into a curated
bundle. Same idea as Gemini Gems or ChatGPT custom GPTs, but built from
your own Ramorie data.

**Two access modes (PR5, v6.7.0):**

The MCP tool exposes a **manifest mode (default)** that mirrors how
Claude Code already handles large codebases — `Glob` to list, `Read`
on demand. You never load the universe into context.

| Mode | What returns | Cost | When |
|------|--------------|------|------|
| `manifest` (default) | item titles + tokens + tags + status, **no bodies** | ~500-1500 tokens for 50 items | session start, scope discovery |
| `full` | rendered XML/JSON/MD bundle with previews | proportional to pack size | small packs, single-shot use |

**Manifest workflow (recommended)**:
1. `load_context_pack(pack_id="auth-refactor")` → manifest of items
2. Agent reads titles + tags, decides which 3-5 items it actually needs
3. `get_memory(id)` / `get_task(id)` for each chosen item — single body
   per call, no waste
4. Done. Total context cost = manifest (~1K) + selected items (~3-5K)
   instead of full pack (~30-50K).

This is exactly the pattern Claude Code uses against your filesystem:
`Glob("*.tsx")` returns 200 paths, then `Read("path")` opens just the
files actually touched.

**MCP tools:**
- `load_context_pack(pack_id, mode?, format?, budget_tokens?, sections?)` — 🔵 ESSENTIAL.
  Default `mode="manifest"` returns body-less index. Use `mode="full"`
  only for small packs when you really want everything inline.
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
1. **Session start with a known scope** — `load_context_pack(pack_id="auth-refactor")` returns manifest. Pick the items you need; fetch with `get_memory(id)` / `get_task(id)`.
2. **Switching scopes** — `manage_context_pack(action="set_active", ...)` then `load_context_pack` again.
3. **After remembering important facts** — `manage_context_pack(action="link_memories", memoryIds=[...])` to keep the pack current.
4. **Don't** — call `load_context_pack` mid-task for unrelated topics; use `find()` for ad-hoc queries.
5. **Don't fetch everything** — manifest tells you tokens per item. Skip 8K-token memories if you only need a 500-token answer.
6. **Don't re-load the same pack** — manifest is deterministic for a given pack id; the agent already has it in earlier context.

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

*Ramorie MCP v6.9.0. Source: https://github.com/kutbudev/ramorie-cli*
