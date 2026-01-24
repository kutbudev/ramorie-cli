# Procedural Memory / Skills System

Ramorie supports **procedural memory** (skills) through the existing `add_memory` and `recall` tools. A skill is a special memory type that stores reusable procedures, patterns, or workflows that the agent has learned.

Unlike regular memories (which store facts), skills store **HOW to do things**.

## Overview

| Aspect | Regular Memory | Skill Memory |
|--------|---------------|--------------|
| Purpose | Store facts, decisions, preferences | Store procedures, workflows, patterns |
| Type | `general`, `decision`, `bug_fix`, etc. | `skill` |
| Content | What happened, what was decided | How to do something step-by-step |
| Retrieval | `recall` with any term | `recall` with `type=skill` filter |

## Creating Skills

Use `add_memory` with `type=skill`:

```json
{
  "tool": "add_memory",
  "arguments": {
    "project": "my-project",
    "content": "When deploying to production: 1) Run full test suite with `yarn test` 2) Check for pending migrations with `yarn migrate:status` 3) Build production bundle with `yarn build` 4) Deploy with `yarn deploy:prod` 5) Verify health endpoint responds 200",
    "type": "skill"
  }
}
```

### Skill Content Best Practices

Write skills as clear, actionable procedures:

1. **Start with a trigger condition** - When should this skill be applied?
2. **List steps in order** - What actions to take, in sequence
3. **Include specific commands** - Exact commands, paths, or code patterns
4. **Note edge cases** - What to watch for or avoid
5. **End with verification** - How to confirm success

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

### List All Skills for a Project

```json
{
  "tool": "recall",
  "arguments": {
    "term": "*",
    "project": "my-project",
    "type": "skill"
  }
}
```

### Search Skills by Topic

```json
{
  "tool": "recall",
  "arguments": {
    "term": "deploy production",
    "type": "skill"
  }
}
```

### Search Skills with Minimum Relevance

```json
{
  "tool": "recall",
  "arguments": {
    "term": "database migration",
    "type": "skill",
    "min_score": 0.7
  }
}
```

## Agent Usage Patterns

### Learning a New Skill

When an agent completes a multi-step procedure successfully, it should store the procedure as a skill:

```
Agent completes deployment successfully
  -> Stores: add_memory(type="skill", content="When deploying: 1) ... 2) ... 3) ...")
```

### Applying a Learned Skill

Before starting a known procedure, the agent should check for relevant skills:

```
User asks: "Deploy to production"
  -> Agent: recall(term="deploy production", type="skill")
  -> If found: Follow the stored procedure
  -> If not found: Ask user for steps, then store as skill
```

### Evolving Skills

When a procedure changes (e.g., new step added, tool changed), create a new skill memory with the updated content. The old one remains in history but the newer one will be returned first by `recall` (sorted by recency).

## Integration with Context Packs

Skills can be added to context packs for organized retrieval:

```json
{
  "tool": "manage_context_pack",
  "arguments": {
    "action": "add_memory",
    "packId": "pack-uuid",
    "memoryId": "skill-memory-uuid"
  }
}
```

This allows grouping related skills (e.g., all deployment skills, all debugging skills) into a single context pack that can be activated with `manage_focus`.

## FAQ

**Q: How is a skill different from a regular memory with type=pattern?**
A: A `pattern` describes a recurring observation or code pattern. A `skill` describes an actionable procedure - specific steps to follow. Use `pattern` for "I noticed X tends to happen" and `skill` for "When X happens, do Y then Z."

**Q: Should I store every successful procedure as a skill?**
A: Store procedures that are likely to be repeated and have more than 2-3 steps. Simple one-liners don't need to be skills.

**Q: What about skills that apply across projects?**
A: Currently skills are project-scoped. If a skill applies broadly, store it in each relevant project or in a shared/common project.

**Q: Can skills reference other skills?**
A: Not directly (no memory-to-memory relations yet). Use descriptive content that mentions related procedures by name so `recall` can find them together.
