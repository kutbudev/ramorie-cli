package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ============================================================================
// MERGED TOOL HANDLERS - Consolidated from individual tools
// ============================================================================

// --- manage_focus: get/set/clear focus ---

type ManageFocusInput struct {
	PackID string `json:"pack_id,omitempty"` // Set focus to this pack
	Clear  bool   `json:"clear,omitempty"`   // Clear current focus
}

func handleManageFocus(ctx context.Context, req *mcp.CallToolRequest, input ManageFocusInput) (*mcp.CallToolResult, interface{}, error) {
	packID := strings.TrimSpace(input.PackID)

	if input.Clear {
		// Clear focus
		if err := apiClient.ClearFocus(); err != nil {
			return nil, nil, err
		}
		return mustTextResult(map[string]interface{}{"ok": true, "message": "Focus cleared"}), nil, nil
	}

	if packID != "" {
		// Set focus
		focus, err := apiClient.SetFocus(packID)
		if err != nil {
			return nil, nil, err
		}
		result := map[string]interface{}{"ok": true, "message": "Focus updated"}
		if focus.ActivePack != nil {
			result["active_pack"] = map[string]interface{}{
				"id":             focus.ActivePack.ID,
				"name":           focus.ActivePack.Name,
				"contexts_count": focus.ActivePack.ContextsCount,
				"memories_count": focus.ActivePack.MemoriesCount,
				"tasks_count":    focus.ActivePack.TasksCount,
			}
		}
		return mustTextResult(result), nil, nil
	}

	// Get focus (default when no params)
	focus, err := apiClient.GetFocus()
	if err != nil {
		return nil, nil, err
	}
	if focus.ActivePack == nil {
		return mustTextResult(map[string]interface{}{
			"active_context_pack_id": nil,
			"active_pack":            nil,
			"message":                "No active focus set. Use manage_focus with pack_id to activate a context pack.",
		}), nil, nil
	}
	return mustTextResult(map[string]interface{}{
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
	}), nil, nil
}

// --- manage_task: start/complete/progress ---

type ManageTaskInput struct {
	TaskID   string  `json:"taskId"`
	Action   string  `json:"action"`             // start, complete, stop, progress
	Progress float64 `json:"progress,omitempty"` // 0-100, for action=progress
}

func handleManageTask(ctx context.Context, req *mcp.CallToolRequest, input ManageTaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		return nil, nil, errors.New("action is required (start|complete|stop|progress)")
	}

	switch action {
	case "start":
		if err := apiClient.StartTask(taskID); err != nil {
			return nil, nil, err
		}
		return mustTextResult(map[string]interface{}{"ok": true, "message": "Task started. Memories will now auto-link to this task."}), nil, nil

	case "complete":
		if err := apiClient.CompleteTask(taskID); err != nil {
			return nil, nil, err
		}
		return mustTextResult(map[string]interface{}{"ok": true, "message": "Task completed."}), nil, nil

	case "stop":
		if err := apiClient.StopTask(taskID); err != nil {
			return nil, nil, err
		}
		return mustTextResult(map[string]interface{}{"ok": true, "message": "Task stopped. Active task cleared."}), nil, nil

	case "progress":
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
			"message": fmt.Sprintf("Task progress updated to %d%%", progress),
			"task":    result,
		}), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: start, complete, stop, or progress", action)
	}
}

// --- manage_context_pack: create/update/add_memory/add_task ---

type ManageContextPackInput struct {
	Action      string   `json:"action"`                // create, update, add_memory, add_task
	PackID      string   `json:"packId,omitempty"`      // Required for update/add_memory/add_task
	Name        string   `json:"name,omitempty"`        // For create/update
	Type        string   `json:"type,omitempty"`        // For create (project, integration, decision, custom)
	Description string   `json:"description,omitempty"` // For create/update
	Status      string   `json:"status,omitempty"`      // For update
	Tags        []string `json:"tags,omitempty"`        // For create
	MemoryID    string   `json:"memoryId,omitempty"`    // For add_memory
	TaskID      string   `json:"taskId,omitempty"`      // For add_task
}

func handleManageContextPack(ctx context.Context, req *mcp.CallToolRequest, input ManageContextPackInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		return nil, nil, errors.New("action is required (create|update|add_memory|add_task)")
	}

	switch action {
	case "create":
		name := strings.TrimSpace(input.Name)
		packType := strings.TrimSpace(input.Type)
		if name == "" {
			return nil, nil, errors.New("name is required for create action")
		}
		if packType == "" {
			return nil, nil, errors.New("type is required for create action (project, integration, decision, custom)")
		}
		pack, err := apiClient.CreateContextPack(name, packType, strings.TrimSpace(input.Description), "draft", input.Tags)
		if err != nil {
			return nil, nil, err
		}
		return mustTextResult(pack), nil, nil

	case "update":
		packID := strings.TrimSpace(input.PackID)
		if packID == "" {
			return nil, nil, errors.New("packId is required for update action")
		}
		updates := make(map[string]interface{})
		if name := strings.TrimSpace(input.Name); name != "" {
			updates["name"] = name
		}
		if desc := strings.TrimSpace(input.Description); desc != "" {
			updates["description"] = desc
		}
		if status := strings.TrimSpace(input.Status); status != "" {
			updates["status"] = status
		}
		if len(updates) == 0 {
			return nil, nil, errors.New("at least one field to update is required (name, description, or status)")
		}
		pack, err := apiClient.UpdateContextPack(packID, updates)
		if err != nil {
			return nil, nil, err
		}
		return mustTextResult(pack), nil, nil

	case "add_memory":
		packID := strings.TrimSpace(input.PackID)
		memoryID := strings.TrimSpace(input.MemoryID)
		if packID == "" || memoryID == "" {
			return nil, nil, errors.New("packId and memoryId are required for add_memory action")
		}
		pack, err := apiClient.GetContextPack(packID)
		if err != nil {
			return nil, nil, err
		}
		memoryIDs := []string{memoryID}
		if pack.MemoryIDs != nil {
			if ids, ok := pack.MemoryIDs.([]interface{}); ok {
				for _, id := range ids {
					if idStr, ok := id.(string); ok && idStr != memoryID {
						memoryIDs = append(memoryIDs, idStr)
					}
				}
			}
		}
		result, err := apiClient.UpdateContextPack(packID, map[string]interface{}{"memory_ids": memoryIDs})
		if err != nil {
			return nil, nil, err
		}
		return mustTextResult(result), nil, nil

	case "add_task":
		packID := strings.TrimSpace(input.PackID)
		taskID := strings.TrimSpace(input.TaskID)
		if packID == "" || taskID == "" {
			return nil, nil, errors.New("packId and taskId are required for add_task action")
		}
		pack, err := apiClient.GetContextPack(packID)
		if err != nil {
			return nil, nil, err
		}
		taskIDs := []string{taskID}
		if pack.TaskIDs != nil {
			if ids, ok := pack.TaskIDs.([]interface{}); ok {
				for _, id := range ids {
					if idStr, ok := id.(string); ok && idStr != taskID {
						taskIDs = append(taskIDs, idStr)
					}
				}
			}
		}
		result, err := apiClient.UpdateContextPack(packID, map[string]interface{}{"task_ids": taskIDs})
		if err != nil {
			return nil, nil, err
		}
		return mustTextResult(result), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: create, update, add_memory, or add_task", action)
	}
}

// --- manage_subtasks: create/list/complete/update ---

type ManageSubtasksInput struct {
	Action      string `json:"action"`                // create, list, complete, update
	TaskID      string `json:"task_id"`               // Parent task ID (required for create/list)
	SubtaskID   string `json:"subtask_id,omitempty"`  // Required for complete/update
	Description string `json:"description,omitempty"` // For create/update
	Status      string `json:"status,omitempty"`      // For update (TODO, IN_PROGRESS, COMPLETED)
	Priority    string `json:"priority,omitempty"`    // For update (L, M, H)
}

func handleManageSubtasks(ctx context.Context, req *mcp.CallToolRequest, input ManageSubtasksInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		return nil, nil, errors.New("action is required (create|list|complete|update)")
	}

	switch action {
	case "create":
		taskID := strings.TrimSpace(input.TaskID)
		desc := strings.TrimSpace(input.Description)
		if taskID == "" {
			return nil, nil, errors.New("task_id is required")
		}
		if desc == "" {
			return nil, nil, errors.New("description is required")
		}
		if _, err := uuid.Parse(taskID); err != nil {
			return nil, nil, errors.New("invalid task_id format - must be a valid UUID")
		}
		subtask, err := apiClient.CreateSubtask(taskID, desc)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create subtask: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": "Subtask created",
			"subtask": subtask,
		}), nil, nil

	case "list":
		taskID := strings.TrimSpace(input.TaskID)
		if taskID == "" {
			return nil, nil, errors.New("task_id is required")
		}
		if _, err := uuid.Parse(taskID); err != nil {
			return nil, nil, errors.New("invalid task_id format - must be a valid UUID")
		}
		subtasks, err := apiClient.ListSubtasks(taskID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get subtasks: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"task_id":  taskID,
			"subtasks": subtasks,
			"count":    len(subtasks),
		}), nil, nil

	case "complete":
		subtaskID := strings.TrimSpace(input.SubtaskID)
		if subtaskID == "" {
			return nil, nil, errors.New("subtask_id is required")
		}
		if _, err := uuid.Parse(subtaskID); err != nil {
			return nil, nil, errors.New("invalid subtask_id format - must be a valid UUID")
		}
		subtask, err := apiClient.CompleteSubtask(subtaskID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to complete subtask: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": "Subtask completed",
			"subtask": subtask,
		}), nil, nil

	case "update":
		subtaskID := strings.TrimSpace(input.SubtaskID)
		if subtaskID == "" {
			return nil, nil, errors.New("subtask_id is required")
		}
		if _, err := uuid.Parse(subtaskID); err != nil {
			return nil, nil, errors.New("invalid subtask_id format - must be a valid UUID")
		}
		req := api.UpdateSubtaskRequest{}
		if desc := strings.TrimSpace(input.Description); desc != "" {
			req.Description = &desc
		}
		if status := strings.TrimSpace(strings.ToUpper(input.Status)); status != "" {
			validStatuses := map[string]bool{"TODO": true, "IN_PROGRESS": true, "COMPLETED": true}
			if !validStatuses[status] {
				return nil, nil, errors.New("invalid status - must be TODO, IN_PROGRESS, or COMPLETED")
			}
			req.Status = &status
		}
		if priority := strings.TrimSpace(strings.ToUpper(input.Priority)); priority != "" {
			validPriorities := map[string]bool{"L": true, "M": true, "H": true}
			if !validPriorities[priority] {
				return nil, nil, errors.New("invalid priority - must be L (Low), M (Medium), or H (High)")
			}
			req.Priority = &priority
		}
		subtask, err := apiClient.UpdateSubtask(subtaskID, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to update subtask: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": "Subtask updated",
			"subtask": subtask,
		}), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: create, list, complete, or update", action)
	}
}

// --- manage_dependencies: add/list/remove ---

type ManageDependenciesInput struct {
	Action      string `json:"action"`                  // add, list, remove
	TaskID      string `json:"task_id"`                 // Required for all actions
	DependsOnID string `json:"depends_on_id,omitempty"` // Required for add/remove
	Direction   string `json:"direction,omitempty"`     // For list: "deps" (default) or "dependents"
}

func handleManageDependencies(ctx context.Context, req *mcp.CallToolRequest, input ManageDependenciesInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	taskID := strings.TrimSpace(input.TaskID)

	if action == "" {
		return nil, nil, errors.New("action is required (add|list|remove)")
	}
	if taskID == "" {
		return nil, nil, errors.New("task_id is required")
	}
	if _, err := uuid.Parse(taskID); err != nil {
		return nil, nil, errors.New("invalid task_id format - must be a valid UUID")
	}

	switch action {
	case "add":
		depsOnID := strings.TrimSpace(input.DependsOnID)
		if depsOnID == "" {
			return nil, nil, errors.New("depends_on_id is required for add action")
		}
		if _, err := uuid.Parse(depsOnID); err != nil {
			return nil, nil, errors.New("invalid depends_on_id format - must be a valid UUID")
		}
		if _, err := apiClient.AddTaskDependency(taskID, depsOnID); err != nil {
			return nil, nil, fmt.Errorf("failed to add dependency: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":            true,
			"message":       fmt.Sprintf("Dependency added: %s depends on %s", taskID[:8], depsOnID[:8]),
			"task_id":       taskID,
			"depends_on_id": depsOnID,
		}), nil, nil

	case "list":
		direction := strings.TrimSpace(strings.ToLower(input.Direction))
		if direction == "" || direction == "deps" {
			deps, err := apiClient.GetTaskDependencies(taskID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get dependencies: %w", err)
			}
			return mustTextResult(map[string]interface{}{
				"task_id":      taskID,
				"direction":    "dependencies",
				"dependencies": deps,
				"count":        len(deps),
			}), nil, nil
		}
		// dependents
		deps, err := apiClient.GetTaskDependents(taskID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get dependents: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"task_id":    taskID,
			"direction":  "dependents",
			"dependents": deps,
			"count":      len(deps),
		}), nil, nil

	case "remove":
		depsOnID := strings.TrimSpace(input.DependsOnID)
		if depsOnID == "" {
			return nil, nil, errors.New("depends_on_id is required for remove action")
		}
		if _, err := uuid.Parse(depsOnID); err != nil {
			return nil, nil, errors.New("invalid depends_on_id format - must be a valid UUID")
		}
		if err := apiClient.RemoveTaskDependency(taskID, depsOnID); err != nil {
			return nil, nil, fmt.Errorf("failed to remove dependency: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": fmt.Sprintf("Dependency removed between %s and %s", taskID[:8], depsOnID[:8]),
		}), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: add, list, or remove", action)
	}
}

// --- manage_plan: create/status/list/apply/cancel ---

type ManagePlanInput struct {
	Action       string  `json:"action"`                  // create, status, list, apply, cancel
	PlanID       string  `json:"plan_id,omitempty"`       // Required for status/apply/cancel
	Requirements string  `json:"requirements,omitempty"`  // Required for create
	Title        string  `json:"title,omitempty"`         // For create
	Type         string  `json:"type,omitempty"`          // For create/list
	Project      string  `json:"project,omitempty"`       // For create/list
	Consensus    float64 `json:"consensus,omitempty"`     // For create
	BudgetUSD    float64 `json:"budget_usd,omitempty"`    // For create
	Status       string  `json:"status,omitempty"`        // For list filter
	Limit        float64 `json:"limit,omitempty"`         // For list
	ApplyTasks   bool    `json:"apply_tasks,omitempty"`   // For apply
	ApplyADRs    bool    `json:"apply_adrs,omitempty"`    // For apply
	TaskStatus   string  `json:"task_status,omitempty"`   // For apply
	ADRStatus    string  `json:"adr_status,omitempty"`    // For apply
}

func handleManagePlan(ctx context.Context, req *mcp.CallToolRequest, input ManagePlanInput) (*mcp.CallToolResult, interface{}, error) {
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		return nil, nil, errors.New("action is required (create|status|list|apply|cancel)")
	}

	switch action {
	case "create":
		// Extract progress token before req is shadowed by API request
		progressToken := req.Params.GetProgressToken()
		session := req.Session

		requirements := strings.TrimSpace(input.Requirements)
		if requirements == "" {
			return nil, nil, errors.New("requirements is required for create action")
		}
		apiReq := api.CreatePlanRequest{
			Requirements: requirements,
		}
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

		// If client provided a progress token, watch the plan until completion
		if progressToken != nil && session != nil {
			if finalPlan := watchPlanProgress(ctx, session, progressToken, plan.ID); finalPlan != nil {
				plan = finalPlan
			}
		}

		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": "Plan created",
			"plan":    plan,
		}), nil, nil

	case "status":
		planID := strings.TrimSpace(input.PlanID)
		if planID == "" {
			return nil, nil, errors.New("plan_id is required for status action")
		}
		plan, err := apiClient.GetPlan(planID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get plan: %w", err)
		}
		return mustTextResult(plan), nil, nil

	case "list":
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

	case "apply":
		planID := strings.TrimSpace(input.PlanID)
		if planID == "" {
			return nil, nil, errors.New("plan_id is required for apply action")
		}
		req := api.ApplyPlanRequest{
			ApplyTasks: input.ApplyTasks,
			ApplyADRs:  input.ApplyADRs,
		}
		if s := strings.TrimSpace(input.TaskStatus); s != "" {
			req.TaskStatus = s
		}
		if s := strings.TrimSpace(input.ADRStatus); s != "" {
			req.ADRStatus = s
		}
		result, err := apiClient.ApplyPlan(planID, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to apply plan: %w", err)
		}
		return mustTextResult(result), nil, nil

	case "cancel":
		planID := strings.TrimSpace(input.PlanID)
		if planID == "" {
			return nil, nil, errors.New("plan_id is required for cancel action")
		}
		if err := apiClient.CancelPlan(planID); err != nil {
			return nil, nil, fmt.Errorf("failed to cancel plan: %w", err)
		}
		return mustTextResult(map[string]interface{}{
			"ok":      true,
			"message": "Plan cancelled",
			"plan_id": planID,
		}), nil, nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Must be: create, status, list, apply, or cancel", action)
	}
}

// watchPlanProgress polls plan status and sends MCP progress notifications.
// It blocks until the plan reaches a terminal state or times out.
// Returns the final plan state.
func watchPlanProgress(ctx context.Context, session *mcp.ServerSession, token any, planID string) *api.Plan {
	const (
		pollInterval = 3 * time.Second
		timeout      = 5 * time.Minute
	)

	deadline := time.After(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastPlan *api.Plan
	lastStatus := ""

	// Send initial notification
	_ = session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Progress:      0,
		Total:         100,
		Message:       "Plan created, starting execution...",
	})

	for {
		select {
		case <-ctx.Done():
			return lastPlan
		case <-deadline:
			return lastPlan
		case <-ticker.C:
			plan, err := apiClient.GetPlan(planID)
			if err != nil {
				continue // retry on error
			}
			lastPlan = plan

			// Send progress notification if status changed
			if plan.Status != lastStatus {
				lastStatus = plan.Status

				msg, progress := planStatusMessage(plan)

				_ = session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
					ProgressToken: token,
					Progress:      progress,
					Total:         100,
					Message:       msg,
				})
			}

			// Check for terminal states
			switch plan.Status {
			case "completed", "failed", "cancelled":
				return plan
			}
		}
	}
}

// planStatusMessage returns a human-readable message and progress value for a plan status.
func planStatusMessage(plan *api.Plan) (string, float64) {
	progress := float64(plan.Progress)

	// If the backend provides progress, use it; otherwise estimate from status
	if progress == 0 {
		switch plan.Status {
		case "pending":
			progress = 0
		case "routing":
			progress = 10
		case "discovery":
			progress = 30
		case "defining":
			progress = 50
		case "developing":
			progress = 70
		case "delivering":
			progress = 90
		case "completed":
			progress = 100
		}
	}

	var msg string
	switch plan.Status {
	case "routing":
		msg = "Phase 1/5: Routing - Selecting agents..."
	case "discovery":
		msg = "Phase 2/5: Discovery - Analyzing requirements..."
	case "defining":
		msg = "Phase 3/5: Defining - Creating proposals..."
	case "developing":
		msg = "Phase 4/5: Developing - Building consensus..."
	case "delivering":
		msg = "Phase 5/5: Delivering - Generating artifacts..."
	case "completed":
		msg = "Plan completed successfully"
	case "failed":
		msg = "Plan failed"
		if plan.Error != "" {
			msg += ": " + plan.Error
		}
	case "cancelled":
		msg = "Plan cancelled"
	default:
		msg = fmt.Sprintf("Status: %s", plan.Status)
	}

	return msg, progress
}
