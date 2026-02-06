package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ============================================================================
// UNIFIED TOOL HANDLERS - Simplified from 48 to 15 tools
// ============================================================================

// --- unified_task: list/get/create/start/complete/stop/progress/note/move ---

type UnifiedTaskInput struct {
	Action      string  `json:"action"`                     // list, get, create, start, complete, stop, progress, note, move
	Project     string  `json:"project,omitempty"`          // For list, create
	TaskID      string  `json:"taskId,omitempty"`           // For get, start, complete, stop, progress, note, move
	Description string  `json:"description,omitempty"`      // For create, note
	Priority    string  `json:"priority,omitempty"`         // For create (L/M/H)
	Progress    float64 `json:"progress,omitempty"`         // For progress (0-100)
	Status      string  `json:"status,omitempty"`           // For list filter
	Query       string  `json:"query,omitempty"`            // For list (keyword search)
	NextPriority bool   `json:"next_priority,omitempty"`    // For list (prioritized TODO)
	ProjectID   string  `json:"projectId,omitempty"`        // For move (target project)
	Limit       float64 `json:"limit,omitempty"`
	Cursor      string  `json:"cursor,omitempty"`
}

func handleUnifiedTask(ctx context.Context, req *mcp.CallToolRequest, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "list" // Default to list
	}

	switch action {
	case "list":
		return handleTaskList(ctx, input)
	case "get":
		return handleTaskGet(ctx, input)
	case "create":
		return handleTaskCreate(ctx, input)
	case "start":
		return handleTaskStart(ctx, input)
	case "complete":
		return handleTaskComplete(ctx, input)
	case "stop":
		return handleTaskStop(ctx, input)
	case "progress":
		return handleTaskProgress(ctx, input)
	case "note":
		return handleTaskNote(ctx, input)
	case "move":
		return handleTaskMove(ctx, input)
	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: list, get, create, start, complete, stop, progress, note, or move", action)
	}
}

func handleTaskList(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("'project' parameter is REQUIRED for list action. Use list_projects to see available projects.")
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	query := strings.TrimSpace(input.Query)
	status := strings.TrimSpace(input.Status)

	var tasks []models.Task
	if input.NextPriority {
		if status == "" {
			status = "TODO"
		}
		tasks, err = apiClient.ListTasksQuery(projectID, status, query, nil, nil)
		if err != nil {
			return nil, nil, err
		}
		// Sort by priority
		sortTasksByPriority(tasks)
	} else if query != "" {
		tasks, err = apiClient.ListTasksQuery(projectID, status, query, nil, nil)
		if err != nil {
			return nil, nil, err
		}
	} else {
		tasks, err = apiClient.ListTasks(projectID, status)
		if err != nil {
			return nil, nil, err
		}
	}

	limit := int(input.Limit)
	if input.NextPriority && limit <= 0 {
		limit = 5
	}
	if limit <= 0 {
		limit = 20
	}

	paginatedTasks, nextCursor, total := paginateSlice(tasks, input.Cursor, limit)

	var decryptedTasks []map[string]interface{}
	for _, t := range paginatedTasks {
		decryptedTitle, decryptedDesc := decryptTaskFields(&t)
		taskMap := map[string]interface{}{
			"id":          t.ID.String(),
			"project_id":  t.ProjectID.String(),
			"title":       decryptedTitle,
			"description": decryptedDesc,
			"status":      t.Status,
			"priority":    t.Priority,
			"created_at":  t.CreatedAt,
		}
		if t.Project != nil {
			taskMap["project"] = map[string]interface{}{
				"id":   t.Project.ID.String(),
				"name": t.Project.Name,
			}
		}
		decryptedTasks = append(decryptedTasks, taskMap)
	}

	return mustTextResult(formatPaginatedResponse(decryptedTasks, nextCursor, total, getContextString())), nil, nil
}

func handleTaskGet(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required for get action")
	}
	task, err := apiClient.GetTask(taskID)
	if err != nil {
		return nil, nil, err
	}

	decryptedTitle, decryptedDesc := decryptTaskFields(task)
	result := map[string]interface{}{
		"id":          task.ID.String(),
		"project_id":  task.ProjectID.String(),
		"title":       decryptedTitle,
		"description": decryptedDesc,
		"status":      task.Status,
		"priority":    task.Priority,
		"created_at":  task.CreatedAt,
		"updated_at":  task.UpdatedAt,
	}
	if task.Project != nil {
		result["project"] = map[string]interface{}{
			"id":   task.Project.ID.String(),
			"name": task.Project.Name,
		}
	}
	if len(task.Annotations) > 0 {
		result["annotations"] = task.Annotations
	}

	return mustTextResult(result), nil, nil
}

func handleTaskCreate(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	if err := checkSessionInit("task"); err != nil {
		return nil, nil, err
	}

	description := strings.TrimSpace(input.Description)
	if description == "" {
		return nil, nil, errors.New("description is required for create action")
	}

	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("'project' parameter is REQUIRED for create action. Use list_projects to see available projects.")
	}

	priority := normalizePriority(input.Priority)
	projectID, orgID, err := resolveProjectWithOrg(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	session := GetCurrentSession()
	var meta *api.AgentMetadata
	if session != nil && session.Initialized {
		meta = &api.AgentMetadata{
			AgentName:  session.AgentName,
			AgentModel: session.AgentModel,
			SessionID:  session.ID,
			CreatedVia: "mcp",
		}
	}

	// Personal project only — encrypt with personal key (org projects skip encryption)
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled && orgID == "" && crypto.IsVaultUnlocked() {
		encryptedDesc, nonce, isEncrypted, err := crypto.EncryptContent(description)
		if err == nil && isEncrypted {
			task, err := apiClient.CreateEncryptedTaskWithMeta(projectID, encryptedDesc, nonce, priority, meta)
			if err != nil {
				return nil, nil, err
			}
			return mustTextResult(map[string]interface{}{
				"action":    "created",
				"task":      task,
				"encrypted": true,
				"_message":  "Task created (encrypted)",
			}), nil, nil
		}
	}

	task, err := apiClient.CreateTaskWithMeta(projectID, description, "", priority, meta)
	if err != nil {
		return nil, nil, err
	}

	return mustTextResult(map[string]interface{}{
		"action":   "created",
		"task":     task,
		"_message": "Task created successfully",
	}), nil, nil
}

func handleTaskStart(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required for start action")
	}
	if err := apiClient.StartTask(taskID); err != nil {
		return nil, nil, err
	}
	return mustTextResult(map[string]interface{}{"ok": true, "message": "Task started. Memories will now auto-link."}), nil, nil
}

func handleTaskComplete(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required for complete action")
	}
	if err := apiClient.CompleteTask(taskID); err != nil {
		return nil, nil, err
	}
	return mustTextResult(map[string]interface{}{"ok": true, "message": "Task completed."}), nil, nil
}

func handleTaskStop(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required for stop action")
	}
	if err := apiClient.StopTask(taskID); err != nil {
		return nil, nil, err
	}
	return mustTextResult(map[string]interface{}{"ok": true, "message": "Task stopped."}), nil, nil
}

func handleTaskProgress(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required for progress action")
	}
	progress := int(input.Progress)
	if progress < 0 || progress > 100 {
		return nil, nil, errors.New("progress must be between 0 and 100")
	}
	result, err := apiClient.UpdateTask(taskID, map[string]interface{}{"progress": progress})
	if err != nil {
		return nil, nil, err
	}
	return mustTextResult(map[string]interface{}{
		"ok":      true,
		"message": fmt.Sprintf("Progress updated to %d%%", progress),
		"task":    result,
	}), nil, nil
}

func handleTaskNote(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	note := strings.TrimSpace(input.Description)
	if taskID == "" || note == "" {
		return nil, nil, errors.New("taskId and description (note content) are required for note action")
	}

	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled && crypto.IsVaultUnlocked() {
		encryptedNote, nonce, isEncrypted, err := crypto.EncryptContent(note)
		if err == nil && isEncrypted {
			annotation, err := apiClient.CreateEncryptedAnnotation(taskID, encryptedNote, nonce)
			if err != nil {
				return nil, nil, err
			}
			return mustTextResult(annotation), nil, nil
		}
	}

	annotation, err := apiClient.CreateAnnotation(taskID, note)
	if err != nil {
		return nil, nil, err
	}
	return mustTextResult(annotation), nil, nil
}

func handleTaskMove(ctx context.Context, input UnifiedTaskInput) (*mcp.CallToolResult, interface{}, error) {
	if err := checkSessionInit("task"); err != nil {
		return nil, nil, err
	}

	taskID := strings.TrimSpace(input.TaskID)
	targetProject := strings.TrimSpace(input.ProjectID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required for move action")
	}
	if targetProject == "" {
		return nil, nil, errors.New("projectId is required for move action")
	}

	projectID, err := resolveProjectID(apiClient, targetProject)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve project: %w", err)
	}

	task, err := apiClient.UpdateTask(taskID, map[string]interface{}{"project_id": projectID})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to move task: %w", err)
	}

	return mustTextResult(map[string]interface{}{
		"ok":      true,
		"message": "Task moved successfully",
		"task":    task,
	}), nil, nil
}

// --- unified_memory: list/get ---

type UnifiedMemoryInput struct {
	Action   string  `json:"action"`           // list, get
	Project  string  `json:"project"`          // For list
	MemoryID string  `json:"memoryId"`         // For get
	Term     string  `json:"term,omitempty"`   // For list filter
	Limit    float64 `json:"limit,omitempty"`
	Cursor   string  `json:"cursor,omitempty"`
}

func handleUnifiedMemory(ctx context.Context, req *mcp.CallToolRequest, input UnifiedMemoryInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "list":
		if strings.TrimSpace(input.Project) == "" {
			return nil, nil, errors.New("'project' parameter is REQUIRED for list action")
		}

		projectID, err := resolveProjectID(apiClient, input.Project)
		if err != nil {
			return nil, nil, err
		}
		memories, err := apiClient.ListMemories(projectID, "")
		if err != nil {
			return nil, nil, err
		}

		term := strings.TrimSpace(input.Term)
		if term != "" {
			filtered := memories[:0]
			for _, m := range memories {
				decryptedContent := decryptMemoryContent(&m)
				if strings.Contains(strings.ToLower(decryptedContent), strings.ToLower(term)) {
					filtered = append(filtered, m)
				}
			}
			memories = filtered
		}

		limit := int(input.Limit)
		if limit <= 0 {
			limit = 20
		}

		paginatedMemories, nextCursor, total := paginateSlice(memories, input.Cursor, limit)

		var decryptedMemories []map[string]interface{}
		for _, m := range paginatedMemories {
			decryptedContent := decryptMemoryContent(&m)
			memMap := map[string]interface{}{
				"id":         m.ID.String(),
				"project_id": m.ProjectID.String(),
				"content":    decryptedContent,
				"created_at": m.CreatedAt,
			}
			if m.Project != nil {
				memMap["project"] = map[string]interface{}{
					"id":   m.Project.ID.String(),
					"name": m.Project.Name,
				}
			}
			decryptedMemories = append(decryptedMemories, memMap)
		}

		return mustTextResult(formatPaginatedResponse(decryptedMemories, nextCursor, total, getContextString())), nil, nil

	case "get":
		memoryID := strings.TrimSpace(input.MemoryID)
		if memoryID == "" {
			return nil, nil, errors.New("memoryId is required for get action")
		}
		memory, err := apiClient.GetMemory(memoryID)
		if err != nil {
			return nil, nil, err
		}

		decryptedContent := decryptMemoryContent(memory)
		result := map[string]interface{}{
			"id":         memory.ID.String(),
			"project_id": memory.ProjectID.String(),
			"content":    decryptedContent,
			"tags":       memory.Tags,
			"created_at": memory.CreatedAt,
			"updated_at": memory.UpdatedAt,
		}
		if memory.Project != nil {
			result["project"] = map[string]interface{}{
				"id":   memory.Project.ID.String(),
				"name": memory.Project.Name,
			}
		}

		return mustTextResult(result), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: list or get", action)
	}
}

// --- unified_decision: create/list ---

type UnifiedDecisionInput struct {
	Action       string  `json:"action"`                  // create, list
	Project      string  `json:"project,omitempty"`       // For both
	Title        string  `json:"title,omitempty"`         // For create
	Description  string  `json:"description,omitempty"`   // For create
	Status       string  `json:"status,omitempty"`        // For list filter
	Area         string  `json:"area,omitempty"`          // For create/list
	Context      string  `json:"context,omitempty"`       // For create
	Consequences string  `json:"consequences,omitempty"`  // For create
	Limit        float64 `json:"limit,omitempty"`
	Cursor       string  `json:"cursor,omitempty"`
}

func handleUnifiedDecision(ctx context.Context, req *mcp.CallToolRequest, input UnifiedDecisionInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "create":
		title := strings.TrimSpace(input.Title)
		if title == "" {
			return nil, nil, errors.New("title is required for create action")
		}

		if strings.TrimSpace(input.Project) == "" {
			return nil, nil, errors.New("'project' parameter is REQUIRED for create action")
		}

		projectID, err := resolveProjectID(apiClient, input.Project)
		if err != nil {
			return nil, nil, err
		}

		decision, err := apiClient.CreateDecision(
			projectID,
			title,
			strings.TrimSpace(input.Description),
			"draft", // Agent-created decisions are always drafts
			strings.TrimSpace(input.Area),
			strings.TrimSpace(input.Context),
			strings.TrimSpace(input.Consequences),
			"agent",
		)
		if err != nil {
			return nil, nil, err
		}
		return mustTextResult(map[string]interface{}{
			"action":   "created",
			"decision": decision,
			"_message": "Decision created as draft",
		}), nil, nil

	case "list":
		var projectID string
		if project := strings.TrimSpace(input.Project); project != "" {
			var err error
			projectID, err = resolveProjectID(apiClient, project)
			if err != nil {
				return nil, nil, err
			}
		}

		decisions, err := apiClient.ListDecisions(projectID, strings.TrimSpace(input.Status), strings.TrimSpace(input.Area), 0)
		if err != nil {
			return nil, nil, err
		}

		limit := int(input.Limit)
		if limit <= 0 {
			limit = 20
		}

		paginated, nextCursor, total := paginateSlice(decisions, input.Cursor, limit)
		return mustTextResult(formatPaginatedResponse(paginated, nextCursor, total, getContextString())), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: create or list", action)
	}
}

// --- unified_skill: list/create/surface/execute/complete/stats/generate/update ---

type UnifiedSkillInput struct {
	Action      string   `json:"action"`                // list, create, surface, execute, complete, stats, generate, update
	Project     string   `json:"project,omitempty"`     // For list, create, surface, generate
	SkillID     string   `json:"skill_id,omitempty"`    // For execute, complete, stats, update
	ExecutionID string   `json:"execution_id,omitempty"` // For complete
	Context     string   `json:"context,omitempty"`     // For surface, execute
	Trigger     string   `json:"trigger,omitempty"`     // For create, update
	Description string   `json:"description,omitempty"` // For create, generate, update
	Steps       []string `json:"steps,omitempty"`       // For create, update
	Validation  string   `json:"validation,omitempty"`  // For create, update
	Tags        []string `json:"tags,omitempty"`        // For create, update
	Success     bool     `json:"success,omitempty"`     // For complete
	Notes       string   `json:"notes,omitempty"`       // For complete
	AutoSave    bool     `json:"auto_save,omitempty"`   // For generate
	Limit       int      `json:"limit,omitempty"`
}

func handleUnifiedSkill(ctx context.Context, req *mcp.CallToolRequest, input UnifiedSkillInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "list":
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		projectID := ""
		if input.Project != "" {
			resolved, err := resolveProjectID(apiClient, input.Project)
			if err == nil {
				projectID = resolved
			}
		}
		skills, err := apiClient.ListSkills(projectID, limit)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list skills: %w", err)
		}

		var result []map[string]interface{}
		for _, skill := range skills {
			item := map[string]interface{}{
				"id":         skill.ID,
				"content":    skill.Content,
				"trigger":    skill.Trigger,
				"steps":      skill.Steps,
				"validation": skill.Validation,
				"created_at": skill.CreatedAt,
			}
			if skill.Project != nil {
				item["project_name"] = skill.Project.Name
			}
			result = append(result, item)
		}

		return mustTextResult(map[string]interface{}{
			"action": "listed",
			"skills": result,
			"count":  len(result),
		}), nil, nil

	case "create":
		if err := checkSessionInit("skill"); err != nil {
			return nil, nil, err
		}

		if input.Project == "" {
			return nil, nil, errors.New("project is required for create action")
		}
		if input.Trigger == "" {
			return nil, nil, errors.New("trigger is required for create action")
		}
		if input.Description == "" {
			return nil, nil, errors.New("description is required for create action")
		}
		if len(input.Steps) == 0 {
			return nil, nil, errors.New("steps array is required for create action")
		}

		// Resolve project with org info for encryption handling
		projectID, orgID, err := resolveProjectWithOrg(apiClient, input.Project)
		if err != nil || projectID == "" {
			return nil, nil, errors.New("project is required for create action")
		}

		// Check encryption based on project scope (org vs personal)
		cfg, _ := config.LoadConfig()

		// Personal project only — encrypt with personal key (org projects skip encryption)
		if orgID == "" && cfg != nil && cfg.EncryptionEnabled && crypto.IsVaultUnlocked() {
			encryptedContent, nonce, isEncrypted, err := crypto.EncryptContent(input.Description)
			if err == nil && isEncrypted {
				skill, err := apiClient.CreateEncryptedSkill(projectID, input.Trigger, encryptedContent, nonce, input.Steps, input.Validation, input.Tags, "personal", "")
				if err != nil {
					return nil, nil, fmt.Errorf("failed to create encrypted skill: %w", err)
				}
				return mustTextResult(map[string]interface{}{
					"action":    "created",
					"skill":     skill,
					"encrypted": true,
					"_message":  "Skill created successfully (encrypted)",
				}), nil, nil
			}
		}

		// Create non-encrypted skill
		skill, err := apiClient.CreateSkill(projectID, input.Trigger, input.Description, input.Steps, input.Validation, input.Tags)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create skill: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"action":   "created",
			"skill":    skill,
			"_message": "Skill created successfully",
		}), nil, nil

	case "surface":
		contextStr := strings.TrimSpace(input.Context)
		if contextStr == "" {
			return nil, nil, errors.New("context is required for surface action")
		}

		limit := input.Limit
		if limit <= 0 {
			limit = 5
		}

		projectID := ""
		if input.Project != "" {
			pid, err := resolveProjectID(apiClient, input.Project)
			if err == nil {
				projectID = pid
			}
		}

		// Get memories and filter for skills with matching triggers
		memories, err := apiClient.ListMemories(projectID, "")
		if err != nil {
			return nil, nil, err
		}

		contextLower := strings.ToLower(contextStr)
		contextWords := strings.Fields(contextLower)

		type scoredSkill struct {
			skill interface{}
			score int
		}
		var scored []scoredSkill

		for _, m := range memories {
			if m.Type != "skill" {
				continue
			}

			decryptedContent := decryptMemoryContent(&m)
			score := 0

			triggerStr := ""
			if m.Trigger != nil {
				triggerStr = strings.ToLower(*m.Trigger)
			}

			for _, word := range contextWords {
				if len(word) < 3 {
					continue
				}
				if triggerStr != "" && strings.Contains(triggerStr, word) {
					score += 30
				}
				if strings.Contains(strings.ToLower(decryptedContent), word) {
					score += 10
				}
			}

			if score < 10 {
				continue
			}

			result := map[string]interface{}{
				"id":      m.ID.String(),
				"content": decryptedContent,
				"score":   score,
			}
			if m.Trigger != nil && *m.Trigger != "" {
				result["trigger"] = *m.Trigger
			}
			if m.Steps != nil {
				result["steps"] = m.Steps
			}
			if m.Validation != nil && *m.Validation != "" {
				result["validation"] = *m.Validation
			}
			if m.Project != nil {
				result["project"] = m.Project.Name
			}

			scored = append(scored, scoredSkill{skill: result, score: score})
		}

		// Sort by score descending
		for i := 0; i < len(scored)-1; i++ {
			for j := i + 1; j < len(scored); j++ {
				if scored[j].score > scored[i].score {
					scored[i], scored[j] = scored[j], scored[i]
				}
			}
		}

		if len(scored) > limit {
			scored = scored[:limit]
		}

		var results []interface{}
		for _, s := range scored {
			results = append(results, s.skill)
		}

		return mustTextResult(map[string]interface{}{
			"action":       "surfaced",
			"context":      contextStr,
			"skills_found": len(results),
			"skills":       results,
		}), nil, nil

	case "execute":
		if err := checkSessionInit("skill"); err != nil {
			return nil, nil, err
		}
		if input.SkillID == "" {
			return nil, nil, errors.New("skill_id is required for execute action")
		}

		session := GetCurrentSession()
		var agentName, agentModel *string
		if session != nil {
			agentName = &session.AgentName
			agentModel = &session.AgentModel
		}

		execution, err := apiClient.StartSkillExecution(input.SkillID, input.Context, agentName, agentModel)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to start execution: %w", err)
		}

		var steps []string
		if execution.Skill != nil {
			steps = execution.Skill.Steps
		}

		return mustTextResult(map[string]interface{}{
			"action":       "execution_started",
			"execution_id": execution.ID,
			"skill_id":     execution.SkillID,
			"trigger":      execution.Skill.Trigger,
			"steps":        steps,
			"validation":   execution.Skill.Validation,
			"_message":     fmt.Sprintf("Follow these %d steps, then call skill(action=complete)", len(steps)),
		}), nil, nil

	case "complete":
		if err := checkSessionInit("skill"); err != nil {
			return nil, nil, err
		}
		if input.ExecutionID == "" {
			return nil, nil, errors.New("execution_id is required for complete action")
		}

		execution, err := apiClient.CompleteSkillExecution(input.ExecutionID, input.Success, input.Notes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to complete execution: %w", err)
		}

		status := "succeeded"
		if !input.Success {
			status = "failed"
		}

		return mustTextResult(map[string]interface{}{
			"action":       "execution_completed",
			"execution_id": execution.ID,
			"success":      input.Success,
			"_message":     fmt.Sprintf("Execution %s", status),
		}), nil, nil

	case "stats":
		if err := checkSessionInit("skill"); err != nil {
			return nil, nil, err
		}
		if input.SkillID == "" {
			return nil, nil, errors.New("skill_id is required for stats action")
		}

		stats, err := apiClient.GetSkillStats(input.SkillID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get skill stats: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"action":           "stats_retrieved",
			"skill_id":         stats.SkillID,
			"total_executions": stats.TotalExecutions,
			"success_count":    stats.SuccessCount,
			"failure_count":    stats.FailureCount,
			"success_rate":     fmt.Sprintf("%.1f%%", stats.SuccessRate*100),
			"last_executed_at": stats.LastExecutedAt,
		}), nil, nil

	case "generate":
		if err := checkSessionInit("skill"); err != nil {
			return nil, nil, err
		}
		if input.Description == "" {
			return nil, nil, errors.New("description is required for generate action")
		}

		var projectID *string
		if input.Project != "" {
			resolved, err := resolveProjectID(apiClient, input.Project)
			if err == nil && resolved != "" {
				projectID = &resolved
			}
		}

		response, err := apiClient.GenerateSkill(input.Description, projectID, input.AutoSave)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate skill: %w", err)
		}

		result := map[string]interface{}{
			"action":     "generated",
			"skill":      response.Skill,
			"ai_model":   response.AIModel,
			"latency_ms": response.LatencyMs,
		}

		if response.SavedID != nil {
			result["saved_id"] = response.SavedID
			result["_message"] = fmt.Sprintf("Skill generated and saved (ID: %s)", response.SavedID)
		} else {
			result["_message"] = "Skill generated (not saved - use auto_save=true or create manually)"
		}

		return mustTextResult(result), nil, nil

	case "update":
		if err := checkSessionInit("skill"); err != nil {
			return nil, nil, err
		}
		if input.SkillID == "" {
			return nil, nil, errors.New("skill_id is required for update action")
		}

		updates := make(map[string]interface{})
		if input.Trigger != "" {
			updates["trigger"] = input.Trigger
		}
		if input.Description != "" {
			updates["content"] = input.Description
		}
		if len(input.Steps) > 0 {
			updates["steps"] = input.Steps
		}
		if input.Validation != "" {
			updates["validation"] = input.Validation
		}
		if len(input.Tags) > 0 {
			hasSkillTag := false
			for _, t := range input.Tags {
				if t == "skill" {
					hasSkillTag = true
					break
				}
			}
			if !hasSkillTag {
				input.Tags = append(input.Tags, "skill")
			}
			updates["tags"] = input.Tags
		}

		if len(updates) == 0 {
			return nil, nil, errors.New("at least one field to update is required")
		}

		updated, err := apiClient.UpdateMemory(input.SkillID, updates)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to update skill: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"action":   "updated",
			"skill":    updated,
			"_message": "Skill updated successfully",
		}), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: list, create, surface, execute, complete, stats, generate, or update", action)
	}
}

// --- unified_entity: list/get/create/create_relationship/graph/memories/entity_memories/stats/traverse/extract ---

type UnifiedEntityInput struct {
	Action             string   `json:"action"`                        // list, get, create, create_rel, graph, memories, entity_memories, stats, traverse, extract
	EntityID           string   `json:"entity_id,omitempty"`           // For get, graph, entity_memories, traverse
	MemoryID           string   `json:"memory_id,omitempty"`           // For memories
	Name               string   `json:"name,omitempty"`                // For create
	Type               string   `json:"type,omitempty"`                // For create, list filter
	Description        string   `json:"description,omitempty"`         // For create, create_rel
	Aliases            []string `json:"aliases,omitempty"`             // For create
	Project            string   `json:"project,omitempty"`             // For create, list
	Confidence         float64  `json:"confidence,omitempty"`          // For create
	SourceEntityID     string   `json:"source_entity_id,omitempty"`    // For create_rel
	TargetEntityID     string   `json:"target_entity_id,omitempty"`    // For create_rel, traverse
	RelationshipType   string   `json:"relationship_type,omitempty"`   // For create_rel
	Label              string   `json:"label,omitempty"`               // For create_rel
	Strength           float64  `json:"strength,omitempty"`            // For create_rel
	Hops               float64  `json:"hops,omitempty"`                // For graph, entity_memories, traverse
	MaxDepth           int      `json:"max_depth,omitempty"`           // For traverse
	RelationshipTypes  []string `json:"relationship_types,omitempty"`  // For traverse
	EntityTypes        []string `json:"entity_types,omitempty"`        // For traverse
	Mode               string   `json:"mode,omitempty"`                // For traverse (paths, cluster, neighbors)
	Content            string   `json:"content,omitempty"`             // For extract
	Query              string   `json:"query,omitempty"`               // For list
	Limit              float64  `json:"limit,omitempty"`
	Offset             float64  `json:"offset,omitempty"`
}

func handleUnifiedEntity(ctx context.Context, req *mcp.CallToolRequest, input UnifiedEntityInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "list":
		entityType := strings.TrimSpace(input.Type)
		query := strings.TrimSpace(input.Query)

		projectID := ""
		if project := strings.TrimSpace(input.Project); project != "" {
			pid, err := resolveProjectID(apiClient, project)
			if err == nil {
				projectID = pid
			}
		}

		limit := int(input.Limit)
		if limit <= 0 {
			limit = 50
		}
		offset := int(input.Offset)

		response, err := apiClient.ListEntities(entityType, projectID, query, limit, offset)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list entities: %w", err)
		}

		entities := make([]map[string]interface{}, 0, len(response.Entities))
		for _, e := range response.Entities {
			entity := map[string]interface{}{
				"id":            e.ID.String(),
				"name":          e.Name,
				"type":          e.Type,
				"mention_count": e.MentionCount,
				"confidence":    e.Confidence,
			}
			if e.Description != nil {
				entity["description"] = *e.Description
			}
			if len(e.Aliases) > 0 {
				entity["aliases"] = e.Aliases
			}
			entities = append(entities, entity)
		}

		return mustTextResult(map[string]interface{}{
			"entities": entities,
			"total":    response.Total,
			"limit":    response.Limit,
			"offset":   response.Offset,
		}), nil, nil

	case "get":
		entityID := strings.TrimSpace(input.EntityID)
		if entityID == "" {
			return nil, nil, errors.New("entity_id is required for get action")
		}

		entity, err := apiClient.GetEntity(entityID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get entity: %w", err)
		}

		result := map[string]interface{}{
			"id":              entity.ID.String(),
			"name":            entity.Name,
			"normalized_name": entity.NormalizedName,
			"type":            entity.Type,
			"mention_count":   entity.MentionCount,
			"confidence":      entity.Confidence,
			"created_at":      entity.CreatedAt,
			"updated_at":      entity.UpdatedAt,
		}
		if entity.Description != nil {
			result["description"] = *entity.Description
		}
		if len(entity.Aliases) > 0 {
			result["aliases"] = entity.Aliases
		}

		return mustTextResult(result), nil, nil

	case "create":
		if err := checkSessionInit("entity"); err != nil {
			return nil, nil, err
		}

		name := strings.TrimSpace(input.Name)
		if name == "" {
			return nil, nil, errors.New("name is required for create action")
		}

		entityType := strings.TrimSpace(input.Type)
		if entityType == "" {
			return nil, nil, errors.New("type is required for create action")
		}

		validTypes := map[string]bool{
			"person": true, "tool": true, "concept": true, "project": true,
			"organization": true, "location": true, "event": true,
			"document": true, "api": true, "other": true,
		}
		if !validTypes[entityType] {
			return nil, nil, fmt.Errorf("invalid entity type: %s", entityType)
		}

		createReq := &models.CreateEntityRequest{
			Name:    name,
			Type:    models.GraphEntityType(entityType),
			Aliases: input.Aliases,
		}

		if desc := strings.TrimSpace(input.Description); desc != "" {
			createReq.Description = &desc
		}

		if project := strings.TrimSpace(input.Project); project != "" {
			projectID, err := resolveProjectID(apiClient, project)
			if err == nil {
				createReq.ProjectID = &projectID
			}
		}

		if input.Confidence > 0 && input.Confidence <= 1 {
			createReq.Confidence = &input.Confidence
		}

		entity, err := apiClient.CreateEntity(createReq)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create entity: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"action":   "created",
			"entity":   entity,
			"_message": fmt.Sprintf("Entity '%s' created", entity.Name),
		}), nil, nil

	case "create_rel", "create_relationship":
		if err := checkSessionInit("entity"); err != nil {
			return nil, nil, err
		}

		sourceID := strings.TrimSpace(input.SourceEntityID)
		if sourceID == "" {
			return nil, nil, errors.New("source_entity_id is required for create_rel action")
		}

		targetID := strings.TrimSpace(input.TargetEntityID)
		if targetID == "" {
			return nil, nil, errors.New("target_entity_id is required for create_rel action")
		}

		relType := strings.TrimSpace(input.RelationshipType)
		if relType == "" {
			return nil, nil, errors.New("relationship_type is required for create_rel action")
		}

		validTypes := map[string]bool{
			"uses": true, "works_on": true, "related_to": true, "depends_on": true,
			"part_of": true, "created_by": true, "belongs_to": true, "connects_to": true,
			"replaces": true, "similar_to": true, "contradicts": true, "references": true,
			"implements": true, "extends": true,
		}
		if !validTypes[relType] {
			return nil, nil, fmt.Errorf("invalid relationship type: %s", relType)
		}

		createReq := &models.CreateRelationshipRequest{
			SourceEntityID:   sourceID,
			TargetEntityID:   targetID,
			RelationshipType: models.RelationshipType(relType),
		}

		if label := strings.TrimSpace(input.Label); label != "" {
			createReq.Label = &label
		}

		if desc := strings.TrimSpace(input.Description); desc != "" {
			createReq.Description = &desc
		}

		if input.Strength > 0 && input.Strength <= 1 {
			createReq.Strength = &input.Strength
		}

		relationship, err := apiClient.CreateRelationship(createReq)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create relationship: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"action":       "created",
			"relationship": relationship,
			"_message":     fmt.Sprintf("Relationship '%s' created", relType),
		}), nil, nil

	case "graph":
		entityID := strings.TrimSpace(input.EntityID)
		if entityID == "" {
			return nil, nil, errors.New("entity_id is required for graph action")
		}

		hops := int(input.Hops)
		if hops < 1 {
			hops = 1
		}
		if hops > 3 {
			hops = 3
		}

		response, err := apiClient.GetEntityGraph(entityID, hops)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get entity graph: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"root_entity": response.RootEntity,
			"nodes":       response.Nodes,
			"edges":       response.Edges,
			"hops":        response.Hops,
			"node_count":  response.NodeCount,
			"edge_count":  response.EdgeCount,
		}), nil, nil

	case "memories":
		memoryID := strings.TrimSpace(input.MemoryID)
		if memoryID == "" {
			return nil, nil, errors.New("memory_id is required for memories action")
		}

		response, err := apiClient.GetMemoryEntities(memoryID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get memory entities: %w", err)
		}

		entities := make([]map[string]interface{}, 0, len(response.Entities))
		for _, e := range response.Entities {
			entity := map[string]interface{}{
				"id":            e.ID.String(),
				"name":          e.Name,
				"type":          e.Type,
				"confidence":    e.Confidence,
				"mention_count": e.MentionCount,
			}
			if e.Description != nil {
				entity["description"] = *e.Description
			}
			entities = append(entities, entity)
		}

		return mustTextResult(map[string]interface{}{
			"entities":  entities,
			"memory_id": response.MemoryID,
			"total":     response.Total,
		}), nil, nil

	case "entity_memories":
		entityID := strings.TrimSpace(input.EntityID)
		if entityID == "" {
			return nil, nil, errors.New("entity_id is required for entity_memories action")
		}

		hops := int(input.Hops)
		if hops < 0 {
			hops = 0
		}
		if hops > 3 {
			hops = 3
		}

		limit := int(input.Limit)
		if limit <= 0 {
			limit = 50
		}

		response, err := apiClient.GetEntityMemories(entityID, hops, limit)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get entity memories: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"memory_ids": response.MemoryIDs,
			"entity_id":  response.EntityID,
			"hops":       response.Hops,
			"total":      response.Total,
		}), nil, nil

	case "stats":
		response, err := apiClient.GetEntityStats()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get entity stats: %w", err)
		}

		return mustTextResult(map[string]interface{}{
			"total_entities":      response.TotalEntities,
			"total_relationships": response.TotalRelationships,
			"entities_by_type":    response.EntitiesByType,
		}), nil, nil

	case "traverse":
		if err := checkSessionInit("entity"); err != nil {
			return nil, nil, err
		}

		startID := strings.TrimSpace(input.EntityID)
		if startID == "" {
			return nil, nil, errors.New("entity_id (start) is required for traverse action")
		}

		maxDepth := input.MaxDepth
		if maxDepth <= 0 || maxDepth > 5 {
			maxDepth = 3
		}

		mode := strings.TrimSpace(input.Mode)
		if mode == "" {
			mode = "cluster"
		}

		if mode == "neighbors" {
			maxDepth = 1
		}

		graph, err := apiClient.GetEntityGraph(startID, maxDepth)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to traverse graph: %w", err)
		}

		result := map[string]interface{}{
			"mode":         mode,
			"start_entity": startID,
			"max_depth":    maxDepth,
			"nodes":        graph.Nodes,
			"edges":        graph.Edges,
			"node_count":   graph.NodeCount,
			"edge_count":   graph.EdgeCount,
		}

		if mode == "paths" && input.TargetEntityID != "" {
			targetFound := false
			for _, n := range graph.Nodes {
				if nodeID, ok := n["id"].(string); ok && nodeID == input.TargetEntityID {
					targetFound = true
					break
				}
			}
			result["target_entity"] = input.TargetEntityID
			result["path_exists"] = targetFound
		}

		return mustTextResult(result), nil, nil

	case "extract":
		if err := checkSessionInit("entity"); err != nil {
			return nil, nil, err
		}

		content := strings.TrimSpace(input.Content)
		if content == "" {
			return nil, nil, errors.New("content is required for extract action")
		}
		if len(content) < 10 {
			return nil, nil, errors.New("content must be at least 10 characters")
		}

		result, err := apiClient.PreviewExtraction(content)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extract entities: %w", err)
		}

		entityCount := 0
		relCount := 0
		if entities, ok := result["entities"].([]interface{}); ok {
			entityCount = len(entities)
		}
		if rels, ok := result["relationships"].([]interface{}); ok {
			relCount = len(rels)
		}

		result["_message"] = fmt.Sprintf("Extracted %d entities and %d relationships (preview only)", entityCount, relCount)

		return mustTextResult(result), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: list, get, create, create_rel, graph, memories, entity_memories, stats, traverse, or extract", action)
	}
}

// --- unified_admin: consolidate/cleanup/orgs/switch_org/export/import/plan/analyze ---

type UnifiedAdminInput struct {
	Action           string                 `json:"action"`                     // consolidate, cleanup, orgs, switch_org, export, import, plan, analyze
	Project          string                 `json:"project,omitempty"`          // For consolidate, cleanup, plan, analyze
	StaleDays        int                    `json:"stale_days,omitempty"`       // For consolidate
	PromoteThreshold float64                `json:"promote_threshold,omitempty"` // For consolidate
	ArchiveThreshold float64                `json:"archive_threshold,omitempty"` // For consolidate
	DryRun           bool                   `json:"dry_run,omitempty"`          // For cleanup
	BatchSize        int                    `json:"batch_size,omitempty"`       // For cleanup
	OrgID            string                 `json:"orgId,omitempty"`            // For switch_org
	PackID           string                 `json:"pack_id,omitempty"`          // For export
	Bundle           map[string]interface{} `json:"bundle,omitempty"`           // For import
	ConflictMode     string                 `json:"conflict_mode,omitempty"`    // For import
	// Plan fields
	PlanID       string  `json:"plan_id,omitempty"`
	Requirements string  `json:"requirements,omitempty"`
	Title        string  `json:"title,omitempty"`
	Type         string  `json:"type,omitempty"`
	Consensus    float64 `json:"consensus,omitempty"`
	BudgetUSD    float64 `json:"budget_usd,omitempty"`
	Status       string  `json:"status,omitempty"`
	Limit        float64 `json:"limit,omitempty"`
	ApplyTasks   bool    `json:"apply_tasks,omitempty"`
	ApplyADRs    bool    `json:"apply_adrs,omitempty"`
	TaskStatus   string  `json:"task_status,omitempty"`
	ADRStatus    string  `json:"adr_status,omitempty"`
	// Analyze fields
	Description string `json:"description,omitempty"`
	TechStack   string `json:"tech_stack,omitempty"`
	Conventions string `json:"conventions,omitempty"`
	Files       []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"files,omitempty"`
}

func handleUnifiedAdmin(ctx context.Context, req *mcp.CallToolRequest, input UnifiedAdminInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		return nil, nil, errors.New("action is required (consolidate|cleanup|orgs|switch_org|export|import|plan_create|plan_status|plan_list|plan_apply|plan_cancel|analyze)")
	}

	switch action {
	case "consolidate":
		project := strings.TrimSpace(input.Project)
		if project == "" {
			return nil, nil, errors.New("project is required for consolidate action")
		}

		projectID, err := resolveProjectID(apiClient, project)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve project: %w", err)
		}

		payload := map[string]interface{}{"project_id": projectID}
		if input.StaleDays > 0 {
			payload["stale_days"] = input.StaleDays
		}
		if input.PromoteThreshold > 0 {
			payload["promote_threshold"] = input.PromoteThreshold
		}
		if input.ArchiveThreshold > 0 {
			payload["archive_threshold"] = input.ArchiveThreshold
		}

		if session := GetCurrentSession(); session != nil && session.ActiveOrgID != nil {
			payload["org_id"] = session.ActiveOrgID.String()
		}

		result, err := apiClient.EnqueueJob("memory_consolidate", payload)
		if err != nil {
			return nil, nil, fmt.Errorf("enqueue consolidation job: %w", err)
		}

		return mustTextResult(result), nil, nil

	case "cleanup":
		payload := map[string]interface{}{}

		if project := strings.TrimSpace(input.Project); project != "" {
			projectID, err := resolveProjectID(apiClient, project)
			if err != nil {
				return nil, nil, fmt.Errorf("resolve project: %w", err)
			}
			payload["project_id"] = projectID
		}

		if input.DryRun {
			payload["dry_run"] = true
		}
		if input.BatchSize > 0 {
			payload["batch_size"] = input.BatchSize
		}

		if session := GetCurrentSession(); session != nil && session.ActiveOrgID != nil {
			payload["org_id"] = session.ActiveOrgID.String()
		}

		result, err := apiClient.EnqueueJob("memory_cleanup", payload)
		if err != nil {
			return nil, nil, fmt.Errorf("enqueue cleanup job: %w", err)
		}

		return mustTextResult(result), nil, nil

	case "orgs":
		orgs, err := apiClient.ListOrganizations()
		if err != nil {
			return nil, nil, err
		}
		return mustTextResult(formatMCPResponse(orgs, getContextString())), nil, nil

	case "switch_org":
		orgID := strings.TrimSpace(input.OrgID)
		if orgID == "" {
			return nil, nil, errors.New("orgId is required for switch_org action")
		}

		org, err := apiClient.SwitchOrganization(orgID)
		if err != nil {
			return nil, nil, err
		}

		return mustTextResult(org), nil, nil

	case "export":
		if err := checkSessionInit("admin"); err != nil {
			return nil, nil, err
		}

		packID := strings.TrimSpace(input.PackID)
		if packID == "" {
			return nil, nil, errors.New("pack_id is required for export action")
		}

		respBody, err := apiClient.Request("GET", "/context-packs/"+packID+"/export", nil)
		if err != nil {
			return nil, nil, fmt.Errorf("export failed: %w", err)
		}

		var bundle map[string]interface{}
		if err := json.Unmarshal(respBody, &bundle); err != nil {
			return nil, nil, fmt.Errorf("failed to parse export bundle: %w", err)
		}

		result := formatMCPResponse(bundle, getContextString())
		result["_message"] = "Context pack exported successfully"

		return mustTextResult(result), nil, nil

	case "import":
		if err := checkSessionInit("admin"); err != nil {
			return nil, nil, err
		}

		if input.Bundle == nil {
			return nil, nil, errors.New("bundle is required for import action")
		}
		project := strings.TrimSpace(input.Project)
		if project == "" {
			return nil, nil, errors.New("project is required for import action")
		}

		projectID, err := resolveProjectID(apiClient, project)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve project: %w", err)
		}

		importReq := map[string]interface{}{
			"bundle":     input.Bundle,
			"project_id": projectID,
		}
		if input.ConflictMode != "" {
			importReq["conflict_mode"] = input.ConflictMode
		}

		respBody, err := apiClient.Request("POST", "/context-packs/import", importReq)
		if err != nil {
			return nil, nil, fmt.Errorf("import failed: %w", err)
		}

		var importResult map[string]interface{}
		if err := json.Unmarshal(respBody, &importResult); err != nil {
			return nil, nil, fmt.Errorf("failed to parse import result: %w", err)
		}

		result := formatMCPResponse(importResult, getContextString())
		result["_message"] = "Context pack imported successfully"

		return mustTextResult(result), nil, nil

	case "plan_create":
		requirements := strings.TrimSpace(input.Requirements)
		if requirements == "" {
			return nil, nil, errors.New("requirements is required for plan_create action")
		}
		apiReq := api.CreatePlanRequest{Requirements: requirements}
		if t := strings.TrimSpace(input.Title); t != "" {
			apiReq.Title = t
		}
		if t := strings.TrimSpace(input.Type); t != "" {
			apiReq.Type = t
		}
		if p := strings.TrimSpace(input.Project); p != "" {
			apiReq.ProjectID = p
		}
		if input.Consensus > 0 || input.BudgetUSD > 0 {
			cfg := &api.PlanConfiguration{}
			if input.Consensus > 0 {
				cfg.ConsensusThreshold = input.Consensus
			}
			if input.BudgetUSD > 0 {
				cfg.BudgetUSD = input.BudgetUSD
			}
			apiReq.Configuration = cfg
		}
		plan, err := apiClient.CreatePlan(apiReq)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create plan: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"action":   "plan_created",
			"plan":     plan,
			"_message": "Plan created",
		}), nil, nil

	case "plan_status":
		planID := strings.TrimSpace(input.PlanID)
		if planID == "" {
			return nil, nil, errors.New("plan_id is required for plan_status action")
		}
		plan, err := apiClient.GetPlan(planID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get plan: %w", err)
		}
		return mustTextResult(plan), nil, nil

	case "plan_list":
		filter := api.ListPlansFilter{}
		if s := strings.TrimSpace(input.Status); s != "" {
			filter.Status = s
		}
		if t := strings.TrimSpace(input.Type); t != "" {
			filter.Type = t
		}
		if p := strings.TrimSpace(input.Project); p != "" {
			filter.ProjectID = p
		}
		if input.Limit > 0 {
			filter.Limit = int(input.Limit)
		}
		plans, err := apiClient.ListPlans(filter)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list plans: %w", err)
		}
		return mustTextResult(formatMCPResponse(plans, getContextString())), nil, nil

	case "plan_apply":
		planID := strings.TrimSpace(input.PlanID)
		if planID == "" {
			return nil, nil, errors.New("plan_id is required for plan_apply action")
		}
		applyReq := api.ApplyPlanRequest{
			ApplyTasks: input.ApplyTasks,
			ApplyADRs:  input.ApplyADRs,
		}
		if s := strings.TrimSpace(input.TaskStatus); s != "" {
			applyReq.TaskStatus = s
		}
		if s := strings.TrimSpace(input.ADRStatus); s != "" {
			applyReq.ADRStatus = s
		}
		result, err := apiClient.ApplyPlan(planID, applyReq)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to apply plan: %w", err)
		}
		return mustTextResult(result), nil, nil

	case "plan_cancel":
		planID := strings.TrimSpace(input.PlanID)
		if planID == "" {
			return nil, nil, errors.New("plan_id is required for plan_cancel action")
		}
		if err := apiClient.CancelPlan(planID); err != nil {
			return nil, nil, fmt.Errorf("failed to cancel plan: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": "Plan cancelled",
			"plan_id": planID,
		}), nil, nil

	case "analyze":
		if err := checkSessionInit("admin"); err != nil {
			return nil, nil, err
		}

		projectName := strings.TrimSpace(input.Project)
		if projectName == "" {
			return nil, nil, errors.New("project is required for analyze action")
		}

		projectID, err := resolveProjectID(apiClient, projectName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve project: %w", err)
		}

		var analysisReq api.AnalyzeProjectRequest

		if len(input.Files) > 0 {
			analysisReq.Files = make([]api.FileInput, len(input.Files))
			for i, f := range input.Files {
				analysisReq.Files[i] = api.FileInput{
					Path:    f.Path,
					Content: f.Content,
				}
			}
		}

		if input.Description != "" || input.TechStack != "" || input.Conventions != "" {
			manualInput := &api.ManualProjectInput{
				Description: input.Description,
				Conventions: input.Conventions,
			}
			if input.TechStack != "" {
				parts := strings.Split(input.TechStack, ",")
				for _, p := range parts {
					if trimmed := strings.TrimSpace(p); trimmed != "" {
						manualInput.TechStack = append(manualInput.TechStack, trimmed)
					}
				}
			}
			analysisReq.ManualInput = manualInput
		}

		if len(analysisReq.Files) == 0 && analysisReq.ManualInput == nil {
			return nil, nil, errors.New("either files or manual input is required for analyze action")
		}

		result, err := apiClient.AnalyzeProject(projectID, analysisReq)
		if err != nil {
			return nil, nil, fmt.Errorf("analysis failed: %w", err)
		}

		response := map[string]interface{}{
			"project_id":     result.ProjectID,
			"analyzed_at":    result.AnalyzedAt,
			"source":         result.Source,
			"confidence":     result.Confidence,
			"ai_model":       result.AIModel,
			"files_analyzed": result.FilesAnalyzed,
		}

		if len(result.TechStack) > 0 {
			response["tech_stack"] = result.TechStack
		}
		if len(result.SuggestedMemories) > 0 {
			response["suggested_memories"] = result.SuggestedMemories
		}
		if len(result.SuggestedDecisions) > 0 {
			response["suggested_decisions"] = result.SuggestedDecisions
		}
		if len(result.SuggestedTasks) > 0 {
			response["suggested_tasks"] = result.SuggestedTasks
		}

		totalSuggestions := len(result.SuggestedMemories) + len(result.SuggestedDecisions) + len(result.SuggestedTasks)
		response["_message"] = fmt.Sprintf("Analysis complete! Found %d suggestions", totalSuggestions)

		return mustTextResult(response), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Valid actions: consolidate, cleanup, orgs, switch_org, export, import, plan_create, plan_status, plan_list, plan_apply, plan_cancel, analyze", action)
	}
}

// Helper function for sorting tasks by priority
func sortTasksByPriority(tasks []models.Task) {
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			pi := priorityRank(tasks[i].Priority)
			pj := priorityRank(tasks[j].Priority)
			if pj > pi || (pi == pj && tasks[j].CreatedAt.Before(tasks[i].CreatedAt)) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}
