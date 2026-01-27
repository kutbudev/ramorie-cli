package mcp

// AgentDirectives contains the auto-directive system for AI agents
// This instructs agents to proactively use Ramorie for memory, tasks, and recall
type AgentDirectives struct {
	// Structured triggers - machine readable
	Triggers TriggerSet `json:"triggers"`

	// Natural language prompt - agent readable
	SystemPrompt string `json:"system_prompt"`

	// Quick reference
	Cheatsheet string `json:"cheatsheet"`

	// Multi-agent awareness
	MultiAgentNote string `json:"multi_agent_note"`
}

// TriggerSet contains all trigger categories
type TriggerSet struct {
	Memory []TriggerRule `json:"memory"`
	Task   []TriggerRule `json:"task"`
	Recall []TriggerRule `json:"recall"`
}

// TriggerRule defines when an action should be automatically taken
type TriggerRule struct {
	Condition string   `json:"condition"`
	Action    string   `json:"action"`
	Examples  []string `json:"examples"`
}

// GetAgentDirectives returns the complete directive system for AI agents
func GetAgentDirectives() AgentDirectives {
	return AgentDirectives{
		Triggers: TriggerSet{
			Memory: []TriggerRule{
				{
					Condition: "User states a preference or configuration choice",
					Action:    "remember(content) immediately",
					Examples: []string{
						"User: 'I prefer TypeScript over JavaScript' → save as memory",
						"User: 'Use dark mode for all UI components' → save as memory",
						"User: 'Always use yarn instead of npm' → save as memory",
					},
				},
				{
					Condition: "Bug is solved with a specific solution",
					Action:    "remember(problem + solution)",
					Examples: []string{
						"Fixed CORS issue by adding proxy config → save problem AND solution",
						"Resolved 'module not found' with yarn install → save for future reference",
						"Database connection timeout fixed with pool settings → save as memory",
					},
				},
				{
					Condition: "Architectural or technical decision is made",
					Action:    "remember(decision + rationale)",
					Examples: []string{
						"User: 'Use Redis instead of PostgreSQL for sessions' → save decision",
						"Decided to use WebSocket instead of polling → save with reasoning",
						"Chose Tailwind CSS over styled-components → save decision and why",
					},
				},
				{
					Condition: "New pattern or best practice is discovered",
					Action:    "remember(pattern description)",
					Examples: []string{
						"Found effective error handling pattern → save for reuse",
						"Learned optimal database indexing strategy → save as memory",
						"Discovered performance optimization technique → save as memory",
					},
				},
			},
			Task: []TriggerRule{
				{
					Condition: "User explicitly requests work to be done",
					Action:    "create_task(project, description) immediately",
					Examples: []string{
						"User: 'Add a logout button' → create task",
						"User: 'Fix the login bug' → create task",
						"User: 'Implement dark mode' → create task",
					},
				},
				{
					Condition: "Work is deferred to later",
					Action:    "create_task(project, deferred_work)",
					Examples: []string{
						"User: 'We'll handle this later' → create task",
						"User: 'Let's postpone the refactoring' → create task",
						"'This can wait until next sprint' → create task",
					},
				},
				{
					Condition: "TODO/FIXME comment found in code",
					Action:    "create_task(project, todo_description)",
					Examples: []string{
						"Found '// TODO: add validation' → create task",
						"Found '// FIXME: handle edge case' → create task",
						"Found '// HACK: temporary solution' → create task for proper fix",
					},
				},
				{
					Condition: "Dependency or prerequisite is identified",
					Action:    "create_task(project, prerequisite)",
					Examples: []string{
						"'We need to set up the database first' → create task",
						"'Authentication must be implemented before this' → create task",
						"'This requires the API to be ready' → create task",
					},
				},
			},
			Recall: []TriggerRule{
				{
					Condition: "Before answering any question",
					Action:    "recall(relevant_terms, project) FIRST",
					Examples: []string{
						"User asks about auth → recall 'authentication' first",
						"User asks about API → recall 'api endpoint' first",
						"User mentions a feature → recall related memories first",
					},
				},
				{
					Condition: "Encountering an error or bug",
					Action:    "recall(error_keywords) to find past solutions",
					Examples: []string{
						"CORS error → recall 'cors' for previous solutions",
						"Build failure → recall 'build error' for past fixes",
						"Type error → recall 'typescript error' for similar issues",
					},
				},
				{
					Condition: "Before making a technical decision",
					Action:    "recall(decision_topic) to check previous decisions",
					Examples: []string{
						"Choosing database → recall 'database decision' first",
						"Selecting framework → recall 'framework choice' first",
						"Architecture question → recall 'architecture decision' first",
					},
				},
				{
					Condition: "Starting work on new project/technology",
					Action:    "recall(project_name OR technology) for context",
					Examples: []string{
						"Starting React work → recall 'react' for project conventions",
						"New feature in Orkai → recall 'orkai' for existing patterns",
						"Working with API → recall 'api' for established standards",
					},
				},
			},
		},
		SystemPrompt: `## Ramorie Memory Protocol

You are connected to Ramorie, a persistent memory system shared across AI agents.

### PROACTIVE BEHAVIOR (No need to ask user)

**AUTO-SAVE as Memory:**
- User states a preference → save immediately
- Bug solved → save problem + solution
- Architectural decision made → save with rationale
- New pattern learned → save for future reference

**AUTO-CREATE as Task:**
- User says "do X", "add Y" → create task
- Work deferred "later", "tomorrow" → create task
- Found TODO/FIXME in code → create task
- Dependency identified → create prerequisite task

**ALWAYS RECALL FIRST:**
- Before answering questions → recall relevant context
- On errors → recall similar past solutions
- Before decisions → recall previous decisions
- New project/tech → recall related memories

### MULTI-AGENT AWARENESS

You are ONE of potentially MULTIPLE agents working simultaneously.
- Another agent may have just added relevant context
- ALWAYS recall before acting - data is real-time
- Check for duplicates before saving memories
- Your memories help other agents too

### ANTI-PATTERNS (Don't do these)
- Don't ask "should I save this?" - just save it
- Don't save trivial/temporary information
- Don't create duplicate memories - recall first
- Don't forget to specify project parameter`,

		Cheatsheet: `
┌─────────────────────────────────────────────────┐
│ RAMORIE QUICK REFERENCE                         │
├─────────────────────────────────────────────────┤
│ SAVE: remember(content)              │
│ FIND: recall(term, project?)                    │
│ TASK: create_task(project, description)         │
│ LIST: list_memories(project) / list_tasks(...)  │
├─────────────────────────────────────────────────┤
│ Always recall BEFORE answering                  │
│ Save decisions, solutions, preferences          │
│ Create tasks for deferred work                  │
│ Other agents can see your memories              │
└─────────────────────────────────────────────────┘`,

		MultiAgentNote: `IMPORTANT: Multiple AI agents may be connected simultaneously.
- Agent in Tab 1 saves a memory → Agent in Tab 2 can recall it immediately
- Always recall() before starting work to get latest context
- Avoid duplicate memories - check before saving
- Your work builds shared knowledge for all agents`,
	}
}

// GetDirectivesAsMap converts directives to map for JSON response
func GetDirectivesAsMap() map[string]interface{} {
	d := GetAgentDirectives()

	memoryTriggers := make([]map[string]interface{}, len(d.Triggers.Memory))
	for i, t := range d.Triggers.Memory {
		memoryTriggers[i] = map[string]interface{}{
			"condition": t.Condition,
			"action":    t.Action,
			"examples":  t.Examples,
		}
	}

	taskTriggers := make([]map[string]interface{}, len(d.Triggers.Task))
	for i, t := range d.Triggers.Task {
		taskTriggers[i] = map[string]interface{}{
			"condition": t.Condition,
			"action":    t.Action,
			"examples":  t.Examples,
		}
	}

	recallTriggers := make([]map[string]interface{}, len(d.Triggers.Recall))
	for i, t := range d.Triggers.Recall {
		recallTriggers[i] = map[string]interface{}{
			"condition": t.Condition,
			"action":    t.Action,
			"examples":  t.Examples,
		}
	}

	return map[string]interface{}{
		"triggers": map[string]interface{}{
			"memory": memoryTriggers,
			"task":   taskTriggers,
			"recall": recallTriggers,
		},
		"system_prompt":    d.SystemPrompt,
		"cheatsheet":       d.Cheatsheet,
		"multi_agent_note": d.MultiAgentNote,
	}
}
