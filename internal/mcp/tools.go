package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
)

type toolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolDefinitions returns the list of available MCP tools
// Tools are organized by priority:
// 🔴 ESSENTIAL - Core functionality, always use
// 🟡 COMMON - Frequently used
// 🟢 ADVANCED - Specialized scenarios
func ToolDefinitions() []toolDef {
	return []toolDef{
		// ============================================================================
		// 🔴 ESSENTIAL - Agent Onboarding (CALL THESE FIRST!)
		// ============================================================================
		{
			Name:        "get_ramorie_info",
			Description: "🔴 ESSENTIAL | 🧠 CALL THIS FIRST! Get comprehensive information about Ramorie - what it is, how to use it, and agent guidelines.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "setup_agent",
			Description: "🔴 ESSENTIAL | Initialize agent session. Returns current context, active project, pending tasks, and recommended actions.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},

		// ============================================================================
		// 🔴 ESSENTIAL - Project Management
		// ============================================================================
		{
			Name:        "list_projects",
			Description: "🔴 ESSENTIAL | List all projects. Check this to see available projects and which one is active.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "set_active_project",
			Description: "🔴 ESSENTIAL | Set the active project. All new tasks and memories will be created in this project.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectName": map[string]interface{}{"type": "string", "description": "Project name or ID"}}, "required": []string{"projectName"}},
		},

		// ============================================================================
		// 🔴 ESSENTIAL - Task Management (Core)
		// ============================================================================
		{
			Name:        "list_tasks",
			Description: "🔴 ESSENTIAL | List tasks with filtering. 💡 Call before create_task to check for duplicates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "description": "Filter: TODO, IN_PROGRESS, COMPLETED"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID"}, "limit": map[string]interface{}{"type": "number", "description": "Max results"}}},
		},
		{
			Name:        "create_task",
			Description: "🔴 ESSENTIAL | Create a new task. ⚠️ Always check list_tasks first to avoid duplicates!",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"description": map[string]interface{}{"type": "string", "description": "Task description - clear and actionable"}, "priority": map[string]interface{}{"type": "string", "description": "Priority: H=High, M=Medium, L=Low"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID (uses active if not specified)"}}, "required": []string{"description"}},
		},
		{
			Name:        "get_task",
			Description: "🔴 ESSENTIAL | Get task details including notes and metadata.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "start_task",
			Description: "🔴 ESSENTIAL | Start working on a task. Sets status to IN_PROGRESS and enables memory auto-linking.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "complete_task",
			Description: "🔴 ESSENTIAL | Mark task as completed. Use when work is finished.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "get_next_tasks",
			Description: "🔴 ESSENTIAL | Get prioritized TODO tasks. 💡 Use at session start to see what needs attention.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"count": map[string]interface{}{"type": "number", "description": "Number of tasks (default: 5)"}, "project": map[string]interface{}{"type": "string"}}},
		},

		// ============================================================================
		// 🔴 ESSENTIAL - Memory Management (Core)
		// ============================================================================
		{
			Name:        "add_memory",
			Description: "🔴 ESSENTIAL | Store important information to knowledge base. Auto-links to active task. 💡 If it matters later, add it here!",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"content": map[string]interface{}{"type": "string", "description": "Memory content - be descriptive"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID"}}, "required": []string{"content"}},
		},
		{
			Name:        "list_memories",
			Description: "🔴 ESSENTIAL | List memories with optional filtering by project or term.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}, "term": map[string]interface{}{"type": "string", "description": "Filter by keyword"}, "limit": map[string]interface{}{"type": "number"}}},
		},

		// ============================================================================
		// 🟡 COMMON - Task Management (Extended)
		// ============================================================================
		{
			Name:        "add_task_note",
			Description: "🟡 COMMON | Add a note/annotation to a task. Use for progress updates or context.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "note": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "note"}},
		},
		{
			Name:        "update_progress",
			Description: "🟡 COMMON | Update task progress percentage (0-100).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "progress": map[string]interface{}{"type": "number"}}, "required": []string{"taskId", "progress"}},
		},
		{
			Name:        "search_tasks",
			Description: "🟡 COMMON | Search tasks by keyword. Use to find specific tasks.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string", "description": "Search query"}, "status": map[string]interface{}{"type": "string"}, "project": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}, "required": []string{"query"}},
		},
		{
			Name:        "get_active_task",
			Description: "🟡 COMMON | Get the currently active task. Memories auto-link to this task.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},

		// ============================================================================
		// 🟡 COMMON - Memory Management (Extended)
		// ============================================================================
		{
			Name:        "get_memory",
			Description: "🟡 COMMON | Get memory details by ID.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "recall",
			Description: "🟡 COMMON | Advanced memory search with multi-word support, filters, and relations. Supports: OR search (space-separated), AND search (comma-separated), project/tag filtering.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"term": map[string]interface{}{
						"type":        "string",
						"description": "Search terms. Space = OR (any match), comma = AND (all must match). Example: 'traefik docker' finds either, 'traefik,docker' finds both.",
					},
					"project": map[string]interface{}{
						"type":        "string",
						"description": "Filter by project name or ID",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Filter by tag name",
					},
					"linked_task": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, only return memories linked to a task",
					},
					"include_relations": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, include full project and task details (default: true)",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Max results (default: 20)",
					},
					"min_score": map[string]interface{}{
						"type":        "number",
						"description": "Minimum relevance score 0-100 (default: 0)",
					},
				},
				"required": []string{"term"},
			},
		},

		// ============================================================================
		// 🟡 COMMON - Decisions (ADRs)
		// ============================================================================
		{
			Name:        "create_decision",
			Description: "🟡 COMMON | Record an architectural decision (ADR). Use for important technical choices.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string", "description": "Decision title"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string", "description": "draft, proposed, approved, deprecated"}, "area": map[string]interface{}{"type": "string", "description": "Frontend, Backend, Architecture, etc."}, "context": map[string]interface{}{"type": "string", "description": "Why this decision?"}, "consequences": map[string]interface{}{"type": "string", "description": "What are the impacts?"}}, "required": []string{"title"}},
		},
		{
			Name:        "list_decisions",
			Description: "🟡 COMMON | List architectural decisions. Review past decisions before making new ones.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "description": "draft, proposed, approved, deprecated"}, "area": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}},
		},

		// ============================================================================
		// 🟡 COMMON - Reports
		// ============================================================================
		{
			Name:        "get_stats",
			Description: "🟡 COMMON | Get task statistics and completion rates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}}},
		},

		// ============================================================================
		// 🟢 ADVANCED - Less frequently used
		// ============================================================================
		{
			Name:        "create_project",
			Description: "🟢 ADVANCED | Create a new project. ⚠️ Check list_projects first - don't create duplicates!",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "Project name - must be unique"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		{
			Name:        "get_cursor_rules",
			Description: "🟢 ADVANCED | Get Cursor IDE rules for Ramorie. Returns markdown for .cursorrules file.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"format": map[string]interface{}{"type": "string", "description": "markdown (default) or json"}}},
		},
		{
			Name:        "export_project",
			Description: "🟢 ADVANCED | Export project report in markdown format.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}, "format": map[string]interface{}{"type": "string"}}, "required": []string{"project"}},
		},
		{
			Name:        "stop_task",
			Description: "🟢 ADVANCED | Pause a task. Clears active task, keeps IN_PROGRESS status.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
	}
}

// CallTool executes a tool by name with given arguments
func CallTool(client *api.Client, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	// ============================================================================
	// AGENT ONBOARDING
	// ============================================================================
	case "get_ramorie_info":
		return getRamorieInfo(), nil

	case "get_cursor_rules":
		format, _ := args["format"].(string)
		if format == "" {
			format = "markdown"
		}
		return getCursorRules(format), nil

	case "setup_agent":
		return setupAgent(client)

	// ============================================================================
	// PROJECT MANAGEMENT
	// ============================================================================
	case "list_projects":
		return client.ListProjects()

	case "set_active_project":
		projectName, _ := args["projectName"].(string)
		projectName = strings.TrimSpace(projectName)
		if projectName == "" {
			return nil, errors.New("projectName is required")
		}
		projects, err := client.ListProjects()
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.Name == projectName || strings.HasPrefix(p.ID.String(), projectName) {
				if err := client.SetProjectActive(p.ID.String()); err != nil {
					return nil, err
				}
				cfg, _ := config.LoadConfig()
				if cfg == nil {
					cfg = &config.Config{}
				}
				cfg.ActiveProjectID = p.ID.String()
				_ = config.SaveConfig(cfg)
				return map[string]interface{}{"ok": true, "project_id": p.ID.String(), "name": p.Name}, nil
			}
		}
		return nil, errors.New("project not found")

	case "create_project":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateProject(name, strings.TrimSpace(description))

	// ============================================================================
	// TASK MANAGEMENT
	// ============================================================================
	case "list_tasks":
		status, _ := args["status"].(string)
		projectIdentifier, _ := args["project"].(string)
		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}
		tasks, err := client.ListTasks(projectID, strings.TrimSpace(status))
		if err != nil {
			return nil, err
		}
		limit := toInt(args["limit"])
		if limit > 0 && limit < len(tasks) {
			tasks = tasks[:limit]
		}
		return tasks, nil

	case "create_task":
		description, _ := args["description"].(string)
		description = strings.TrimSpace(description)
		if description == "" {
			return nil, errors.New("description is required")
		}
		priority, _ := args["priority"].(string)
		priority = normalizePriority(priority)
		projectIdentifier, _ := args["project"].(string)
		projectID, err := resolveProjectID(client, projectIdentifier)
		if err != nil {
			return nil, err
		}
		task, err := client.CreateTask(projectID, description, "", priority)
		if err != nil {
			return nil, err
		}
		return task, nil

	case "get_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.GetTask(taskID)

	case "start_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.StartTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "message": "Task started. Memories will now auto-link to this task."}, nil

	case "complete_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.CompleteTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true}, nil

	case "stop_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.StopTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true}, nil

	case "get_next_tasks":
		count := toInt(args["count"])
		if count <= 0 {
			count = 5
		}
		projectIdentifier, _ := args["project"].(string)

		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}

		tasks, err := client.ListTasksQuery(projectID, "TODO", "", nil, nil)
		if err != nil {
			return nil, err
		}

		sort.Slice(tasks, func(i, j int) bool {
			pi := priorityRank(tasks[i].Priority)
			pj := priorityRank(tasks[j].Priority)
			if pi != pj {
				return pi > pj
			}
			return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		})

		if count < len(tasks) {
			tasks = tasks[:count]
		}
		return tasks, nil

	case "add_task_note":
		taskID, _ := args["taskId"].(string)
		note, _ := args["note"].(string)
		taskID = strings.TrimSpace(taskID)
		note = strings.TrimSpace(note)
		if taskID == "" || note == "" {
			return nil, errors.New("taskId and note are required")
		}
		return client.CreateAnnotation(taskID, note)

	case "update_progress":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		progress := toInt(args["progress"])
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if progress < 0 || progress > 100 {
			return nil, errors.New("progress must be between 0 and 100")
		}
		return client.UpdateTask(taskID, map[string]interface{}{"progress": progress})

	case "search_tasks":
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return nil, errors.New("query is required")
		}
		status, _ := args["status"].(string)
		projectIdentifier, _ := args["project"].(string)
		limit := toInt(args["limit"])

		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}

		tasks, err := client.ListTasksQuery(projectID, strings.TrimSpace(status), query, nil, nil)
		if err != nil {
			return nil, err
		}
		if limit > 0 && limit < len(tasks) {
			tasks = tasks[:limit]
		}
		return tasks, nil

	case "get_active_task":
		return client.GetActiveTask()

	// ============================================================================
	// MEMORY MANAGEMENT
	// ============================================================================
	case "add_memory":
		content, _ := args["content"].(string)
		content = strings.TrimSpace(content)
		if content == "" {
			return nil, errors.New("content is required")
		}
		projectIdentifier, _ := args["project"].(string)
		projectID, err := resolveProjectID(client, projectIdentifier)
		if err != nil {
			return nil, err
		}
		return client.CreateMemory(projectID, content)

	case "list_memories":
		projectIdentifier, _ := args["project"].(string)
		term, _ := args["term"].(string)
		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}
		memories, err := client.ListMemories(projectID, "")
		if err != nil {
			return nil, err
		}
		term = strings.TrimSpace(term)
		if term != "" {
			filtered := memories[:0]
			for _, m := range memories {
				if strings.Contains(strings.ToLower(m.Content), strings.ToLower(term)) {
					filtered = append(filtered, m)
				}
			}
			memories = filtered
		}
		limit := toInt(args["limit"])
		if limit > 0 && limit < len(memories) {
			memories = memories[:limit]
		}
		return memories, nil

	case "get_memory":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		return client.GetMemory(memoryID)

	case "recall":
		term, _ := args["term"].(string)
		term = strings.TrimSpace(term)
		if term == "" {
			return nil, errors.New("term is required")
		}

		// Parse options
		limit := toInt(args["limit"])
		if limit == 0 {
			limit = 20
		}
		minScore := toInt(args["min_score"])
		projectFilter, _ := args["project"].(string)
		tagFilter, _ := args["tag"].(string)
		linkedTaskOnly, _ := args["linked_task"].(bool)
		includeRelations := true
		if ir, ok := args["include_relations"].(bool); ok {
			includeRelations = ir
		}

		// Resolve project filter if provided
		projectID := ""
		if strings.TrimSpace(projectFilter) != "" {
			pid, err := resolveProjectID(client, projectFilter)
			if err == nil {
				projectID = pid
			}
		}

		// Fetch memories
		memories, err := client.ListMemories(projectID, "")
		if err != nil {
			return nil, err
		}

		// Parse search terms
		// Comma-separated = AND (all must match)
		// Space-separated = OR (any match)
		isAndSearch := strings.Contains(term, ",")
		var searchTerms []string
		if isAndSearch {
			for _, t := range strings.Split(term, ",") {
				t = strings.TrimSpace(strings.ToLower(t))
				if t != "" {
					searchTerms = append(searchTerms, t)
				}
			}
		} else {
			for _, t := range strings.Fields(term) {
				t = strings.TrimSpace(strings.ToLower(t))
				if t != "" {
					searchTerms = append(searchTerms, t)
				}
			}
		}

		// Score and filter memories
		type scoredMemory struct {
			memory interface{}
			score  int
		}
		var scored []scoredMemory

		for _, m := range memories {
			// Filter: linked_task
			if linkedTaskOnly && m.LinkedTaskID == nil {
				continue
			}

			// Filter: tag
			if tagFilter != "" {
				hasTag := false
				if tags, ok := m.Tags.([]interface{}); ok {
					for _, tag := range tags {
						if tagStr, ok := tag.(string); ok {
							if strings.EqualFold(tagStr, tagFilter) {
								hasTag = true
								break
							}
						}
					}
				}
				if !hasTag {
					continue
				}
			}

			// Calculate relevance score
			contentLower := strings.ToLower(m.Content)
			score := 0
			matchCount := 0

			for _, term := range searchTerms {
				if strings.Contains(contentLower, term) {
					matchCount++
					// Base score for match
					score += 20
					// Bonus for exact word match
					if strings.Contains(contentLower, " "+term+" ") ||
						strings.HasPrefix(contentLower, term+" ") ||
						strings.HasSuffix(contentLower, " "+term) {
						score += 10
					}
					// Bonus for title/header match (## or ###)
					if strings.Contains(contentLower, "## "+term) ||
						strings.Contains(contentLower, "### "+term) {
						score += 15
					}
					// Bonus for multiple occurrences
					occurrences := strings.Count(contentLower, term)
					if occurrences > 1 {
						score += min(occurrences*5, 25)
					}
				}
			}

			// AND search: all terms must match
			if isAndSearch && matchCount < len(searchTerms) {
				continue
			}
			// OR search: at least one term must match
			if !isAndSearch && matchCount == 0 {
				continue
			}

			// Bonus for linked task (organized knowledge)
			if m.LinkedTaskID != nil {
				score += 5
			}

			// Skip if below minimum score
			if score < minScore {
				continue
			}

			// Build result object
			result := map[string]interface{}{
				"id":         m.ID.String(),
				"content":    m.Content,
				"score":      score,
				"created_at": m.CreatedAt,
			}

			// Include relations
			if includeRelations {
				if m.Project != nil {
					result["project"] = map[string]interface{}{
						"id":   m.Project.ID.String(),
						"name": m.Project.Name,
					}
				}
				if m.LinkedTaskID != nil {
					result["linked_task_id"] = m.LinkedTaskID.String()
				}
				if m.Tags != nil {
					result["tags"] = m.Tags
				}
			}

			scored = append(scored, scoredMemory{memory: result, score: score})
		}

		// Sort by score (descending)
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		// Apply limit and extract results
		var results []interface{}
		for i, s := range scored {
			if i >= limit {
				break
			}
			results = append(results, s.memory)
		}

		return map[string]interface{}{
			"term":        term,
			"search_mode": map[bool]string{true: "AND", false: "OR"}[isAndSearch],
			"count":       len(results),
			"total_found": len(scored),
			"results":     results,
		}, nil

	// ============================================================================
	// DECISIONS (ADRs)
	// ============================================================================
	case "create_decision":
		title, _ := args["title"].(string)
		title = strings.TrimSpace(title)
		if title == "" {
			return nil, errors.New("title is required")
		}
		description, _ := args["description"].(string)
		status, _ := args["status"].(string)
		area, _ := args["area"].(string)
		context, _ := args["context"].(string)
		consequences, _ := args["consequences"].(string)

		return client.CreateDecision(
			title,
			strings.TrimSpace(description),
			strings.TrimSpace(status),
			strings.TrimSpace(area),
			strings.TrimSpace(context),
			strings.TrimSpace(consequences),
		)

	case "list_decisions":
		status, _ := args["status"].(string)
		area, _ := args["area"].(string)
		limit := toInt(args["limit"])
		decisions, err := client.ListDecisions(strings.TrimSpace(status), strings.TrimSpace(area), limit)
		if err != nil {
			return nil, err
		}
		return decisions, nil

	// ============================================================================
	// REPORTS
	// ============================================================================
	case "get_stats":
		b, err := client.Request("GET", "/reports/stats", nil)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("invalid stats response")
		}
		return out, nil

	case "export_project":
		projectIdentifier, _ := args["project"].(string)
		format, _ := args["format"].(string)
		if format == "" {
			format = "markdown"
		}

		projectID, err := resolveProjectID(client, projectIdentifier)
		if err != nil {
			return nil, err
		}

		projects, err := client.ListProjects()
		if err != nil {
			return nil, err
		}

		var project *struct {
			Name        string
			Description string
		}
		for _, p := range projects {
			if p.ID.String() == projectID {
				project = &struct {
					Name        string
					Description string
				}{p.Name, p.Description}
				break
			}
		}

		if project == nil {
			return nil, errors.New("project not found")
		}

		tasks, err := client.ListTasks(projectID, "")
		if err != nil {
			return nil, err
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", project.Name))
		if project.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", project.Description))
		}

		total := len(tasks)
		completed := 0
		inProgress := 0
		pending := 0
		for _, t := range tasks {
			switch t.Status {
			case "COMPLETED":
				completed++
			case "IN_PROGRESS":
				inProgress++
			default:
				pending++
			}
		}

		sb.WriteString("## Statistics\n\n")
		sb.WriteString(fmt.Sprintf("- **Total:** %d\n", total))
		sb.WriteString(fmt.Sprintf("- **Completed:** %d\n", completed))
		sb.WriteString(fmt.Sprintf("- **In Progress:** %d\n", inProgress))
		sb.WriteString(fmt.Sprintf("- **Pending:** %d\n\n", pending))

		sb.WriteString("## Tasks\n\n")
		for _, t := range tasks {
			status := "⏳"
			if t.Status == "COMPLETED" {
				status = "✅"
			} else if t.Status == "IN_PROGRESS" {
				status = "🔄"
			}
			sb.WriteString(fmt.Sprintf("- %s **%s** [%s]\n", status, t.Title, t.Priority))
		}

		return map[string]interface{}{
			"project":  project.Name,
			"format":   format,
			"markdown": sb.String(),
		}, nil

	default:
		return nil, errors.New("tool not implemented")
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func priorityRank(p string) int {
	switch strings.ToUpper(strings.TrimSpace(p)) {
	case "H", "HIGH":
		return 3
	case "M", "MEDIUM":
		return 2
	case "L", "LOW":
		return 1
	default:
		return 0
	}
}

func resolveProjectID(client *api.Client, projectIdentifier string) (string, error) {
	projectIdentifier = strings.TrimSpace(projectIdentifier)
	if projectIdentifier == "" {
		cfg, err := config.LoadConfig()
		if err == nil && cfg.ActiveProjectID != "" {
			return cfg.ActiveProjectID, nil
		}
		projects, err := client.ListProjects()
		if err != nil {
			return "", err
		}
		for _, p := range projects {
			if p.IsActive {
				return p.ID.String(), nil
			}
		}
		return "", errors.New("no active project - use set_active_project first")
	}

	projects, err := client.ListProjects()
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.Name == projectIdentifier || strings.HasPrefix(p.ID.String(), projectIdentifier) {
			return p.ID.String(), nil
		}
	}

	return "", errors.New("project not found")
}

func normalizePriority(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return "M"
	}
	switch s {
	case "H", "HIGH":
		return "H"
	case "M", "MEDIUM":
		return "M"
	case "L", "LOW":
		return "L"
	default:
		return "M"
	}
}

func toInt(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		var x int
		_, _ = fmt.Sscanf(t, "%d", &x)
		return x
	default:
		return 0
	}
}

// ============================================================================
// AGENT ONBOARDING & SELF-DOCUMENTATION
// ============================================================================

func getRamorieInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":    "Ramorie",
		"version": "1.10.0",
		"tagline": "AI Agent Memory & Task Management System",
		"description": `Ramorie is a persistent memory and task management system for AI agents.
It enables context preservation across sessions, task tracking, and knowledge storage.`,

		"tool_count": 25,
		"tool_priority_guide": map[string]string{
			"🔴 ESSENTIAL": "Core functionality - use these regularly",
			"🟡 COMMON":    "Frequently used - call when needed",
			"🟢 ADVANCED":  "Specialized - only for specific scenarios",
		},

		"quickstart": []string{
			"1. setup_agent → Get current context and recommendations",
			"2. list_projects → See available projects",
			"3. set_active_project → Set your working project",
			"4. get_next_tasks → See prioritized TODO tasks",
			"5. start_task → Begin working (enables memory auto-link)",
			"6. add_memory → Store important discoveries",
			"7. complete_task → Mark work as done",
		},

		"core_rules": []string{
			"✅ Always check list_tasks before creating new tasks",
			"✅ Use add_memory to persist important information",
			"✅ Start a task before adding memories for auto-linking",
			"✅ Record architectural decisions with create_decision",
			"❌ Never delete without explicit user approval",
			"❌ Never create duplicate projects",
		},

		"tools_by_category": map[string][]string{
			"🔴 agent":    {"get_ramorie_info", "setup_agent"},
			"🔴 project":  {"list_projects", "set_active_project"},
			"🔴 task":     {"list_tasks", "create_task", "get_task", "start_task", "complete_task", "get_next_tasks"},
			"🔴 memory":   {"add_memory", "list_memories"},
			"🟡 task":     {"add_task_note", "update_progress", "search_tasks", "get_active_task"},
			"🟡 memory":   {"get_memory", "recall"},
			"🟡 decision": {"create_decision", "list_decisions"},
			"🟡 reports":  {"get_stats"},
			"🟢 project":  {"create_project"},
			"🟢 agent":    {"get_cursor_rules"},
			"🟢 reports":  {"export_project"},
			"🟢 task":     {"stop_task"},
		},
	}
}

func getCursorRules(format string) map[string]interface{} {
	rules := `# 🧠 Ramorie MCP Usage Rules

## Core Principle
**"If it matters later, it belongs in Ramorie."**

## Tool Priority
- 🔴 ESSENTIAL: Core functionality, use regularly
- 🟡 COMMON: Frequently used, call when needed
- 🟢 ADVANCED: Specialized scenarios only

## Session Workflow

### Start of Session
1. ` + "`setup_agent`" + ` - Get current context
2. ` + "`list_projects`" + ` - Check available projects
3. ` + "`get_next_tasks`" + ` - See what needs attention

### During Work
1. ` + "`start_task`" + ` - Begin working (enables memory auto-link)
2. ` + "`add_memory`" + ` - Store important discoveries
3. ` + "`add_task_note`" + ` - Add progress notes
4. ` + "`complete_task`" + ` - Mark as done

### Key Rules
- ✅ Check ` + "`list_tasks`" + ` before creating new tasks
- ✅ Use ` + "`add_memory`" + ` for important information
- ✅ Record decisions with ` + "`create_decision`" + `
- ❌ Never delete without user approval
- ❌ Never create duplicate projects

## Available Tools (25 total)

### 🔴 ESSENTIAL (12)
- get_ramorie_info, setup_agent
- list_projects, set_active_project
- list_tasks, create_task, get_task, start_task, complete_task, get_next_tasks
- add_memory, list_memories

### 🟡 COMMON (9)
- add_task_note, update_progress, search_tasks, get_active_task
- get_memory, recall
- create_decision, list_decisions
- get_stats

### 🟢 ADVANCED (4)
- create_project, get_cursor_rules, export_project, stop_task
`

	result := map[string]interface{}{
		"format": format,
		"rules":  rules,
		"usage":  "Add this to your .cursorrules file",
	}

	return result
}

func setupAgent(client *api.Client) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"status":  "ready",
		"message": "🧠 Ramorie agent session initialized",
		"version": "1.10.0",
	}

	// Get active project
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.ActiveProjectID != "" {
		result["active_project_id"] = cfg.ActiveProjectID
	}

	// List projects
	projects, err := client.ListProjects()
	if err == nil {
		for _, p := range projects {
			if p.IsActive {
				result["active_project"] = map[string]interface{}{
					"id":   p.ID.String(),
					"name": p.Name,
				}
				break
			}
		}
		result["projects_count"] = len(projects)
	}

	// Get active task
	activeTask, err := client.GetActiveTask()
	if err == nil && activeTask != nil {
		result["active_task"] = map[string]interface{}{
			"id":     activeTask.ID.String(),
			"title":  activeTask.Title,
			"status": activeTask.Status,
		}
	}

	// Get TODO tasks count
	if cfg != nil && cfg.ActiveProjectID != "" {
		tasks, err := client.ListTasks(cfg.ActiveProjectID, "TODO")
		if err == nil {
			result["pending_tasks_count"] = len(tasks)
		}
	}

	// Get stats
	statsBytes, err := client.Request("GET", "/reports/stats", nil)
	if err == nil {
		var stats map[string]interface{}
		if json.Unmarshal(statsBytes, &stats) == nil {
			result["stats"] = stats
		}
	}

	// Recommendations
	recommendations := []string{}
	if result["active_project"] == nil {
		recommendations = append(recommendations, "⚠️ Set an active project: set_active_project")
	}
	if result["active_task"] == nil {
		recommendations = append(recommendations, "💡 Start a task for memory auto-linking: start_task")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✅ Ready to work! Use get_next_tasks to see priorities")
	}
	result["next_steps"] = recommendations

	return result, nil
}
