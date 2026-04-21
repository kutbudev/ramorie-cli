# Ramorie MCP - Cursor Rules

> Bu dosyayı `.cursorrules` olarak projenize kopyalayın.

## MCP Server Yapılandırması

Ramorie MCP server'ı Cursor'da kullanmak için `.cursor/mcp.json` dosyasına ekleyin:

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

---

## Core Principles

### 1. Context-First Approach
Always start by initializing the session:
```
setup_agent → Initialize session and get project context
task(action="list") → See pending work
```

### 2. Task-Driven Development
Every piece of work should be tracked:
```
create_task → Start new work
start_task → Begin working
add_task_note → Log progress
complete_task → Finish work
```

### 3. Knowledge Persistence
Save everything valuable:
```
add_memory → Store learnings
recall → Search existing knowledge
create_decision → Record important decisions
```

---

## Workflow Rules

### Starting New Work
```
1. Check context: get_active_context_pack
2. Create task: create_task with clear description
3. Start task: start_task (sets as active)
4. Plan: add_task_note with initial approach
```

### During Development
```
- Log progress: add_task_note after each milestone
- Update progress: update_progress (0-100%)
- Save learnings: add_memory for reusable info
- Record decisions: create_decision for architectural choices
```

### Completing Work
```
1. Final note: add_task_note with summary
2. Complete: complete_task
3. Optional: add_memory for key learnings
```

---

## Memory Bank Usage

### When to Save Memory
- Code patterns that work well
- Configuration snippets
- API endpoints and their usage
- Error solutions
- Performance optimizations

### Memory Format
```
Good: "PostgreSQL JSONB indexing: CREATE INDEX idx_data ON table USING GIN (data jsonb_path_ops); Improves query performance 10x for JSON searches."

Bad: "db index stuff"
```

### Searching Memory
Before asking the user:
```
recall "relevant keywords" → Check if answer exists
```

---

## Decision Recording (ADR)

### When to Record
- Architecture changes
- Technology choices
- API design decisions
- Security policies
- Performance trade-offs

### Decision Format
```json
{
  "title": "Clear, descriptive title",
  "description": "Brief summary",
  "area": "Frontend|Backend|Architecture|DevOps|Security",
  "status": "draft|proposed|approved|deprecated",
  "context": "Why this decision was made",
  "consequences": "Impact and trade-offs"
}
```

---

## Context Management

### Active Context = Current Focus
- Each project/feature should have its own context pack
- Switch context when changing focus
- Context helps filter relevant tasks and memories

### Switching Context
```
1. stop_task → Pause current work
2. activate_context_pack → Switch to new context
3. get_next_tasks → See pending tasks
4. start_task → Begin new task
```

---

## Quick Reference

| Action | Tool | Required Params |
|--------|------|-----------------|
| New task | `create_task` | description |
| Start work | `start_task` | taskId |
| Log progress | `add_task_note` | taskId, note |
| Update % | `update_progress` | taskId, progress |
| Complete | `complete_task` | taskId |
| Save info | `add_memory` | content |
| Search info | `recall` | term |
| Record decision | `create_decision` | title |
| Switch context | `activate_context_pack` | packId |

---

## Anti-Patterns

### Don't Do This
1. ❌ Working without creating a task
2. ❌ Long sessions without logging progress
3. ❌ Skipping decision documentation
4. ❌ Leaving memories without tags
5. ❌ Starting work without checking context

### Do This Instead
1. ✅ Always create_task before starting
2. ✅ add_task_note after each milestone
3. ✅ create_decision for important choices
4. ✅ add_memory with descriptive content
5. ✅ get_active_context_pack at session start

---

## Progress Tracking

| Percentage | Stage |
|------------|-------|
| 0% | Not started |
| 25% | Planning/Research |
| 50% | Implementation started |
| 75% | Testing phase |
| 100% | Completed |

---

## Example Session

```
// Session start
setup_agent → session ready, project auto-detected
task(action="list") → no pending work

// New work
create_task "Implement password reset flow"
start_task "abc123"
add_task_note "abc123" "Plan: 1. Email service 2. Token generation 3. Reset endpoint"

// Progress
add_task_note "abc123" "Email service integrated with SendGrid"
update_progress "abc123" 25
add_memory "SendGrid API: Use v3/mail/send endpoint, requires API key in Authorization header"

// Decision
create_decision {
  title: "SendGrid for transactional emails",
  area: "Backend",
  context: "Need reliable email delivery, SendGrid has good deliverability",
  consequences: "Monthly cost ~$15, vendor lock-in for email templates"
}

// Complete
add_task_note "abc123" "Password reset flow complete with tests"
update_progress "abc123" 100
complete_task "abc123"
```

---

*Ramorie MCP v1.7.0 - 57 tools available*
