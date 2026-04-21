# Procedural Memory / Skills System

Ramorie supports **procedural memory** (skills) through the `remember`,
`memory`, `find`, and `recall` tools. A skill is a memory whose content
captures a reusable procedure, pattern, or workflow the agent has learned.

Unlike regular memories (which store facts), skills store **HOW to do things**.

> **v5.0.0:** The old top-level `skill` tool has been removed. Create skills
> with `remember` (auto type detection) or generate them from prior activity
> with `memory(action="generate", goal="...")`.

## Overview

| Aspect | Regular Memory | Skill Memory |
|--------|---------------|--------------|
| Purpose | Facts, decisions, preferences | Procedures, workflows, patterns |
| Type | `general`, `bug_fix`, etc. (auto-detected) | `skill` (auto-detected from step-like content) |
| Content | What happened, what was decided | How to do something step-by-step |
| Retrieval | `recall` / `find` with any term | `find` with topic, or `memory(action="list", type="skill")` |

## Creating Skills

### Option A — `remember` (preferred for ad-hoc capture)

```json
{
  "tool": "remember",
  "arguments": {
    "project": "my-project",
    "content": "When deploying to production: 1) Run full test suite with `yarn test` 2) Check for pending migrations with `yarn migrate:status` 3) Build production bundle with `yarn build` 4) Deploy with `yarn deploy:prod` 5) Verify health endpoint responds 200"
  }
}
```

The server auto-detects step-like content and tags it as a `skill`.

### Option B — `memory(action="generate")` (synthesize from activity)

Ask the server to distill a skill from recent task notes + memories:

```json
{
  "tool": "memory",
  "arguments": {
    "action": "generate",
    "goal": "How to deploy this project to production safely"
  }
}
```

### Skill Content Best Practices

Write skills as clear, actionable procedures:

1. **Start with a trigger condition** — When should this skill be applied?
2. **List steps in order** — What actions to take, in sequence
3. **Include specific commands** — Exact commands, paths, or code patterns
4. **Note edge cases** — What to watch for or avoid
5. **End with verification** — How to confirm success

### Good Skill Examples

**Deployment Procedure:**
```
When deploying to production: 1) Run `yarn test --ci` and ensure all pass 2) Run `yarn build` and check bundle size < 500KB 3) Run `yarn deploy:prod --dry-run` first 4) If dry-run passes, run `yarn deploy:prod` 5) Verify: curl https://api.example.com/health returns 200
```

**Bug Investigation Pattern:**
```
When investigating a runtime error in the API: 1) Check logs with `kubectl logs -f deployment/api --tail=100` 2) Look for stack traces and note the originating file 3) Check if the error correlates with recent deployments via `git log --since=1.hour` 4) Reproduce locally with `yarn dev` and the same request payload 5) Write a failing test before fixing
```

**Code Review Checklist:**
```
When reviewing a PR for this project: 1) Check that new files follow the /src/{feature}/ directory structure 2) Verify all new API endpoints have validation middleware 3) Ensure database queries use parameterized inputs (no string interpolation) 4) Check that error responses use the standard ErrorResponse format 5) Verify tests cover both happy path and at least one error case
```

**Git Workflow:**
```
When creating a feature branch: 1) Pull latest main: `git checkout main && git pull` 2) Create branch: `git checkout -b feat/TICKET-description` 3) Make changes in small, atomic commits 4) Before PR: rebase on main with `git rebase main` 5) Push and create PR with the template
```

## Retrieving Skills

### List all skills for a project

```json
{
  "tool": "memory",
  "arguments": {
    "action": "list",
    "project": "my-project",
    "type": "skill"
  }
}
```

### Search skills by topic (fuzzy)

```json
{
  "tool": "find",
  "arguments": {
    "query": "deploy production",
    "type": "skill"
  }
}
```

### Search skills with minimum relevance

```json
{
  "tool": "find",
  "arguments": {
    "query": "database migration",
    "type": "skill",
    "min_score": 0.7
  }
}
```

### Exact/lexical search

```json
{
  "tool": "recall",
  "arguments": {
    "term": "yarn deploy:prod",
    "type": "skill"
  }
}
```

## Agent Usage Patterns

### Learning a new skill

When the agent completes a multi-step procedure successfully, store it:

```
Agent completes deployment successfully
  → remember("When deploying: 1) ... 2) ... 3) ...")
```

### Applying a learned skill

Before starting a known procedure, check for relevant skills:

```
User asks: "Deploy to production"
  → find(query="deploy production", type="skill")
  → If found: follow the stored procedure
  → If not found: ask the user for steps, then remember() the result
```

### Evolving skills

When a procedure changes (new step, tool swap), create a new memory with
the updated content. The older one remains in history; `find` / `recall`
return the most recent match first by default.

## FAQ

**Q: How is a skill different from a regular memory with type=pattern?**
A: A `pattern` describes a recurring observation ("X tends to happen when..."). A `skill` describes an actionable procedure ("When X happens, do Y then Z"). The server auto-tags step-numbered content as `skill`.

**Q: Should I store every successful procedure as a skill?**
A: Store procedures that are likely to be repeated and have more than 2-3 steps. Simple one-liners don't need to be skills.

**Q: What about skills that apply across projects?**
A: Skills are project-scoped by default. For cross-project skills, store them in a shared/common project or omit the `project` argument to save them against the user's default scope.

**Q: Can skills reference other skills?**
A: Not directly in content. Use the knowledge graph via the `entity` tool (`action="link"`) to connect related skills.
