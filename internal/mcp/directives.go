package mcp

// AgentDirectives contains the simplified directive system for AI agents
// Reduced from verbose explanations to 3 core rules for better agent compliance
type AgentDirectives struct {
	// The three core rules - machine and agent readable
	Rules []AgentRule `json:"rules"`

	// Behavioral triggers - IF-THEN patterns
	Triggers []BehaviorTrigger `json:"triggers"`

	// Anti-patterns to avoid
	AntiPatterns []string `json:"anti_patterns"`

	// Quick reference for agents
	Cheatsheet string `json:"cheatsheet"`
}

// AgentRule defines a core behavioral rule
type AgentRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	When        string `json:"when"`
	Action      string `json:"action"`
}

// BehaviorTrigger defines an IF-THEN behavioral pattern
type BehaviorTrigger struct {
	If   string `json:"if"`
	Then string `json:"then"`
}

// GetAgentDirectives returns the simplified 3-rule directive system
func GetAgentDirectives() AgentDirectives {
	return AgentDirectives{
		Rules: []AgentRule{
			{
				Name:        "RECALL FIRST",
				Description: "Search memories BEFORE responding to any question or starting any work",
				When:        "Before ANY response, decision, or action",
				Action:      "recall(term: \"relevant_keywords\")",
			},
			{
				Name:        "REMEMBER ALWAYS",
				Description: "Auto-save without asking user permission",
				When:        "Preference stated, bug solved, decision made, pattern learned",
				Action:      "remember(content: \"...\", project: \"...\")",
			},
			{
				Name:        "TASK EVERYTHING",
				Description: "Auto-create tasks from conversation",
				When:        "User says 'do X', 'fix Y', 'later', 'TODO', or defers work",
				Action:      "task(action: \"create\", project: \"...\", description: \"...\")",
			},
		},

		Triggers: []BehaviorTrigger{
			// Recall triggers
			{If: "user_asks_question", Then: "recall() BEFORE responding"},
			{If: "user_mentions_topic", Then: "recall(term: topic) for context"},
			{If: "encountering_error", Then: "recall(term: error_type) for past solutions"},
			{If: "making_decision", Then: "recall(term: topic) to check past decisions"},

			// Remember triggers
			{If: "user_states_preference", Then: "remember() immediately WITHOUT asking"},
			{If: "bug_is_solved", Then: "remember(problem + solution) WITHOUT asking"},
			{If: "decision_is_made", Then: "remember(decision + rationale) WITHOUT asking"},
			{If: "pattern_discovered", Then: "remember(pattern) WITHOUT asking"},

			// Task triggers
			{If: "user_says_do_X", Then: "task(create) immediately"},
			{If: "user_says_fix_Y", Then: "task(create) immediately"},
			{If: "user_says_later", Then: "task(create) to track deferred work"},
			{If: "TODO_found_in_code", Then: "task(create) for the TODO item"},
		},

		AntiPatterns: []string{
			"NEVER ask 'should I save this?' - just save it",
			"NEVER skip recall before answering",
			"NEVER ask 'should I create a task?' - just create it",
			"NEVER wait to be told to use memory - be proactive",
		},

		Cheatsheet: `# RAMORIE - 3 RULES

1. RECALL FIRST
   → recall(term) BEFORE any response

2. REMEMBER ALWAYS
   → remember(content) on preference/decision/solution
   → NO permission needed

3. TASK EVERYTHING
   → task(create) on "do X", "fix Y", "later"
   → NO permission needed

Quick ref:
  recall(term)                    # Search memories
  remember(content, project)      # Save memory
  task(create, project, desc)     # Create task
  projects()                      # List projects`,
	}
}

// GetDirectivesAsMap converts directives to map for JSON response
func GetDirectivesAsMap() map[string]interface{} {
	d := GetAgentDirectives()

	rules := make([]map[string]interface{}, len(d.Rules))
	for i, r := range d.Rules {
		rules[i] = map[string]interface{}{
			"name":        r.Name,
			"description": r.Description,
			"when":        r.When,
			"action":      r.Action,
		}
	}

	triggers := make([]map[string]interface{}, len(d.Triggers))
	for i, t := range d.Triggers {
		triggers[i] = map[string]interface{}{
			"if":   t.If,
			"then": t.Then,
		}
	}

	return map[string]interface{}{
		"rules":         rules,
		"triggers":      triggers,
		"anti_patterns": d.AntiPatterns,
		"cheatsheet":    d.Cheatsheet,
	}
}
