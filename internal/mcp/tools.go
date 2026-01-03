package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
)

// ToolInput is a generic input struct for tools that use map[string]interface{}
type ToolInput struct {
	Args map[string]interface{} `json:"-"`
}

// registerTools registers all MCP tools with the server using go-sdk
// The SDK automatically infers InputSchema from the handler's input struct type
func registerTools(server *mcp.Server) {
	// ============================================================================
	// 🔴 ESSENTIAL - Agent Onboarding
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_ramorie_info",
		Description: "🔴 ESSENTIAL | 🧠 CALL THIS FIRST! Get comprehensive information about Ramorie - what it is, how to use it, and agent guidelines.",
	}, handleGetRamorieInfo)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "setup_agent",
		Description: "🔴 ESSENTIAL | Initialize agent session. Returns current context, active project, pending tasks, and recommended actions.",
	}, handleSetupAgent)

	// ============================================================================
	// 🔴 ESSENTIAL - Project Management
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "🔴 ESSENTIAL | List all projects. Check this to see available projects and which one is active.",
	}, handleListProjects)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_active_project",
		Description: "🔴 ESSENTIAL | Set the active project. All new tasks and memories will be created in this project.",
	}, handleSetActiveProject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_project",
		Description: "🟢 ADVANCED | Create a new project. ⚠️ Check list_projects first - don't create duplicates!",
	}, handleCreateProject)

	// ============================================================================
	// 🔴 ESSENTIAL - Task Management
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "🔴 ESSENTIAL | List tasks with filtering. 💡 Call before create_task to check for duplicates.",
	}, handleListTasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_task",
		Description: "🔴 ESSENTIAL | Create a new task. ⚠️ Always check list_tasks first to avoid duplicates!",
	}, handleCreateTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task",
		Description: "🔴 ESSENTIAL | Get task details including notes and metadata.",
	}, handleGetTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_task",
		Description: "🔴 ESSENTIAL | Start working on a task. Sets status to IN_PROGRESS and enables memory auto-linking.",
	}, handleStartTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "complete_task",
		Description: "🔴 ESSENTIAL | Mark task as completed. Use when work is finished.",
	}, handleCompleteTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_task",
		Description: "🟢 ADVANCED | Pause a task. Clears active task, keeps IN_PROGRESS status.",
	}, handleStopTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_next_tasks",
		Description: "🔴 ESSENTIAL | Get prioritized TODO tasks. 💡 Use at session start to see what needs attention.",
	}, handleGetNextTasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_task_note",
		Description: "🟡 COMMON | Add a note/annotation to a task. Use for progress updates or context.",
	}, handleAddTaskNote)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_progress",
		Description: "🟡 COMMON | Update task progress percentage (0-100).",
	}, handleUpdateProgress)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_tasks",
		Description: "🟡 COMMON | Search tasks by keyword. Use to find specific tasks.",
	}, handleSearchTasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_active_task",
		Description: "🟡 COMMON | Get the currently active task. Memories auto-link to this task.",
	}, handleGetActiveTask)

	// ============================================================================
	// 🔴 ESSENTIAL - Memory Management
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_memory",
		Description: "🔴 ESSENTIAL | Store important information to knowledge base. Auto-links to active task. 💡 If it matters later, add it here!",
	}, handleAddMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_memories",
		Description: "🔴 ESSENTIAL | List memories with optional filtering by project or term.",
	}, handleListMemories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_memory",
		Description: "🟡 COMMON | Get memory details by ID.",
	}, handleGetMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall",
		Description: "🟡 COMMON | Advanced memory search with multi-word support, filters, and relations. Supports: OR search (space-separated), AND search (comma-separated), project/tag filtering.",
	}, handleRecall)

	// ============================================================================
	// 🔴 ESSENTIAL - Focus Management
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_focus",
		Description: "🔴 ESSENTIAL | Get user's current focus (active workspace). Returns the active context pack and its details.",
	}, handleGetFocus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_focus",
		Description: "🔴 ESSENTIAL | Set user's active focus (workspace). Switch to a different context pack.",
	}, handleSetFocus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "clear_focus",
		Description: "🔴 ESSENTIAL | Clear user's active focus. Deactivates the current context pack.",
	}, handleClearFocus)

	// ============================================================================
	// 🟡 COMMON - Decisions (ADRs)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_decision",
		Description: "🟡 COMMON | Record an architectural decision (ADR). Use for important technical choices.",
	}, handleCreateDecision)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_decisions",
		Description: "🟡 COMMON | List architectural decisions. Review past decisions before making new ones.",
	}, handleListDecisions)

	// ============================================================================
	// 🟡 COMMON - Reports
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stats",
		Description: "🟡 COMMON | Get task statistics and completion rates.",
	}, handleGetStats)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "export_project",
		Description: "🟢 ADVANCED | Export project report in markdown format.",
	}, handleExportProject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_cursor_rules",
		Description: "🟢 ADVANCED | Get Cursor IDE rules for Ramorie. Returns markdown for .cursorrules file.",
	}, handleGetCursorRules)
}

// ============================================================================
// TOOL HANDLER FUNCTIONS
// ============================================================================

type EmptyInput struct{}
type EmptyOutput struct{}

type TextOutput struct {
	Text string `json:"text"`
}

func handleGetRamorieInfo(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	return nil, getRamorieInfo(), nil
}

func handleSetupAgent(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	result, err := setupAgent(apiClient)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

func handleListProjects(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, interface{}, error) {
	projects, err := apiClient.ListProjects()
	if err != nil {
		return nil, nil, err
	}
	return nil, projects, nil
}

type SetActiveProjectInput struct {
	ProjectName string `json:"projectName"`
}

func handleSetActiveProject(ctx context.Context, req *mcp.CallToolRequest, input SetActiveProjectInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	projectName := strings.TrimSpace(input.ProjectName)
	if projectName == "" {
		return nil, nil, errors.New("projectName is required")
	}
	projects, err := apiClient.ListProjects()
	if err != nil {
		return nil, nil, err
	}
	for _, p := range projects {
		if p.Name == projectName || strings.HasPrefix(p.ID.String(), projectName) {
			if err := apiClient.SetProjectActive(p.ID.String()); err != nil {
				return nil, nil, err
			}
			cfg, _ := config.LoadConfig()
			if cfg == nil {
				cfg = &config.Config{}
			}
			cfg.ActiveProjectID = p.ID.String()
			_ = config.SaveConfig(cfg)
			return nil, map[string]interface{}{"ok": true, "project_id": p.ID.String(), "name": p.Name}, nil
		}
	}
	return nil, nil, errors.New("project not found")
}

type CreateProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func handleCreateProject(ctx context.Context, req *mcp.CallToolRequest, input CreateProjectInput) (*mcp.CallToolResult, interface{}, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}
	project, err := apiClient.CreateProject(name, strings.TrimSpace(input.Description))
	if err != nil {
		return nil, nil, err
	}
	return nil, project, nil
}

type ListTasksInput struct {
	Status  string  `json:"status"`
	Project string  `json:"project"`
	Limit   float64 `json:"limit"`
}

func handleListTasks(ctx context.Context, req *mcp.CallToolRequest, input ListTasksInput) (*mcp.CallToolResult, interface{}, error) {
	projectID := ""
	if strings.TrimSpace(input.Project) != "" {
		pid, err := resolveProjectID(apiClient, input.Project)
		if err != nil {
			return nil, nil, err
		}
		projectID = pid
	}
	tasks, err := apiClient.ListTasks(projectID, strings.TrimSpace(input.Status))
	if err != nil {
		return nil, nil, err
	}
	limit := int(input.Limit)
	if limit > 0 && limit < len(tasks) {
		tasks = tasks[:limit]
	}
	return nil, tasks, nil
}

type CreateTaskInput struct {
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Project     string `json:"project"`
}

func handleCreateTask(ctx context.Context, req *mcp.CallToolRequest, input CreateTaskInput) (*mcp.CallToolResult, interface{}, error) {
	description := strings.TrimSpace(input.Description)
	if description == "" {
		return nil, nil, errors.New("description is required")
	}
	priority := normalizePriority(input.Priority)
	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}
	task, err := apiClient.CreateTask(projectID, description, "", priority)
	if err != nil {
		return nil, nil, err
	}
	return nil, task, nil
}

type TaskIDInput struct {
	TaskID string `json:"taskId"`
}

func handleGetTask(ctx context.Context, req *mcp.CallToolRequest, input TaskIDInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	task, err := apiClient.GetTask(taskID)
	if err != nil {
		return nil, nil, err
	}
	return nil, task, nil
}

func handleStartTask(ctx context.Context, req *mcp.CallToolRequest, input TaskIDInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	if err := apiClient.StartTask(taskID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]interface{}{"ok": true, "message": "Task started. Memories will now auto-link to this task."}, nil
}

func handleCompleteTask(ctx context.Context, req *mcp.CallToolRequest, input TaskIDInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	if err := apiClient.CompleteTask(taskID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]interface{}{"ok": true}, nil
}

func handleStopTask(ctx context.Context, req *mcp.CallToolRequest, input TaskIDInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	if err := apiClient.StopTask(taskID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]interface{}{"ok": true}, nil
}

type GetNextTasksInput struct {
	Count   float64 `json:"count"`
	Project string  `json:"project"`
}

func handleGetNextTasks(ctx context.Context, req *mcp.CallToolRequest, input GetNextTasksInput) (*mcp.CallToolResult, interface{}, error) {
	count := int(input.Count)
	if count <= 0 {
		count = 5
	}
	projectID := ""
	if strings.TrimSpace(input.Project) != "" {
		pid, err := resolveProjectID(apiClient, input.Project)
		if err != nil {
			return nil, nil, err
		}
		projectID = pid
	}
	tasks, err := apiClient.ListTasksQuery(projectID, "TODO", "", nil, nil)
	if err != nil {
		return nil, nil, err
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
	return nil, tasks, nil
}

type AddTaskNoteInput struct {
	TaskID string `json:"taskId"`
	Note   string `json:"note"`
}

func handleAddTaskNote(ctx context.Context, req *mcp.CallToolRequest, input AddTaskNoteInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	note := strings.TrimSpace(input.Note)
	if taskID == "" || note == "" {
		return nil, nil, errors.New("taskId and note are required")
	}
	annotation, err := apiClient.CreateAnnotation(taskID, note)
	if err != nil {
		return nil, nil, err
	}
	return nil, annotation, nil
}

type UpdateProgressInput struct {
	TaskID   string  `json:"taskId"`
	Progress float64 `json:"progress"`
}

func handleUpdateProgress(ctx context.Context, req *mcp.CallToolRequest, input UpdateProgressInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	progress := int(input.Progress)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	if progress < 0 || progress > 100 {
		return nil, nil, errors.New("progress must be between 0 and 100")
	}
	result, err := apiClient.UpdateTask(taskID, map[string]interface{}{"progress": progress})
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

type SearchTasksInput struct {
	Query   string  `json:"query"`
	Status  string  `json:"status"`
	Project string  `json:"project"`
	Limit   float64 `json:"limit"`
}

func handleSearchTasks(ctx context.Context, req *mcp.CallToolRequest, input SearchTasksInput) (*mcp.CallToolResult, interface{}, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, nil, errors.New("query is required")
	}
	projectID := ""
	if strings.TrimSpace(input.Project) != "" {
		pid, err := resolveProjectID(apiClient, input.Project)
		if err != nil {
			return nil, nil, err
		}
		projectID = pid
	}
	tasks, err := apiClient.ListTasksQuery(projectID, strings.TrimSpace(input.Status), query, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	limit := int(input.Limit)
	if limit > 0 && limit < len(tasks) {
		tasks = tasks[:limit]
	}
	return nil, tasks, nil
}

func handleGetActiveTask(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, interface{}, error) {
	task, err := apiClient.GetActiveTask()
	if err != nil {
		return nil, nil, err
	}
	return nil, task, nil
}

type AddMemoryInput struct {
	Content string `json:"content"`
	Project string `json:"project"`
}

func handleAddMemory(ctx context.Context, req *mcp.CallToolRequest, input AddMemoryInput) (*mcp.CallToolResult, interface{}, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, nil, errors.New("content is required")
	}
	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}
	memory, err := apiClient.CreateMemory(projectID, content)
	if err != nil {
		return nil, nil, err
	}
	return nil, memory, nil
}

type ListMemoriesInput struct {
	Project string  `json:"project"`
	Term    string  `json:"term"`
	Limit   float64 `json:"limit"`
}

func handleListMemories(ctx context.Context, req *mcp.CallToolRequest, input ListMemoriesInput) (*mcp.CallToolResult, interface{}, error) {
	projectID := ""
	if strings.TrimSpace(input.Project) != "" {
		pid, err := resolveProjectID(apiClient, input.Project)
		if err != nil {
			return nil, nil, err
		}
		projectID = pid
	}
	memories, err := apiClient.ListMemories(projectID, "")
	if err != nil {
		return nil, nil, err
	}
	term := strings.TrimSpace(input.Term)
	if term != "" {
		filtered := memories[:0]
		for _, m := range memories {
			if strings.Contains(strings.ToLower(m.Content), strings.ToLower(term)) {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}
	limit := int(input.Limit)
	if limit > 0 && limit < len(memories) {
		memories = memories[:limit]
	}
	return nil, memories, nil
}

type GetMemoryInput struct {
	MemoryID string `json:"memoryId"`
}

func handleGetMemory(ctx context.Context, req *mcp.CallToolRequest, input GetMemoryInput) (*mcp.CallToolResult, interface{}, error) {
	memoryID := strings.TrimSpace(input.MemoryID)
	if memoryID == "" {
		return nil, nil, errors.New("memoryId is required")
	}
	memory, err := apiClient.GetMemory(memoryID)
	if err != nil {
		return nil, nil, err
	}
	return nil, memory, nil
}

type RecallInput struct {
	Term             string  `json:"term"`
	Project          string  `json:"project"`
	Tag              string  `json:"tag"`
	LinkedTask       bool    `json:"linked_task"`
	IncludeRelations bool    `json:"include_relations"`
	Limit            float64 `json:"limit"`
	MinScore         float64 `json:"min_score"`
}

func handleRecall(ctx context.Context, req *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	term := strings.TrimSpace(input.Term)
	if term == "" {
		return nil, nil, errors.New("term is required")
	}

	limit := int(input.Limit)
	if limit == 0 {
		limit = 20
	}
	minScore := int(input.MinScore)
	includeRelations := true
	if !input.IncludeRelations && input.Limit > 0 {
		includeRelations = input.IncludeRelations
	}

	projectID := ""
	if strings.TrimSpace(input.Project) != "" {
		pid, err := resolveProjectID(apiClient, input.Project)
		if err == nil {
			projectID = pid
		}
	}

	memories, err := apiClient.ListMemories(projectID, "")
	if err != nil {
		return nil, nil, err
	}

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

	type scoredMemory struct {
		memory interface{}
		score  int
	}
	var scored []scoredMemory

	for _, m := range memories {
		if input.LinkedTask && m.LinkedTaskID == nil {
			continue
		}

		if input.Tag != "" {
			hasTag := false
			if tags, ok := m.Tags.([]interface{}); ok {
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						if strings.EqualFold(tagStr, input.Tag) {
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

		contentLower := strings.ToLower(m.Content)
		score := 0
		matchCount := 0

		for _, t := range searchTerms {
			if strings.Contains(contentLower, t) {
				matchCount++
				score += 20
				if strings.Contains(contentLower, " "+t+" ") ||
					strings.HasPrefix(contentLower, t+" ") ||
					strings.HasSuffix(contentLower, " "+t) {
					score += 10
				}
				if strings.Contains(contentLower, "## "+t) ||
					strings.Contains(contentLower, "### "+t) {
					score += 15
				}
				occurrences := strings.Count(contentLower, t)
				if occurrences > 1 {
					score += min(occurrences*5, 25)
				}
			}
		}

		if isAndSearch && matchCount < len(searchTerms) {
			continue
		}
		if !isAndSearch && matchCount == 0 {
			continue
		}

		if m.LinkedTaskID != nil {
			score += 5
		}

		if score < minScore {
			continue
		}

		result := map[string]interface{}{
			"id":         m.ID.String(),
			"content":    m.Content,
			"score":      score,
			"created_at": m.CreatedAt,
		}

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

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	var results []interface{}
	for i, s := range scored {
		if i >= limit {
			break
		}
		results = append(results, s.memory)
	}

	return nil, map[string]interface{}{
		"term":        term,
		"search_mode": map[bool]string{true: "AND", false: "OR"}[isAndSearch],
		"count":       len(results),
		"total_found": len(scored),
		"results":     results,
	}, nil
}

func handleGetFocus(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	focus, err := apiClient.GetFocus()
	if err != nil {
		return nil, nil, err
	}
	if focus.ActivePack == nil {
		return nil, map[string]interface{}{
			"active_context_pack_id": nil,
			"active_pack":            nil,
			"message":                "No active focus set. Use set_focus to activate a context pack.",
		}, nil
	}
	return nil, map[string]interface{}{
		"active_context_pack_id": focus.ActiveContextPackID,
		"active_pack": map[string]interface{}{
			"id":             focus.ActivePack.ID,
			"name":           focus.ActivePack.Name,
			"description":    focus.ActivePack.Description,
			"type":           focus.ActivePack.Type,
			"status":         focus.ActivePack.Status,
			"contexts_count": focus.ActivePack.ContextsCount,
			"memories_count": focus.ActivePack.MemoriesCount,
			"tasks_count":    focus.ActivePack.TasksCount,
			"contexts":       focus.ActivePack.Contexts,
		},
	}, nil
}

type SetFocusInput struct {
	PackID string `json:"packId"`
}

func handleSetFocus(ctx context.Context, req *mcp.CallToolRequest, input SetFocusInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	packID := strings.TrimSpace(input.PackID)
	if packID == "" {
		return nil, nil, errors.New("packId is required")
	}
	focus, err := apiClient.SetFocus(packID)
	if err != nil {
		return nil, nil, err
	}
	result := map[string]interface{}{
		"ok":      true,
		"message": "Focus updated successfully",
	}
	if focus.ActivePack != nil {
		result["active_pack"] = map[string]interface{}{
			"id":             focus.ActivePack.ID,
			"name":           focus.ActivePack.Name,
			"contexts_count": focus.ActivePack.ContextsCount,
			"memories_count": focus.ActivePack.MemoriesCount,
			"tasks_count":    focus.ActivePack.TasksCount,
		}
	}
	return nil, result, nil
}

func handleClearFocus(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	if err := apiClient.ClearFocus(); err != nil {
		return nil, nil, err
	}
	return nil, map[string]interface{}{
		"ok":      true,
		"message": "Focus cleared",
	}, nil
}

type CreateDecisionInput struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	Area         string `json:"area"`
	Context      string `json:"context"`
	Consequences string `json:"consequences"`
}

func handleCreateDecision(ctx context.Context, req *mcp.CallToolRequest, input CreateDecisionInput) (*mcp.CallToolResult, interface{}, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}
	decision, err := apiClient.CreateDecision(
		title,
		strings.TrimSpace(input.Description),
		strings.TrimSpace(input.Status),
		strings.TrimSpace(input.Area),
		strings.TrimSpace(input.Context),
		strings.TrimSpace(input.Consequences),
	)
	if err != nil {
		return nil, nil, err
	}
	return nil, decision, nil
}

type ListDecisionsInput struct {
	Status string  `json:"status"`
	Area   string  `json:"area"`
	Limit  float64 `json:"limit"`
}

func handleListDecisions(ctx context.Context, req *mcp.CallToolRequest, input ListDecisionsInput) (*mcp.CallToolResult, interface{}, error) {
	decisions, err := apiClient.ListDecisions(strings.TrimSpace(input.Status), strings.TrimSpace(input.Area), int(input.Limit))
	if err != nil {
		return nil, nil, err
	}
	return nil, decisions, nil
}

type GetStatsInput struct {
	Project string `json:"project"`
}

func handleGetStats(ctx context.Context, req *mcp.CallToolRequest, input GetStatsInput) (*mcp.CallToolResult, interface{}, error) {
	b, err := apiClient.Request("GET", "/reports/stats", nil)
	if err != nil {
		return nil, nil, err
	}
	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, nil, errors.New("invalid stats response")
	}
	return nil, out, nil
}

type ExportProjectInput struct {
	Project string `json:"project"`
	Format  string `json:"format"`
}

func handleExportProject(ctx context.Context, req *mcp.CallToolRequest, input ExportProjectInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	format := input.Format
	if format == "" {
		format = "markdown"
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	projects, err := apiClient.ListProjects()
	if err != nil {
		return nil, nil, err
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
		return nil, nil, errors.New("project not found")
	}

	tasks, err := apiClient.ListTasks(projectID, "")
	if err != nil {
		return nil, nil, err
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

	return nil, map[string]interface{}{
		"project":  project.Name,
		"format":   format,
		"markdown": sb.String(),
	}, nil
}

type GetCursorRulesInput struct {
	Format string `json:"format"`
}

func handleGetCursorRules(ctx context.Context, req *mcp.CallToolRequest, input GetCursorRulesInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	format := input.Format
	if format == "" {
		format = "markdown"
	}
	return nil, getCursorRules(format), nil
}

// ============================================================================
// LEGACY SUPPORT - ToolDefinitions for CLI tools command
// ============================================================================

type toolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

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
		// 🔴 ESSENTIAL - Focus Management (SINGLE SOURCE OF TRUTH for active workspace)
		// ============================================================================
		{
			Name:        "get_focus",
			Description: "🔴 ESSENTIAL | Get user's current focus (active workspace). Returns the active context pack and its details.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "set_focus",
			Description: "🔴 ESSENTIAL | Set user's active focus (workspace). Switch to a different context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string", "description": "Context pack ID to activate"}}, "required": []string{"packId"}},
		},
		{
			Name:        "clear_focus",
			Description: "🔴 ESSENTIAL | Clear user's active focus. Deactivates the current context pack.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
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
		"version": "2.1.0",
		"tagline": "AI Agent Memory & Task Management System",
		"description": `Ramorie is a persistent memory and task management system for AI agents.
It enables context preservation across sessions, task tracking, and knowledge storage.`,

		"tool_count": 28,
		"tool_priority_guide": map[string]string{
			"🔴 ESSENTIAL": "Core functionality - use these regularly",
			"🟡 COMMON":    "Frequently used - call when needed",
			"🟢 ADVANCED":  "Specialized - only for specific scenarios",
		},

		"quickstart": []string{
			"1. setup_agent → Get current context and recommendations",
			"2. get_focus → Check your current active workspace",
			"3. list_projects → See available projects",
			"4. set_active_project → Set your working project",
			"5. get_next_tasks → See prioritized TODO tasks",
			"6. start_task → Begin working (enables memory auto-link)",
			"7. add_memory → Store important discoveries",
			"8. complete_task → Mark work as done",
		},

		"core_rules": []string{
			"✅ Always check list_tasks before creating new tasks",
			"✅ Use add_memory to persist important information",
			"✅ Start a task before adding memories for auto-linking",
			"✅ Use get_focus to check current workspace context",
			"✅ Record architectural decisions with create_decision",
			"❌ Never delete without explicit user approval",
			"❌ Never create duplicate projects",
		},

		"tools_by_category": map[string][]string{
			"🔴 agent":    {"get_ramorie_info", "setup_agent"},
			"🔴 focus":    {"get_focus", "set_focus", "clear_focus"},
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
2. ` + "`get_focus`" + ` - Check active workspace
3. ` + "`list_projects`" + ` - Check available projects
4. ` + "`get_next_tasks`" + ` - See what needs attention

### During Work
1. ` + "`start_task`" + ` - Begin working (enables memory auto-link)
2. ` + "`add_memory`" + ` - Store important discoveries
3. ` + "`add_task_note`" + ` - Add progress notes
4. ` + "`complete_task`" + ` - Mark as done

### Key Rules
- ✅ Check ` + "`list_tasks`" + ` before creating new tasks
- ✅ Use ` + "`add_memory`" + ` for important information
- ✅ Use ` + "`get_focus`" + ` to check current workspace
- ✅ Record decisions with ` + "`create_decision`" + `
- ❌ Never delete without user approval
- ❌ Never create duplicate projects

## Available Tools (28 total)

### 🔴 ESSENTIAL (15)
- get_ramorie_info, setup_agent
- get_focus, set_focus, clear_focus
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
		"version": "2.1.0",
	}

	// Get active project
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.ActiveProjectID != "" {
		result["active_project_id"] = cfg.ActiveProjectID
	}

	// Get current focus (active workspace)
	focus, err := client.GetFocus()
	if err == nil && focus != nil && focus.ActivePack != nil {
		result["active_focus"] = map[string]interface{}{
			"pack_id":        focus.ActiveContextPackID,
			"pack_name":      focus.ActivePack.Name,
			"contexts_count": focus.ActivePack.ContextsCount,
			"memories_count": focus.ActivePack.MemoriesCount,
			"tasks_count":    focus.ActivePack.TasksCount,
		}
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
	if result["active_focus"] == nil {
		recommendations = append(recommendations, "💡 Set an active focus: set_focus (for workspace context)")
	}
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
