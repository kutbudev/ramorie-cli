# Changelog

All notable changes to the Ramorie CLI are documented in this file.

The format is loosely based on [Keep a Changelog](https://keepachangelog.com/),
and this project follows [Semantic Versioning](https://semver.org/).

## [8.0.0] — 2026-05-11

### BREAKING

- **Encryption enforcement moved from user-level to project-level.** Previously
  the `users.encryption_enabled` flag forced every write to be encrypted; this
  blocked CLI/MCP agents from saving rootless memories (no cwd match → no
  project → unconditional 400). The new model uses `projects.encryption_required`
  so users can keep account-wide encryption ON while opting specific projects
  (notably the auto-created `workflow` scratch project) OUT. Behavior change for
  third-party API consumers: a 400 `ENCRYPTION_REQUIRED` response now depends
  on the target project, not the account. Requires backend migration 079
  applied alongside this CLI release.
- **Rootless writes no longer fail with "project is required".** `auto_remember`,
  `remember`, and other tools that fall through to `resolveProjectWithOrg`
  now create-or-reuse a personal `workflow` project on the fly. Callers that
  relied on the old error message to detect "no project selected" must check
  for an explicit project ID in the response instead.

### Added

- `Project.EncryptionRequired` field on the CLI model surface (mirrors the
  backend `projects.encryption_required` column).
- `api.Client.EnsureWorkflowProject()` — idempotent bootstrap that POSTs the
  personal `workflow` scratch project with `encryption_required=false`, and
  recovers from 409 Conflict by listing personal projects.
- `detectGitRemoteRepo()` in `internal/mcp/tools.go` — bounded (500ms ctx
  timeout) helper that extracts the repo name from `git config --get
  remote.origin.url`. Used by `resolveProjectID` / `resolveProjectWithOrg`
  as a "Try 2.5" between cwd fuzzy match and single-project fallback. The
  hard timeout protects against LuLu/firewall hangs (see memory ref
  `6c32b7d1`).
- `resolveProjectID` / `resolveProjectWithOrg` now have a "Try 4" workflow
  scratch fallback: if every other auto-detect step fails, resolve to (or
  auto-create) the personal `workflow` project instead of erroring.

### Changed

- `cmd/ramorie/main.go::Version` bumped to `8.0.0`.
- `npm/package.json::version` bumped to `8.0.0`.
- `handleAutoRemember` no longer short-circuits with a "project is required"
  error when cwd detection fails — the resolver's workflow fallback takes
  over.

## [7.1.0] — 2026-05-11

### Added
- `ramorie setup hooks ...` alias for the existing `ramorie setup-hooks ...`
  command tree. Either invocation works; doc strings reference both.
- `CHANGELOG.md` (this file) — backfilled with the v7.0.x entries below.

### Changed
- `ramorie hook ...` is now formally marked **deprecated** in its help output.
  Prefer `ramorie setup-hooks install` (or the new `ramorie setup hooks install`
  alias), which covers Claude Code, Codex, Cursor, and Windsurf in one call.
  The legacy single-client command is kept registered for backward compat with
  pinned scripts.
- `withProtocolReminder` in `internal/mcp/tools.go` carries a clearer doc
  comment: it mutates the result slot in place; callers must invoke it only
  once per `CallToolResult`.
- `internal/rules/cursor.go::RulesPath()` documents its `os.Getwd()` fallback
  — when called outside a project root it writes to the current working
  directory. `Detect()` is the intended gate before `RulesPath()`.
- Frontend `vitest.setup.tsx` `react-i18next` global mock now accepts the
  full i18next options-object signature (`t("k", { defaultValue, ...vars })`)
  in addition to the legacy string shortcut (`t("k", "default")`). Variable
  interpolation (`{{name}}`) is supported.
- `internal/mcp/register_v4.go` tool-count comment corrected to match the
  current 15-tool registration (7 core + 4 common + 4 advanced).

### Notes
- No new MCP tools, no schema changes, no breaking API changes.
- Minor cleanup-focused release; safe drop-in for any v7.x deployment.

## [7.0.2] — 2026-04 (prior)

### Changed (Breaking-ish)
- **`auto_remember` semantic-find gate.** Before falling back to local Jaccard
  similarity, `auto_remember` now runs the backend `find` pipeline (cosine +
  rerank, FastMode) and treats a score ≥ 0.75 as a semantic duplicate. This
  catches paraphrased duplicates that Jaccard alone missed (smoke test:
  0.867 cosine match where Jaccard reported 0.0). The envelope's
  `match_source` field reports `"semantic_find"` vs `"jaccard"` so callers
  can observe the path taken.

## [7.0.1] — 2026-04 (prior)

### Changed
- **Jaccard duplicate threshold lowered from 0.85 → 0.60.** Production smoke
  tests showed real duplicates slipping through at 0.77 with the old cutoff;
  0.60 is the empirically-tuned value. Distinct lowercase token-set overlap.

## [7.0.0] — 2026-04 (prior)

### Added (Breaking)
- **MCP protocol hardening.** Every find/recall response now carries
  `_meta.protocol_reminder` so the agent sees the next-required-action
  nudge inline with every result. Clients that strictly validate the
  response envelope shape must allow the additional `_meta` keys.
- **One-command setup.** New `ramorie setup` orchestrator runs auth →
  MCP install → hooks → rules → vault unlock → diagnostics in a single
  invocation. Detects every supported client (Claude Code, Claude
  Desktop, Codex, Cursor, Windsurf, VS Code, Zed) automatically.
- **`ramorie setup-hooks`** unified surface for hook-capable clients
  (Claude Code, Codex) and rules-only clients (Cursor, Windsurf). The
  older `ramorie hook ...` command remains registered but is now
  considered legacy.

### Removed
- Active-state MCP tools that didn't match how agents actually work:
  `decision` (use `remember`/`memory` with `type=decision`), `skill`
  (use `remember`/`memory` with `type=skill`), and `manage_focus`
  (focus/active-context concept eliminated).
- `admin.switch_org` action — active-organization concept eliminated;
  scope is now always resolved per-call.

### Breaking Changes Summary
- Agents that hardcoded `_meta` to be absent must accept the new key.
- Agents that called the removed `decision` / `skill` / `manage_focus`
  tools must migrate to `remember` / `memory` / no-op respectively.
- Operators that scripted `admin.action=switch_org` must drop the call;
  resolve `project`/`org_id` per-call instead.
