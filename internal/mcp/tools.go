package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
)

type toolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func ToolDefinitions() []toolDef {
	return []toolDef{
		{
			Name:        "create_task",
			Description: "Create a new task. ⚠️ IMPORTANT: Use existing project (project parameter), do NOT create new projects for tasks. Check list_tasks first to avoid duplicates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"description": map[string]interface{}{"type": "string", "description": "Task description - clear and actionable"}, "priority": map[string]interface{}{"type": "string", "description": "Priority: H=High, M=Medium, L=Low"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID (uses active project if not specified)"}}, "required": []string{"description"}},
		},
		{
			Name:        "list_tasks",
			Description: "List tasks with filtering. 💡 Call this before create_task to check for existing similar tasks and avoid duplicates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "description": "Filter by status: TODO, IN_PROGRESS, COMPLETED"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID"}, "limit": map[string]interface{}{"type": "number", "description": "Maximum results"}}},
		},
		{
			Name:        "search_tasks",
			Description: "Search tasks by keyword. Use before creating similar tasks to avoid duplicates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string", "description": "Search query"}, "status": map[string]interface{}{"type": "string"}, "project": map[string]interface{}{"type": "string"}, "tag": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}, "required": []string{"query"}},
		},
		{
			Name:        "get_next_tasks",
			Description: "Get prioritized tasks for agent workflow. 💡 Use at session start to see what needs attention. Tasks sorted by priority (H>M>L).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"count": map[string]interface{}{"type": "number", "description": "Number of tasks (default: 5)"}, "project": map[string]interface{}{"type": "string"}, "tag": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "get_task",
			Description: "Get task details including notes and metadata.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "start_task",
			Description: "Start a task (sets status to IN_PROGRESS). Use when beginning work on a task.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "complete_task",
			Description: "Complete a task (sets status to COMPLETED). Use when work is finished.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "stop_task",
			Description: "Pause a task (clears active task, keeps IN_PROGRESS status).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "get_active_task",
			Description: "Get the currently active task (for memory auto-linking).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "delete_task",
			Description: "Delete a task (soft delete).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "update_task_status",
			Description: "Update task status (TODO, IN_PROGRESS, COMPLETED).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "status"}},
		},
		{
			Name:        "update_progress",
			Description: "Update task progress percentage (0-100).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "progress": map[string]interface{}{"type": "number"}}, "required": []string{"taskId", "progress"}},
		},
		{
			Name:        "add_task_note",
			Description: "Add a note/annotation to a task.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "note": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "note"}},
		},
		{
			Name:        "create_subtask",
			Description: "Create a subtask for breaking down work.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"parentTaskId": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"parentTaskId", "description"}},
		},
		{
			Name:        "bulk_start_tasks",
			Description: "Start multiple tasks at once.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskIds": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"taskIds"}},
		},
		{
			Name:        "bulk_complete_tasks",
			Description: "Complete multiple tasks at once.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskIds": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"taskIds"}},
		},
		{
			Name:        "list_projects",
			Description: "List all projects. ⚠️ ALWAYS call this before create_project to check existing projects!",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "create_project",
			Description: "Create a new project. ⚠️ IMPORTANT: Call list_projects FIRST! Do NOT create duplicates - use existing project if one with similar name/purpose exists.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "Project name - must be unique"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		{
			Name:        "set_active_project",
			Description: "Set the active project for new tasks.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectName": map[string]interface{}{"type": "string"}}, "required": []string{"projectName"}},
		},
		{
			Name:        "get_project",
			Description: "Get project details.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectId": map[string]interface{}{"type": "string"}}, "required": []string{"projectId"}},
		},
		{
			Name:        "update_project",
			Description: "Update project name or description.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectId": map[string]interface{}{"type": "string"}, "name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"projectId"}},
		},
		{
			Name:        "delete_project",
			Description: "Delete a project.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectId": map[string]interface{}{"type": "string"}}, "required": []string{"projectId"}},
		},
		{
			Name:        "add_memory",
			Description: "Add a new memory/note to the knowledge base. Use for storing important information, learnings, decisions, or context.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"content": map[string]interface{}{"type": "string", "description": "Memory content - be descriptive"}, "project": map[string]interface{}{"type": "string"}}, "required": []string{"content"}},
		},
		{
			Name:        "list_memories",
			Description: "List memories with optional filtering.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}, "term": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "get_task_memories",
			Description: "Get memories linked to a specific task.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "memory_tasks",
			Description: "Get tasks linked to a specific memory.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "create_memory_task_link",
			Description: "Create a manual link between a task and memory.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "memoryId": map[string]interface{}{"type": "string"}, "relationType": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "memoryId"}},
		},
		{
			Name:        "get_memory",
			Description: "Get memory details by ID.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "update_memory",
			Description: "Update memory content or tags.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}, "tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "delete_memory",
			Description: "Delete a memory.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "get_stats",
			Description: "Get task statistics and completion rates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "get_history",
			Description: "Get task activity history for the last N days.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"days": map[string]interface{}{"type": "number", "description": "Number of days (default: 7)"}, "project": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "analyze_task_risks",
			Description: "Analyze potential risks for a task using AI.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "analyze_task_dependencies",
			Description: "Analyze dependencies and blockers for a task using AI.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "duplicate_task",
			Description: "Duplicate a task with tags and notes (status reset to TODO, progress to 0).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "newDescription": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "move_tasks_to_project",
			Description: "Move tasks to another existing project.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskIds": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "targetProject": map[string]interface{}{"type": "string", "description": "Target project name or ID"}}, "required": []string{"taskIds", "targetProject"}},
		},
		{
			Name:        "timeline",
			Description: "Get activity timeline for the last N days.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"days": map[string]interface{}{"type": "number"}, "project": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "recall",
			Description: "Search memories by text (keyword search).",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"term": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}, "required": []string{"term"}},
		},
		{
			Name:        "export_project",
			Description: "Export a project report in markdown format.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}, "format": map[string]interface{}{"type": "string"}}, "required": []string{"project"}},
		},
		{
			Name:        "list_contexts",
			Description: "List all contexts.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "create_context",
			Description: "Create a new context.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		{
			Name:        "set_active_context",
			Description: "Set the active context.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		// Context Pack tools
		{
			Name:        "list_context_packs",
			Description: "List context packs (Active Context). Filter by type, status, or search query.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"type": map[string]interface{}{"type": "string", "description": "Pack type: project, integration, decision, custom"}, "status": map[string]interface{}{"type": "string", "description": "Pack status: draft, published"}, "query": map[string]interface{}{"type": "string", "description": "Search in name/description"}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "get_context_pack",
			Description: "Get context pack details.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}}, "required": []string{"packId"}},
		},
		{
			Name:        "create_context_pack",
			Description: "Create a new context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string", "description": "Pack type: project, integration, decision, custom"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string", "description": "Pack status: draft, published"}, "tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"name", "type"}},
		},
		{
			Name:        "update_context_pack",
			Description: "Update an existing context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}, "name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}, "tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"packId"}},
		},
		{
			Name:        "delete_context_pack",
			Description: "Delete a context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}}, "required": []string{"packId"}},
		},
		{
			Name:        "activate_context_pack",
			Description: "Activate (publish) a context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}}, "required": []string{"packId"}},
		},
		{
			Name:        "get_active_context_pack",
			Description: "Get the currently active context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		// Organization tools
		{
			Name:        "list_organizations",
			Description: "List organizations.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "get_organization",
			Description: "Get organization details.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"orgId": map[string]interface{}{"type": "string"}}, "required": []string{"orgId"}},
		},
		{
			Name:        "create_organization",
			Description: "Create a new organization.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		// Decision tools - for agents to record architectural decisions
		{
			Name:        "list_decisions",
			Description: "List architectural decisions (ADRs). Agents record important decisions here.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "description": "Status filter: draft, proposed, approved, deprecated"}, "area": map[string]interface{}{"type": "string", "description": "Area filter: Frontend, Backend, Architecture, etc."}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "get_decision",
			Description: "Get decision details.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"decisionId": map[string]interface{}{"type": "string", "description": "Decision ID or ADR number (e.g., ADR-001)"}}, "required": []string{"decisionId"}},
		},
		{
			Name:        "create_decision",
			Description: "Create a new architectural decision (ADR). Use to record important technical decisions.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string", "description": "Decision title"}, "description": map[string]interface{}{"type": "string", "description": "Short description"}, "status": map[string]interface{}{"type": "string", "description": "Status: draft, proposed, approved, deprecated"}, "area": map[string]interface{}{"type": "string", "description": "Area: Frontend, Backend, Architecture, DevOps, etc."}, "context": map[string]interface{}{"type": "string", "description": "Context and reasoning for the decision"}, "consequences": map[string]interface{}{"type": "string", "description": "Consequences and impacts of the decision"}}, "required": []string{"title"}},
		},
		{
			Name:        "update_decision",
			Description: "Update an existing decision.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"decisionId": map[string]interface{}{"type": "string"}, "title": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}, "area": map[string]interface{}{"type": "string"}, "context": map[string]interface{}{"type": "string"}, "consequences": map[string]interface{}{"type": "string"}}, "required": []string{"decisionId"}},
		},
		{
			Name:        "delete_decision",
			Description: "Delete a decision.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"decisionId": map[string]interface{}{"type": "string"}}, "required": []string{"decisionId"}},
		},
		// Agent Onboarding & Self-Documentation tools
		{
			Name:        "get_ramorie_info",
			Description: "🧠 CALL THIS FIRST! Get comprehensive information about Ramorie - what it is, how to use it, and agent guidelines. Essential for understanding the system.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "get_cursor_rules",
			Description: "Get Cursor IDE rules for Ramorie integration. Returns markdown rules that can be added to .cursorrules file.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"format": map[string]interface{}{"type": "string", "description": "Output format: markdown (default) or json"}}},
		},
		{
			Name:        "setup_agent",
			Description: "Initialize agent session with Ramorie. Returns current context, active project, and recommended first actions.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	}
}

func CallTool(client *api.Client, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
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

	case "search_tasks":
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return nil, errors.New("query is required")
		}
		status, _ := args["status"].(string)
		projectIdentifier, _ := args["project"].(string)
		tag, _ := args["tag"].(string)
		limit := toInt(args["limit"])

		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}

		tags := []string{}
		if strings.TrimSpace(tag) != "" {
			tags = append(tags, strings.TrimSpace(tag))
		}

		tasks, err := client.ListTasksQuery(projectID, strings.TrimSpace(status), query, nil, tags)
		if err != nil {
			return nil, err
		}
		if limit > 0 && limit < len(tasks) {
			tasks = tasks[:limit]
		}
		return tasks, nil

	case "get_next_tasks":
		count := toInt(args["count"])
		if count <= 0 {
			count = 5
		}
		projectIdentifier, _ := args["project"].(string)
		tag, _ := args["tag"].(string)

		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}

		tags := []string{}
		if strings.TrimSpace(tag) != "" {
			tags = append(tags, strings.TrimSpace(tag))
		}

		// Default to TODO tasks
		tasks, err := client.ListTasksQuery(projectID, "TODO", "", nil, tags)
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
		return map[string]interface{}{"ok": true}, nil

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

	case "get_active_task":
		return client.GetActiveTask()

	case "delete_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.DeleteTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": taskID}, nil

	case "update_task_status":
		taskID, _ := args["taskId"].(string)
		status, _ := args["status"].(string)
		taskID = strings.TrimSpace(taskID)
		status = strings.TrimSpace(status)
		if taskID == "" || status == "" {
			return nil, errors.New("taskId and status are required")
		}
		return client.UpdateTask(taskID, map[string]interface{}{"status": status})

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

	case "add_task_note":
		taskID, _ := args["taskId"].(string)
		note, _ := args["note"].(string)
		taskID = strings.TrimSpace(taskID)
		note = strings.TrimSpace(note)
		if taskID == "" || note == "" {
			return nil, errors.New("taskId and note are required")
		}
		return client.CreateAnnotation(taskID, note)

	case "create_subtask":
		parentTaskID, _ := args["parentTaskId"].(string)
		description, _ := args["description"].(string)
		parentTaskID = strings.TrimSpace(parentTaskID)
		description = strings.TrimSpace(description)
		if parentTaskID == "" || description == "" {
			return nil, errors.New("parentTaskId and description are required")
		}
		return client.CreateSubtask(parentTaskID, description)

	case "bulk_start_tasks":
		ids, err := resolveTaskIDList(client, args["taskIds"])
		if err != nil {
			return nil, err
		}
		status := "IN_PROGRESS"
		if err := client.BulkUpdateTasks(ids, &status, nil, nil); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "count": len(ids)}, nil

	case "bulk_complete_tasks":
		ids, err := resolveTaskIDList(client, args["taskIds"])
		if err != nil {
			return nil, err
		}
		status := "COMPLETED"
		if err := client.BulkUpdateTasks(ids, &status, nil, nil); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "count": len(ids)}, nil

	case "list_projects":
		return client.ListProjects()

	case "create_project":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateProject(name, strings.TrimSpace(description))

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

	case "get_project":
		projectID, _ := args["projectId"].(string)
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required")
		}
		return client.GetProject(projectID)

	case "update_project":
		projectID, _ := args["projectId"].(string)
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required")
		}
		updates := make(map[string]interface{})
		if name, ok := args["name"].(string); ok && strings.TrimSpace(name) != "" {
			updates["name"] = strings.TrimSpace(name)
		}
		if description, ok := args["description"].(string); ok {
			updates["description"] = strings.TrimSpace(description)
		}
		return client.UpdateProject(projectID, updates)

	case "delete_project":
		projectID, _ := args["projectId"].(string)
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required")
		}
		if err := client.DeleteProject(projectID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": projectID}, nil

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

	case "update_memory":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		updates := make(map[string]interface{})
		if content, ok := args["content"].(string); ok && strings.TrimSpace(content) != "" {
			updates["content"] = strings.TrimSpace(content)
		}
		if tagsRaw, ok := args["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
			updates["tags"] = tags
		}
		return client.UpdateMemory(memoryID, updates)

	case "delete_memory":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		if err := client.DeleteMemory(memoryID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": memoryID}, nil

	case "get_task_memories":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.ListTaskMemories(taskID)

	case "memory_tasks":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		return client.ListMemoryTasks(memoryID)

	case "create_memory_task_link":
		taskID, _ := args["taskId"].(string)
		memoryID, _ := args["memoryId"].(string)
		relationType, _ := args["relationType"].(string)
		taskID = strings.TrimSpace(taskID)
		memoryID = strings.TrimSpace(memoryID)
		relationType = strings.TrimSpace(relationType)
		if taskID == "" || memoryID == "" {
			return nil, errors.New("taskId and memoryId are required")
		}
		b, err := client.CreateMemoryTaskLink(taskID, memoryID, relationType)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return map[string]interface{}{"ok": true}, nil
		}
		return out, nil

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

	case "get_history":
		days := toInt(args["days"])
		if days == 0 {
			days = 7
		}
		endpoint := "/reports/history"
		if days > 0 {
			endpoint = fmt.Sprintf("/reports/history?days=%d", days)
		}
		b, err := client.Request("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("invalid history response")
		}
		return out, nil

	case "analyze_task_risks":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.AIRisks(taskID)

	case "analyze_task_dependencies":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.AIDependencies(taskID)

	case "duplicate_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		newDescription, _ := args["newDescription"].(string)

		// Get original task
		original, err := client.GetTask(taskID)
		if err != nil {
			return nil, err
		}

		// Create new task with same properties
		title := original.Title
		if newDescription != "" {
			title = newDescription
		} else {
			title = title + " (kopya)"
		}

		newTask, err := client.CreateTask(
			original.ProjectID.String(),
			title,
			original.Description,
			original.Priority,
		)
		if err != nil {
			return nil, err
		}

		// Copy annotations
		for _, ann := range original.Annotations {
			_, _ = client.CreateAnnotation(newTask.ID.String(), ann.Content)
		}

		return map[string]interface{}{
			"ok":          true,
			"original_id": original.ID.String(),
			"new_id":      newTask.ID.String(),
			"title":       newTask.Title,
		}, nil

	case "move_tasks_to_project":
		taskIdsRaw := args["taskIds"]
		targetProject, _ := args["targetProject"].(string)
		targetProject = strings.TrimSpace(targetProject)
		if targetProject == "" {
			return nil, errors.New("targetProject is required")
		}

		taskIds, err := resolveTaskIDList(client, taskIdsRaw)
		if err != nil {
			return nil, err
		}

		// Resolve project
		projectID, err := resolveProjectID(client, targetProject)
		if err != nil {
			return nil, err
		}

		// Move each task
		movedCount := 0
		for _, id := range taskIds {
			_, err := client.UpdateTask(id, map[string]interface{}{"project_id": projectID})
			if err == nil {
				movedCount++
			}
		}

		return map[string]interface{}{
			"ok":         true,
			"moved":      movedCount,
			"total":      len(taskIds),
			"project_id": projectID,
		}, nil

	case "timeline":
		days := toInt(args["days"])
		if days == 0 {
			days = 7
		}
		projectIdentifier, _ := args["project"].(string)

		endpoint := fmt.Sprintf("/reports/history?days=%d", days)
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err == nil {
				endpoint = fmt.Sprintf("/reports/history?days=%d&project_id=%s", days, pid)
			}
		}

		b, err := client.Request("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("invalid timeline response")
		}
		return out, nil

	case "recall":
		term, _ := args["term"].(string)
		term = strings.TrimSpace(term)
		if term == "" {
			return nil, errors.New("term is required")
		}
		limit := toInt(args["limit"])
		if limit == 0 {
			limit = 10
		}

		// Get all memories and filter by term
		memories, err := client.ListMemories("", "")
		if err != nil {
			return nil, err
		}

		var filtered []interface{}
		for _, m := range memories {
			if strings.Contains(strings.ToLower(m.Content), strings.ToLower(term)) {
				filtered = append(filtered, map[string]interface{}{
					"id":         m.ID.String(),
					"content":    m.Content,
					"created_at": m.CreatedAt,
				})
				if len(filtered) >= limit {
					break
				}
			}
		}

		return map[string]interface{}{
			"term":    term,
			"count":   len(filtered),
			"results": filtered,
		}, nil

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

		// Get project details
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

		// Get tasks
		tasks, err := client.ListTasks(projectID, "")
		if err != nil {
			return nil, err
		}

		// Build markdown report
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", project.Name))
		if project.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", project.Description))
		}

		// Stats
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

		sb.WriteString("## İstatistikler\n\n")
		sb.WriteString(fmt.Sprintf("- **Toplam:** %d\n", total))
		sb.WriteString(fmt.Sprintf("- **Tamamlanan:** %d\n", completed))
		sb.WriteString(fmt.Sprintf("- **Devam Eden:** %d\n", inProgress))
		sb.WriteString(fmt.Sprintf("- **Bekleyen:** %d\n\n", pending))

		// Task list
		sb.WriteString("## Görevler\n\n")
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

	case "list_contexts":
		contexts, err := client.ListContexts()
		if err != nil {
			return nil, err
		}
		return contexts, nil

	case "create_context":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateContext(name, strings.TrimSpace(description))

	case "set_active_context":
		name, _ := args["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.UseContext(name)

	// Context Pack tools
	case "list_context_packs":
		packType, _ := args["type"].(string)
		status, _ := args["status"].(string)
		query, _ := args["query"].(string)
		limit := toInt(args["limit"])

		response, err := client.ListContextPacks(
			strings.TrimSpace(packType),
			strings.TrimSpace(status),
			strings.TrimSpace(query),
			limit,
			0,
		)
		if err != nil {
			return nil, err
		}
		return response, nil

	case "get_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}
		return client.GetContextPack(packID)

	case "create_context_pack":
		name, _ := args["name"].(string)
		packType, _ := args["type"].(string)
		description, _ := args["description"].(string)
		status, _ := args["status"].(string)

		name = strings.TrimSpace(name)
		packType = strings.TrimSpace(packType)
		if name == "" {
			return nil, errors.New("name is required")
		}
		if packType == "" {
			packType = "custom"
		}

		// Parse tags
		var tags []string
		if tagsRaw, ok := args["tags"].([]interface{}); ok {
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
		}

		return client.CreateContextPack(name, packType, strings.TrimSpace(description), strings.TrimSpace(status), tags)

	case "update_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}

		updates := make(map[string]interface{})
		if name, ok := args["name"].(string); ok && strings.TrimSpace(name) != "" {
			updates["name"] = strings.TrimSpace(name)
		}
		if packType, ok := args["type"].(string); ok && strings.TrimSpace(packType) != "" {
			updates["type"] = strings.TrimSpace(packType)
		}
		if description, ok := args["description"].(string); ok {
			updates["description"] = strings.TrimSpace(description)
		}
		if status, ok := args["status"].(string); ok && strings.TrimSpace(status) != "" {
			updates["status"] = strings.TrimSpace(status)
		}
		if tagsRaw, ok := args["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
			updates["tags"] = tags
		}

		return client.UpdateContextPack(packID, updates)

	case "delete_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}
		if err := client.DeleteContextPack(packID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": packID}, nil

	case "activate_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}
		pack, err := client.SetActiveContextPack(packID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "pack": pack}, nil

	case "get_active_context_pack":
		return client.GetActiveContextPack()

	// Organization tools
	case "list_organizations":
		return client.ListOrganizations()

	case "get_organization":
		orgID, _ := args["orgId"].(string)
		orgID = strings.TrimSpace(orgID)
		if orgID == "" {
			return nil, errors.New("orgId is required")
		}
		return client.GetOrganization(orgID)

	case "create_organization":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateOrganization(name, strings.TrimSpace(description))

	// Decision tools - for agents to record architectural decisions
	case "list_decisions":
		status, _ := args["status"].(string)
		area, _ := args["area"].(string)
		limit := toInt(args["limit"])
		decisions, err := client.ListDecisions(strings.TrimSpace(status), strings.TrimSpace(area), limit)
		if err != nil {
			return nil, err
		}
		return decisions, nil

	case "get_decision":
		decisionID, _ := args["decisionId"].(string)
		decisionID = strings.TrimSpace(decisionID)
		if decisionID == "" {
			return nil, errors.New("decisionId is required")
		}
		return client.GetDecision(decisionID)

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

	case "update_decision":
		decisionID, _ := args["decisionId"].(string)
		decisionID = strings.TrimSpace(decisionID)
		if decisionID == "" {
			return nil, errors.New("decisionId is required")
		}

		updates := make(map[string]interface{})
		if title, ok := args["title"].(string); ok && strings.TrimSpace(title) != "" {
			updates["title"] = strings.TrimSpace(title)
		}
		if description, ok := args["description"].(string); ok {
			updates["description"] = strings.TrimSpace(description)
		}
		if status, ok := args["status"].(string); ok && strings.TrimSpace(status) != "" {
			updates["status"] = strings.TrimSpace(status)
		}
		if area, ok := args["area"].(string); ok && strings.TrimSpace(area) != "" {
			updates["area"] = strings.TrimSpace(area)
		}
		if context, ok := args["context"].(string); ok {
			updates["context"] = strings.TrimSpace(context)
		}
		if consequences, ok := args["consequences"].(string); ok {
			updates["consequences"] = strings.TrimSpace(consequences)
		}

		return client.UpdateDecision(decisionID, updates)

	case "delete_decision":
		decisionID, _ := args["decisionId"].(string)
		decisionID = strings.TrimSpace(decisionID)
		if decisionID == "" {
			return nil, errors.New("decisionId is required")
		}
		if err := client.DeleteDecision(decisionID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": decisionID}, nil

	// Agent Onboarding & Self-Documentation tools
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

	default:
		return nil, errors.New("tool not implemented")
	}
}

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

func resolveTaskIDList(client *api.Client, v interface{}) ([]string, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("taskIds must be an array")
	}
	if len(arr) == 0 {
		return nil, errors.New("taskIds cannot be empty")
	}
	ids := make([]string, 0, len(arr))
	for _, raw := range arr {
		s, _ := raw.(string)
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// backend supports short identifier in GET /tasks/:id, but bulk endpoints require full UUID
		t, err := client.GetTask(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, t.ID.String())
	}
	if len(ids) == 0 {
		return nil, errors.New("no valid task ids")
	}
	return ids, nil
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
		return "", errors.New("no active project")
	}

	if _, err := uuid.Parse(projectIdentifier); err == nil {
		return projectIdentifier, nil
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
		"version": "1.9.1",
		"tagline": "AI Agent Memory & Task Management System",
		"description": `Ramorie is a persistent memory and task management system designed for AI agents. 
It enables agents to maintain context across sessions, track tasks, store knowledge, and make architectural decisions.

Key Concepts:
- Projects: Organize work into logical groups
- Tasks: Track work items with status, priority, and progress
- Memories: Store knowledge, learnings, and context for future reference
- Decisions: Record architectural decisions (ADRs) for transparency
- Context Packs: Save and restore working contexts`,

		"capabilities": []string{
			"Persistent task management across sessions",
			"Knowledge/memory storage with auto-linking",
			"Project organization and switching",
			"Architectural decision records (ADRs)",
			"AI-powered task analysis (risks, dependencies)",
			"Context pack management for focus switching",
			"Multi-organization support",
			"Export and reporting capabilities",
		},

		"quickstart": map[string]interface{}{
			"1_first_call":    "get_ramorie_info (you're here!)",
			"2_setup":         "setup_agent - get current context and recommendations",
			"3_list_projects": "list_projects - see available projects",
			"4_get_tasks":     "get_next_tasks - see what needs attention",
			"5_work":          "start_task, add_memory, complete_task",
		},

		"rules": []string{
			"Always check active project before creating tasks",
			"Use add_memory to persist important information",
			"Link memories to tasks for context preservation",
			"Record architectural decisions with create_decision",
			"Never delete without explicit user approval",
			"Search before creating to avoid duplicates",
		},

		"tool_categories": map[string][]string{
			"project":      {"list_projects", "create_project", "get_project", "update_project", "delete_project", "set_active_project"},
			"task":         {"list_tasks", "create_task", "get_task", "start_task", "complete_task", "stop_task", "delete_task", "update_task_status", "update_progress", "add_task_note", "create_subtask", "search_tasks", "get_next_tasks", "get_active_task", "duplicate_task", "move_tasks_to_project", "bulk_start_tasks", "bulk_complete_tasks"},
			"memory":       {"list_memories", "add_memory", "get_memory", "update_memory", "delete_memory", "recall", "get_task_memories", "memory_tasks", "create_memory_task_link"},
			"context":      {"list_contexts", "create_context", "set_active_context"},
			"context_pack": {"list_context_packs", "get_context_pack", "create_context_pack", "update_context_pack", "delete_context_pack", "activate_context_pack", "get_active_context_pack"},
			"organization": {"list_organizations", "get_organization", "create_organization"},
			"decision":     {"list_decisions", "get_decision", "create_decision", "update_decision", "delete_decision"},
			"reports":      {"get_stats", "get_history", "timeline", "export_project"},
			"ai":           {"analyze_task_risks", "analyze_task_dependencies"},
			"agent":        {"get_ramorie_info", "get_cursor_rules", "setup_agent"},
		},
	}
}

func getCursorRules(format string) map[string]interface{} {
	rules := `# 🧠 Ramorie MCP Usage Rules

## Core Principle
**"If it matters later, it belongs in Ramorie."**

## Required Behaviors

### Session Start
1. Call ` + "`setup_agent`" + ` to get current context
2. Check active project with ` + "`list_projects`" + `
3. Review pending tasks with ` + "`get_next_tasks`" + `

### During Work
1. **Task Management:**
   - Search before creating (avoid duplicates)
   - Use ` + "`start_task`" + ` when beginning work
   - Update progress with ` + "`update_progress`" + `
   - Add notes with ` + "`add_task_note`" + ` for context
   - Complete with ` + "`complete_task`" + ` when done

2. **Memory/Knowledge:**
   - Store important discoveries with ` + "`add_memory`" + `
   - Memories auto-link to active task
   - Use ` + "`recall`" + ` to search past knowledge

3. **Decisions:**
   - Record architectural decisions with ` + "`create_decision`" + `
   - Include context, consequences, and area

### Session End
- Save session summary to memory
- Update task progress
- Note any blockers or next steps

## Forbidden Actions
- ❌ Never delete without explicit user approval
- ❌ Never modify completed tasks unless instructed
- ❌ Never bypass Ramorie for task/memory management
- ❌ Never create duplicate projects

## Tool Priority
1. ` + "`get_ramorie_info`" + ` - Understand the system
2. ` + "`setup_agent`" + ` - Get current context
3. ` + "`get_next_tasks`" + ` - See what needs attention
4. ` + "`add_memory`" + ` - Persist important information
5. ` + "`create_decision`" + ` - Record architectural choices
`

	result := map[string]interface{}{
		"format": format,
		"rules":  rules,
		"usage":  "Add this to your .cursorrules file or project rules",
	}

	if format == "json" {
		result["rules_structured"] = map[string]interface{}{
			"session_start": []string{
				"Call setup_agent to get current context",
				"Check active project with list_projects",
				"Review pending tasks with get_next_tasks",
			},
			"during_work": map[string][]string{
				"task_management": {
					"Search before creating tasks",
					"Use start_task when beginning work",
					"Update progress regularly",
					"Add notes for context",
					"Complete tasks when done",
				},
				"memory": {
					"Store important discoveries with add_memory",
					"Memories auto-link to active task",
					"Use recall to search past knowledge",
				},
				"decisions": {
					"Record architectural decisions",
					"Include context and consequences",
				},
			},
			"forbidden": []string{
				"Delete without explicit user approval",
				"Modify completed tasks unless instructed",
				"Bypass Ramorie for task/memory management",
				"Create duplicate projects",
			},
		}
	}

	return result
}

func setupAgent(client *api.Client) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"status":  "ready",
		"message": "Ramorie agent session initialized",
	}

	// Get active project
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.ActiveProjectID != "" {
		result["active_project_id"] = cfg.ActiveProjectID
	}

	// List projects
	projects, err := client.ListProjects()
	if err == nil {
		projectSummary := make([]map[string]interface{}, 0)
		for _, p := range projects {
			ps := map[string]interface{}{
				"id":        p.ID.String(),
				"name":      p.Name,
				"is_active": p.IsActive,
			}
			if p.IsActive {
				result["active_project"] = ps
			}
			projectSummary = append(projectSummary, ps)
		}
		result["projects_count"] = len(projects)
		result["projects"] = projectSummary
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
	recommendations := []string{
		"Call get_next_tasks to see prioritized work items",
	}
	if result["active_project"] == nil {
		recommendations = append(recommendations, "Set an active project with set_active_project")
	}
	if result["active_task"] == nil {
		recommendations = append(recommendations, "Start a task with start_task to enable memory auto-linking")
	}
	result["recommendations"] = recommendations

	return result, nil
}
