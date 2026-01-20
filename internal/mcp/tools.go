package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sort"
	"strings"
)

// ============================================================================
// Encryption Helper Functions for Zero-Knowledge Decryption
// ============================================================================

// decryptMemoryContent decrypts memory content if encrypted and vault is unlocked.
// Returns the plaintext content or a fallback message.
func decryptMemoryContent(m *models.Memory) string {
	if !m.IsEncrypted {
		return m.Content
	}

	// Check if we have encrypted content to decrypt
	if m.EncryptedContent == "" {
		// Content might be "[Encrypted]" placeholder from backend
		return m.Content
	}

	if !crypto.IsVaultUnlocked() {
		return "[Vault Locked - Unlock to view]"
	}

	plaintext, err := crypto.DecryptContent(m.EncryptedContent, m.ContentNonce, true)
	if err != nil {
		return "[Decryption Failed]"
	}

	return plaintext
}

// decryptTaskFields decrypts task title and description if encrypted and vault is unlocked.
// Returns decrypted title and description.
func decryptTaskFields(t *models.Task) (title, description string) {
	if !t.IsEncrypted {
		return t.Title, t.Description
	}

	// Check if vault is unlocked
	if !crypto.IsVaultUnlocked() {
		title = "[Vault Locked - Unlock to view]"
		description = "[Vault Locked - Unlock to view]"
		if t.Title != "" && t.Title != "[Encrypted]" {
			title = t.Title
		}
		if t.Description != "" && t.Description != "[Encrypted]" {
			description = t.Description
		}
		return title, description
	}

	// Decrypt title
	if t.EncryptedTitle != "" {
		decrypted, err := crypto.DecryptContent(t.EncryptedTitle, t.TitleNonce, true)
		if err != nil {
			title = "[Decryption Failed]"
		} else {
			title = decrypted
		}
	} else {
		title = t.Title
	}

	// Decrypt description
	if t.EncryptedDescription != "" {
		decrypted, err := crypto.DecryptContent(t.EncryptedDescription, t.DescriptionNonce, true)
		if err != nil {
			description = "[Decryption Failed]"
		} else {
			description = decrypted
		}
	} else {
		description = t.Description
	}

	return title, description
}

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
		Description: "🔴 ESSENTIAL | Initialize agent session. ⚠️ CALL THIS FIRST! Provide your agent name and model for tracking. Returns current context, active project, pending tasks, and recommended actions.",
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
		Description: "⚠️ DEPRECATED | Set the active project. Use explicit 'project' parameter in tools instead.",
	}, handleSetActiveProject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_project",
		Description: "🟢 ADVANCED | Create a new project. ⚠️ Auto-checks for duplicates - returns similar projects if found. Use force=true to bypass.",
	}, handleCreateProject)

	// ============================================================================
	// 🔴 ESSENTIAL - Task Management
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "🔴 ESSENTIAL | List tasks with filtering. REQUIRED: project parameter. 💡 Call before create_task to check for duplicates.",
	}, handleListTasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_task",
		Description: "🔴 ESSENTIAL | Create a new task. REQUIRED: project, description parameters. ⚠️ Always check list_tasks first to avoid duplicates!",
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
		Name:        "move_task",
		Description: "🟡 COMMON | Move a task to a different project. Use this to fix tasks created in the wrong location.",
	}, handleMoveTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_next_tasks",
		Description: "🔴 ESSENTIAL | Get prioritized TODO tasks. REQUIRED: project parameter. 💡 Use at session start to see what needs attention.",
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
		Description: "🔴 ESSENTIAL | Store important information to knowledge base. REQUIRED: project, content parameters. 💡 If it matters later, add it here!",
	}, handleAddMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_memories",
		Description: "🔴 ESSENTIAL | List memories with filtering. REQUIRED: project parameter.",
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
	// 🟡 COMMON - Context Pack Management
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_context_packs",
		Description: "🟡 COMMON | List all context packs with optional filtering by type and status.",
	}, handleListContextPacks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_context_pack",
		Description: "🟡 COMMON | Create a new context pack. Use to bundle related contexts, memories, and tasks.",
	}, handleCreateContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_context_pack",
		Description: "🟡 COMMON | Get detailed context pack info including linked memories, tasks, and contexts.",
	}, handleGetContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_context_pack",
		Description: "🟡 COMMON | Update a context pack's name, description, type, or status.",
	}, handleUpdateContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_context_pack",
		Description: "🟢 ADVANCED | Delete a context pack. ⚠️ Requires explicit user approval.",
	}, handleDeleteContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_memory_to_pack",
		Description: "🟡 COMMON | Add an existing memory to a context pack.",
	}, handleAddMemoryToPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_task_to_pack",
		Description: "🟡 COMMON | Add an existing task to a context pack.",
	}, handleAddTaskToPack)

	// ============================================================================
	// 🟡 COMMON - Decisions (ADRs)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_decision",
		Description: "🟡 COMMON | Record an architectural decision (ADR). ⚠️ Agent creates DRAFTS only - user must approve. Use for important technical choices.",
	}, handleCreateDecision)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_decisions",
		Description: "🟡 COMMON | List architectural decisions. Review past decisions before making new ones.",
	}, handleListDecisions)

	// ============================================================================
	// 🟡 COMMON - AI Features
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai_next_step",
		Description: "🟡 COMMON | Get AI-suggested next actionable step for a task based on project context.",
	}, handleAINextStep)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai_estimate_time",
		Description: "🟡 COMMON | Get AI-estimated time to complete a task.",
	}, handleAIEstimateTime)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai_analyze_risks",
		Description: "🟡 COMMON | Get AI analysis of potential risks and blockers for a task.",
	}, handleAIAnalyzeRisks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai_find_dependencies",
		Description: "🟡 COMMON | Get AI-identified dependencies and prerequisites for a task.",
	}, handleAIFindDependencies)

	// ============================================================================
	// 🟡 COMMON - Organizations
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_organizations",
		Description: "🟡 COMMON | List user's organizations.",
	}, handleListOrganizations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_organization",
		Description: "🟡 COMMON | Get organization details.",
	}, handleGetOrganization)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_organization",
		Description: "🟢 ADVANCED | Create new organization.",
	}, handleCreateOrganization)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_organization",
		Description: "🟢 ADVANCED | Update organization.",
	}, handleUpdateOrganization)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_organization_members",
		Description: "🟡 COMMON | List organization members.",
	}, handleGetOrganizationMembers)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "invite_to_organization",
		Description: "🟢 ADVANCED | Invite member to organization by email.",
	}, handleInviteToOrganization)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "switch_organization",
		Description: "🟡 COMMON | Switch active organization context.",
	}, handleSwitchOrganization)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_active_organization",
		Description: "🔴 ESSENTIAL | Get current active organization.",
	}, handleGetActiveOrganization)

	// ============================================================================
	// 🟡 COMMON - Reports
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stats",
		Description: "🟡 COMMON | Get task statistics and completion rates. REQUIRED: project parameter.",
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

type SetupAgentInput struct {
	AgentName  string `json:"agent_name,omitempty"`
	AgentModel string `json:"agent_model,omitempty"`
}

func handleSetupAgent(ctx context.Context, req *mcp.CallToolRequest, input SetupAgentInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// Initialize session with agent info
	agentName := strings.TrimSpace(input.AgentName)
	if agentName == "" {
		agentName = "unknown-agent"
	}
	agentModel := strings.TrimSpace(input.AgentModel)

	// Initialize the session
	session := InitializeSession(agentName, agentModel)

	result, err := setupAgent(apiClient)
	if err != nil {
		return nil, nil, err
	}

	// Add session info to result
	result["session"] = map[string]interface{}{
		"id":          session.ID,
		"agent_name":  session.AgentName,
		"agent_model": session.AgentModel,
		"initialized": session.Initialized,
	}

	// Add clear next steps for workflow enforcement
	result["workflow_guide"] = map[string]interface{}{
		"step_1":  "✅ setup_agent called - session initialized",
		"step_2":  "📋 Call list_projects to see available projects",
		"step_3":  "🎯 Call set_active_project to choose your working project",
		"step_4":  "📝 Now you can create tasks and memories in that project",
		"warning": "⚠️ Tasks/memories created without set_active_project will use Personal workspace",
	}

	return nil, result, nil
}

func handleListProjects(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	projects, err := apiClient.ListProjects()
	if err != nil {
		return nil, nil, err
	}
	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(projects, getContextString()), nil
}

type SetActiveProjectInput struct {
	ProjectName string `json:"projectName"`
}

func handleSetActiveProject(ctx context.Context, req *mcp.CallToolRequest, input SetActiveProjectInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// Check session initialization (but allow this tool even without init since it's part of setup workflow)
	if err := checkSessionInit("set_active_project"); err != nil {
		return nil, nil, err
	}

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

			// Store in session for workflow tracking
			SetSessionProject(p.ID)

			return nil, map[string]interface{}{
				"ok":         true,
				"project_id": p.ID.String(),
				"name":       p.Name,
				"_context":   getContextString(),
				"_message":   "✅ Active project set to: " + p.Name,
				"_warning":   "⚠️ DEPRECATED: set_active_project is deprecated. Use explicit 'project' parameter in create_task, add_memory, list_tasks, etc. instead. This tool will be removed in a future version.",
			}, nil
		}
	}
	return nil, nil, errors.New("project not found")
}

type CreateProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Force       bool   `json:"force,omitempty"` // Set true to bypass similarity check
	OrgID       string `json:"org_id,omitempty"` // Organization ID for scoping
}

func handleCreateProject(ctx context.Context, req *mcp.CallToolRequest, input CreateProjectInput) (*mcp.CallToolResult, interface{}, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}

	// Step 1: Check for similar projects (unless force=true)
	if !input.Force {
		suggestions, err := apiClient.SuggestProjects(name, strings.TrimSpace(input.OrgID))
		if err == nil && suggestions != nil {
			// If exact match exists, don't create duplicate
			if suggestions.ExactMatch != nil {
				return nil, map[string]interface{}{
					"status":        "exists",
					"exact_match":   suggestions.ExactMatch,
					"_message":      "✅ Project already exists with this name. Use existing project instead of creating a duplicate.",
					"_action":       "Use set_active_project or specify this project name in your create_task/add_memory calls.",
				}, nil
			}

			// If similar projects found (>60% similarity), warn and require confirmation
			if len(suggestions.Similar) > 0 {
				// Build similar projects list for response
				similarList := make([]map[string]interface{}, 0, len(suggestions.Similar))
				for _, s := range suggestions.Similar {
					if s.Similarity >= 0.6 { // Only show projects with >60% similarity
						similarList = append(similarList, map[string]interface{}{
							"id":         s.ID,
							"name":       s.Name,
							"similarity": fmt.Sprintf("%.0f%%", s.Similarity*100),
							"org_name":   s.OrgName,
						})
					}
				}

				if len(similarList) > 0 {
					return nil, map[string]interface{}{
						"status":           "needs_confirmation",
						"similar_projects": similarList,
						"requested_name":   name,
						"_message":         "⚠️ Similar projects found. To avoid duplicates:\n1. Use an existing project from the list above, OR\n2. Call create_project with force=true to create anyway",
						"_action":          "Either use set_active_project(projectName=\"existing-name\") or call create_project(name=\"...\", force=true)",
					}, nil
				}
			}
		}
		// If suggest API fails or returns no similar, proceed with creation
	}

	// Step 2: Create the project
	project, err := apiClient.CreateProject(name, strings.TrimSpace(input.Description))
	if err != nil {
		return nil, nil, err
	}
	return nil, map[string]interface{}{
		"project":  project,
		"_message": "✅ Project created successfully: " + project.Name,
	}, nil
}

type ListTasksInput struct {
	Status  string  `json:"status,omitempty"`
	Project string  `json:"project"` // REQUIRED - project name or ID
	Limit   float64 `json:"limit,omitempty"`
}

func handleListTasks(ctx context.Context, req *mcp.CallToolRequest, input ListTasksInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("❌ 'project' parameter is REQUIRED. Specify which project to list tasks from.\n\nUse list_projects to see available projects.\nExample: list_tasks(project=\"my-project\")")
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := apiClient.ListTasks(projectID, strings.TrimSpace(input.Status))
	if err != nil {
		return nil, nil, err
	}
	limit := int(input.Limit)
	if limit > 0 && limit < len(tasks) {
		tasks = tasks[:limit]
	}

	// Decrypt task fields before returning
	var decryptedTasks []map[string]interface{}
	for _, t := range tasks {
		decryptedTitle, decryptedDesc := decryptTaskFields(&t)
		taskMap := map[string]interface{}{
			"id":           t.ID.String(),
			"project_id":   t.ProjectID.String(),
			"title":        decryptedTitle,
			"description":  decryptedDesc,
			"status":       t.Status,
			"priority":     t.Priority,
			"tags":         t.Tags,
			"created_at":   t.CreatedAt,
			"updated_at":   t.UpdatedAt,
			"is_encrypted": t.IsEncrypted,
		}
		if t.Project != nil {
			taskMap["project"] = map[string]interface{}{
				"id":   t.Project.ID.String(),
				"name": t.Project.Name,
			}
		}
		if len(t.Annotations) > 0 {
			taskMap["annotations"] = t.Annotations
		}
		decryptedTasks = append(decryptedTasks, taskMap)
	}

	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(decryptedTasks, getContextString()), nil
}

type CreateTaskInput struct {
	Description string `json:"description"`
	Priority    string `json:"priority,omitempty"`
	Project     string `json:"project"` // REQUIRED - project name or ID
}

func handleCreateTask(ctx context.Context, req *mcp.CallToolRequest, input CreateTaskInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// Check session initialization
	if err := checkSessionInit("create_task"); err != nil {
		return nil, nil, err
	}

	description := strings.TrimSpace(input.Description)
	if description == "" {
		return nil, nil, errors.New("description is required")
	}

	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("❌ 'project' parameter is REQUIRED. Specify which project this task belongs to.\n\nUse list_projects to see available projects.\nExample: create_task(description=\"...\", project=\"my-project\")")
	}

	priority := normalizePriority(input.Priority)
	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	// Get agent metadata from current session
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

	// Check if user has encryption enabled and vault is unlocked
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled {
		if !crypto.IsVaultUnlocked() {
			return nil, nil, errors.New("🔒 Vault is locked. Your account has encryption enabled.\n\n" +
				"To unlock, the user must run:\n" +
				"  ramorie setup unlock\n\n" +
				"This only needs to be done once per session (until computer restarts).\n" +
				"Please inform the user to unlock their vault.")
		}

		// Encrypt task description with vault key
		encryptedDesc, nonce, isEncrypted, err := crypto.EncryptContent(description)
		if err != nil {
			return nil, nil, fmt.Errorf("encryption failed: %w", err)
		}

		if isEncrypted {
			// Use encrypted task creation with agent metadata
			task, err := apiClient.CreateEncryptedTaskWithMeta(projectID, encryptedDesc, nonce, priority, meta)
			if err != nil {
				return nil, nil, err
			}

			result := formatMCPResponse(task, getContextString())
			result["_created_in_project"] = projectID
			result["_encrypted"] = true
			if session != nil {
				result["_created_by_agent"] = session.AgentName
			}
			result["_message"] = "✅ Task created (encrypted) in project " + projectID[:8] + "..."
			return nil, result, nil
		}
	}

	// Non-encrypted task creation (user doesn't have encryption enabled)
	task, err := apiClient.CreateTaskWithMeta(projectID, description, "", priority, meta)
	if err != nil {
		return nil, nil, err
	}

	// Return with context info showing where task was created
	result := formatMCPResponse(task, getContextString())
	result["_created_in_project"] = projectID
	if session != nil {
		result["_created_by_agent"] = session.AgentName
	}
	result["_message"] = "✅ Task created successfully in project " + projectID[:8] + "..."

	return nil, result, nil
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

	// Decrypt task fields before returning
	decryptedTitle, decryptedDesc := decryptTaskFields(task)
	result := map[string]interface{}{
		"id":           task.ID.String(),
		"project_id":   task.ProjectID.String(),
		"title":        decryptedTitle,
		"description":  decryptedDesc,
		"status":       task.Status,
		"priority":     task.Priority,
		"tags":         task.Tags,
		"created_at":   task.CreatedAt,
		"updated_at":   task.UpdatedAt,
		"is_encrypted": task.IsEncrypted,
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

	return nil, result, nil
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

type MoveTaskInput struct {
	TaskID    string `json:"taskId"`
	ProjectID string `json:"projectId"`
}

func handleMoveTask(ctx context.Context, req *mcp.CallToolRequest, input MoveTaskInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// Check session initialization
	if err := checkSessionInit("move_task"); err != nil {
		return nil, nil, err
	}

	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}

	targetProject := strings.TrimSpace(input.ProjectID)
	if targetProject == "" {
		return nil, nil, errors.New("projectId is required - specify target project ID or name")
	}

	// Resolve project ID (can be name or UUID)
	projectID, err := resolveProjectID(apiClient, targetProject)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve project: %w", err)
	}

	// Update task with new project_id
	task, err := apiClient.UpdateTask(taskID, map[string]interface{}{
		"project_id": projectID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to move task: %w", err)
	}

	result := formatMCPResponse(task, getContextString())
	result["_moved_to_project"] = projectID
	result["_message"] = "✅ Task moved successfully to project " + projectID[:8] + "..."

	return nil, result, nil
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
	Count   float64 `json:"count,omitempty"`
	Project string  `json:"project"` // REQUIRED - project name or ID
}

func handleGetNextTasks(ctx context.Context, req *mcp.CallToolRequest, input GetNextTasksInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("❌ 'project' parameter is REQUIRED. Specify which project to get next tasks from.\n\nUse list_projects to see available projects.\nExample: get_next_tasks(project=\"my-project\")")
	}

	count := int(input.Count)
	if count <= 0 {
		count = 5
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
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
	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(tasks, getContextString()), nil
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

	// Check if user has encryption enabled and vault is unlocked
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled {
		if !crypto.IsVaultUnlocked() {
			return nil, nil, errors.New("🔒 Vault is locked. Your account has encryption enabled.\n\n" +
				"To unlock, the user must run:\n" +
				"  ramorie setup unlock\n\n" +
				"This only needs to be done once per session (until computer restarts).\n" +
				"Please inform the user to unlock their vault.")
		}

		// Encrypt note content with vault key
		encryptedNote, nonce, isEncrypted, err := crypto.EncryptContent(note)
		if err != nil {
			return nil, nil, fmt.Errorf("encryption failed: %w", err)
		}

		if isEncrypted {
			annotation, err := apiClient.CreateEncryptedAnnotation(taskID, encryptedNote, nonce)
			if err != nil {
				return nil, nil, err
			}
			return nil, annotation, nil
		}
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
	Status  string  `json:"status,omitempty"`
	Project string  `json:"project,omitempty"`
	Limit   float64 `json:"limit,omitempty"`
}

func handleSearchTasks(ctx context.Context, req *mcp.CallToolRequest, input SearchTasksInput) (*mcp.CallToolResult, map[string]interface{}, error) {
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
	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(tasks, getContextString()), nil
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
	Project string `json:"project"` // REQUIRED - project name or ID
}

func handleAddMemory(ctx context.Context, req *mcp.CallToolRequest, input AddMemoryInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// Check session initialization
	if err := checkSessionInit("add_memory"); err != nil {
		return nil, nil, err
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, nil, errors.New("content is required")
	}

	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("❌ 'project' parameter is REQUIRED. Specify which project this memory belongs to.\n\nUse list_projects to see available projects.\nExample: add_memory(content=\"...\", project=\"my-project\")")
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	// Check if user has encryption enabled and vault is unlocked
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled {
		if !crypto.IsVaultUnlocked() {
			return nil, nil, errors.New("🔒 Vault is locked. Your account has encryption enabled.\n\n" +
				"To unlock, the user must run:\n" +
				"  ramorie setup unlock\n\n" +
				"This only needs to be done once per session (until computer restarts).\n" +
				"Please inform the user to unlock their vault.")
		}

		// Encrypt memory content with vault key
		encryptedContent, nonce, isEncrypted, err := crypto.EncryptContent(content)
		if err != nil {
			return nil, nil, fmt.Errorf("encryption failed: %w", err)
		}

		if isEncrypted {
			// Use encrypted memory creation
			memory, err := apiClient.CreateEncryptedMemory(projectID, encryptedContent, nonce)
			if err != nil {
				return nil, nil, err
			}

			result := formatMCPResponse(memory, getContextString())
			result["_created_in_project"] = projectID
			result["_encrypted"] = true
			result["_message"] = "✅ Memory created (encrypted) in project " + projectID[:8] + "..."
			return nil, result, nil
		}
	}

	// Non-encrypted memory creation (user doesn't have encryption enabled)
	memory, err := apiClient.CreateMemory(projectID, content)
	if err != nil {
		return nil, nil, err
	}

	// Return with context info showing where memory was created
	result := formatMCPResponse(memory, getContextString())
	result["_created_in_project"] = projectID
	result["_message"] = "✅ Memory created successfully in project " + projectID[:8] + "..."

	return nil, result, nil
}

type ListMemoriesInput struct {
	Project string  `json:"project"` // REQUIRED - project name or ID
	Term    string  `json:"term,omitempty"`
	Limit   float64 `json:"limit,omitempty"`
}

func handleListMemories(ctx context.Context, req *mcp.CallToolRequest, input ListMemoriesInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("❌ 'project' parameter is REQUIRED. Specify which project to list memories from.\n\nUse list_projects to see available projects.\nExample: list_memories(project=\"my-project\")")
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}
	memories, err := apiClient.ListMemories(projectID, "")
	if err != nil {
		return nil, nil, err
	}

	// Filter by term using decrypted content
	term := strings.TrimSpace(input.Term)
	if term != "" {
		filtered := memories[:0]
		for _, m := range memories {
			// CRITICAL: Decrypt content before filtering
			decryptedContent := decryptMemoryContent(&m)
			if strings.Contains(strings.ToLower(decryptedContent), strings.ToLower(term)) {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}

	limit := int(input.Limit)
	if limit > 0 && limit < len(memories) {
		memories = memories[:limit]
	}

	// Decrypt all memory content before returning
	var decryptedMemories []map[string]interface{}
	for _, m := range memories {
		decryptedContent := decryptMemoryContent(&m)
		memMap := map[string]interface{}{
			"id":           m.ID.String(),
			"project_id":   m.ProjectID.String(),
			"content":      decryptedContent,
			"tags":         m.Tags,
			"created_at":   m.CreatedAt,
			"updated_at":   m.UpdatedAt,
			"is_encrypted": m.IsEncrypted,
		}
		if m.LinkedTaskID != nil {
			memMap["linked_task_id"] = m.LinkedTaskID.String()
		}
		if m.Project != nil {
			memMap["project"] = map[string]interface{}{
				"id":   m.Project.ID.String(),
				"name": m.Project.Name,
			}
		}
		decryptedMemories = append(decryptedMemories, memMap)
	}

	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(decryptedMemories, getContextString()), nil
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

	// Decrypt memory content before returning
	decryptedContent := decryptMemoryContent(memory)
	result := map[string]interface{}{
		"id":           memory.ID.String(),
		"project_id":   memory.ProjectID.String(),
		"content":      decryptedContent,
		"tags":         memory.Tags,
		"created_at":   memory.CreatedAt,
		"updated_at":   memory.UpdatedAt,
		"is_encrypted": memory.IsEncrypted,
	}
	if memory.LinkedTaskID != nil {
		result["linked_task_id"] = memory.LinkedTaskID.String()
	}
	if memory.Project != nil {
		result["project"] = map[string]interface{}{
			"id":   memory.Project.ID.String(),
			"name": memory.Project.Name,
		}
	}

	return nil, result, nil
}

type RecallInput struct {
	Term             string  `json:"term"`
	Project          string  `json:"project,omitempty"`
	Tag              string  `json:"tag,omitempty"`
	LinkedTask       bool    `json:"linked_task,omitempty"`
	IncludeRelations bool    `json:"include_relations,omitempty"`
	Limit            float64 `json:"limit,omitempty"`
	MinScore         float64 `json:"min_score,omitempty"`
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

		// CRITICAL: Decrypt memory content if encrypted
		// Without this, encrypted memories search against "[Encrypted]" and never match
		decryptedContent := decryptMemoryContent(&m)
		contentLower := strings.ToLower(decryptedContent)
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
			"content":    decryptedContent, // Use decrypted content in result
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

type ListContextPacksInput struct {
	Type   string  `json:"type,omitempty"`   // project, integration, decision, custom
	Status string  `json:"status,omitempty"` // draft, published
	Query  string  `json:"query,omitempty"`
	Limit  float64 `json:"limit,omitempty"`
}

func handleListContextPacks(ctx context.Context, req *mcp.CallToolRequest, input ListContextPacksInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	limit := int(input.Limit)
	if limit == 0 {
		limit = 50
	}
	result, err := apiClient.ListContextPacks(
		strings.TrimSpace(input.Type),
		strings.TrimSpace(input.Status),
		strings.TrimSpace(input.Query),
		limit,
		0,
	)
	if err != nil {
		return nil, nil, err
	}
	// Wrap response to fix "expected record, received array" error
	return nil, formatMCPResponse(result, getContextString()), nil
}

type CreateContextPackInput struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // project, integration, decision, custom
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func handleCreateContextPack(ctx context.Context, req *mcp.CallToolRequest, input CreateContextPackInput) (*mcp.CallToolResult, interface{}, error) {
	name := strings.TrimSpace(input.Name)
	packType := strings.TrimSpace(input.Type)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}
	if packType == "" {
		return nil, nil, errors.New("type is required (project, integration, decision, custom)")
	}
	pack, err := apiClient.CreateContextPack(
		name,
		packType,
		strings.TrimSpace(input.Description),
		"draft", // Always create as draft
		input.Tags,
	)
	if err != nil {
		return nil, nil, err
	}
	return nil, pack, nil
}

type GetContextPackInput struct {
	PackID string `json:"packId"`
}

func handleGetContextPack(ctx context.Context, req *mcp.CallToolRequest, input GetContextPackInput) (*mcp.CallToolResult, interface{}, error) {
	packID := strings.TrimSpace(input.PackID)
	if packID == "" {
		return nil, nil, errors.New("packId is required")
	}
	pack, err := apiClient.GetContextPack(packID)
	if err != nil {
		return nil, nil, err
	}
	return nil, pack, nil
}

type UpdateContextPackInput struct {
	PackID      string `json:"packId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

func handleUpdateContextPack(ctx context.Context, req *mcp.CallToolRequest, input UpdateContextPackInput) (*mcp.CallToolResult, interface{}, error) {
	packID := strings.TrimSpace(input.PackID)
	if packID == "" {
		return nil, nil, errors.New("packId is required")
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
		return nil, nil, errors.New("at least one field to update is required")
	}
	pack, err := apiClient.UpdateContextPack(packID, updates)
	if err != nil {
		return nil, nil, err
	}
	return nil, pack, nil
}

type DeleteContextPackInput struct {
	PackID string `json:"packId"`
}

func handleDeleteContextPack(ctx context.Context, req *mcp.CallToolRequest, input DeleteContextPackInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	packID := strings.TrimSpace(input.PackID)
	if packID == "" {
		return nil, nil, errors.New("packId is required")
	}
	if err := apiClient.DeleteContextPack(packID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]interface{}{
		"ok":      true,
		"message": "Context pack deleted",
	}, nil
}

type AddMemoryToPackInput struct {
	PackID   string `json:"packId"`
	MemoryID string `json:"memoryId"`
}

func handleAddMemoryToPack(ctx context.Context, req *mcp.CallToolRequest, input AddMemoryToPackInput) (*mcp.CallToolResult, interface{}, error) {
	packID := strings.TrimSpace(input.PackID)
	memoryID := strings.TrimSpace(input.MemoryID)
	if packID == "" || memoryID == "" {
		return nil, nil, errors.New("packId and memoryId are required")
	}

	// Get current pack to retrieve existing memory IDs
	pack, err := apiClient.GetContextPack(packID)
	if err != nil {
		return nil, nil, err
	}

	// Build memory IDs array
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

	updates := map[string]interface{}{
		"memoryIds": memoryIDs,
	}
	result, err := apiClient.UpdateContextPack(packID, updates)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

type AddTaskToPackInput struct {
	PackID string `json:"packId"`
	TaskID string `json:"taskId"`
}

func handleAddTaskToPack(ctx context.Context, req *mcp.CallToolRequest, input AddTaskToPackInput) (*mcp.CallToolResult, interface{}, error) {
	packID := strings.TrimSpace(input.PackID)
	taskID := strings.TrimSpace(input.TaskID)
	if packID == "" || taskID == "" {
		return nil, nil, errors.New("packId and taskId are required")
	}

	// Get current pack to retrieve existing task IDs
	pack, err := apiClient.GetContextPack(packID)
	if err != nil {
		return nil, nil, err
	}

	// Build task IDs array
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

	updates := map[string]interface{}{
		"taskIds": taskIDs,
	}
	result, err := apiClient.UpdateContextPack(packID, updates)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

// ============================================================================
// AI FEATURES HANDLERS
// ============================================================================

type AITaskInput struct {
	TaskID string `json:"taskId"`
}

func handleAINextStep(ctx context.Context, req *mcp.CallToolRequest, input AITaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	result, err := apiClient.AINextStep(taskID)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

func handleAIEstimateTime(ctx context.Context, req *mcp.CallToolRequest, input AITaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	result, err := apiClient.AIEstimateTime(taskID)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

func handleAIAnalyzeRisks(ctx context.Context, req *mcp.CallToolRequest, input AITaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	result, err := apiClient.AIRisks(taskID)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

func handleAIFindDependencies(ctx context.Context, req *mcp.CallToolRequest, input AITaskInput) (*mcp.CallToolResult, interface{}, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return nil, nil, errors.New("taskId is required")
	}
	result, err := apiClient.AIDependencies(taskID)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

// ============================================================================
// ORGANIZATION HANDLERS
// ============================================================================

func handleListOrganizations(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	orgs, err := apiClient.ListOrganizations()
	if err != nil {
		return nil, nil, err
	}
	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(orgs, getContextString()), nil
}

type GetOrganizationInput struct {
	OrgID string `json:"orgId"`
}

func handleGetOrganization(ctx context.Context, req *mcp.CallToolRequest, input GetOrganizationInput) (*mcp.CallToolResult, interface{}, error) {
	orgID := strings.TrimSpace(input.OrgID)
	if orgID == "" {
		return nil, nil, errors.New("orgId is required")
	}
	org, err := apiClient.GetOrganization(orgID)
	if err != nil {
		return nil, nil, err
	}
	return nil, org, nil
}

type CreateOrganizationInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func handleCreateOrganization(ctx context.Context, req *mcp.CallToolRequest, input CreateOrganizationInput) (*mcp.CallToolResult, interface{}, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}
	org, err := apiClient.CreateOrganization(name, strings.TrimSpace(input.Description))
	if err != nil {
		return nil, nil, err
	}
	return nil, org, nil
}

type UpdateOrganizationInput struct {
	OrgID       string `json:"orgId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

func handleUpdateOrganization(ctx context.Context, req *mcp.CallToolRequest, input UpdateOrganizationInput) (*mcp.CallToolResult, interface{}, error) {
	orgID := strings.TrimSpace(input.OrgID)
	if orgID == "" {
		return nil, nil, errors.New("orgId is required")
	}

	updates := make(map[string]interface{})
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}

	if len(updates) == 0 {
		return nil, nil, errors.New("at least one field (name or description) must be provided")
	}

	org, err := apiClient.UpdateOrganization(orgID, updates)
	if err != nil {
		return nil, nil, err
	}
	return nil, org, nil
}

func handleGetOrganizationMembers(ctx context.Context, req *mcp.CallToolRequest, input GetOrganizationInput) (*mcp.CallToolResult, interface{}, error) {
	orgID := strings.TrimSpace(input.OrgID)
	if orgID == "" {
		return nil, nil, errors.New("orgId is required")
	}

	members, err := apiClient.GetOrganizationMembers(orgID)
	if err != nil {
		return nil, nil, err
	}
	return nil, formatMCPResponse(members, getContextString()), nil
}

type InviteToOrganizationInput struct {
	OrgID string `json:"orgId"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

func handleInviteToOrganization(ctx context.Context, req *mcp.CallToolRequest, input InviteToOrganizationInput) (*mcp.CallToolResult, interface{}, error) {
	orgID := strings.TrimSpace(input.OrgID)
	email := strings.TrimSpace(input.Email)
	if orgID == "" || email == "" {
		return nil, nil, errors.New("orgId and email are required")
	}

	role := strings.TrimSpace(input.Role)
	if role == "" {
		role = "member"
	}

	result, err := apiClient.InviteToOrganization(orgID, email, role)
	if err != nil {
		return nil, nil, err
	}
	return nil, result, nil
}

type SwitchOrganizationInput struct {
	OrgID string `json:"orgId"`
}

func handleSwitchOrganization(ctx context.Context, req *mcp.CallToolRequest, input SwitchOrganizationInput) (*mcp.CallToolResult, interface{}, error) {
	orgID := strings.TrimSpace(input.OrgID)
	if orgID == "" {
		return nil, nil, errors.New("orgId is required")
	}

	// Switch organization via API
	org, err := apiClient.SwitchOrganization(orgID)
	if err != nil {
		return nil, nil, err
	}

	// Update session with the new active organization
	orgUUID, parseErr := uuid.Parse(orgID)
	if parseErr == nil {
		SetSessionOrganization(orgUUID)
	}

	return nil, org, nil
}

func handleGetActiveOrganization(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, interface{}, error) {
	session := GetCurrentSession()

	// If session has an active organization, get its details
	if session.ActiveOrgID != nil {
		org, err := apiClient.GetOrganization(session.ActiveOrgID.String())
		if err != nil {
			return nil, nil, err
		}
		return nil, org, nil
	}

	// Otherwise, list organizations and return info
	orgs, err := apiClient.ListOrganizations()
	if err != nil {
		return nil, nil, err
	}

	if len(orgs) == 0 {
		return nil, nil, errors.New("no organizations found")
	}

	// If only one organization, return it as the default
	if len(orgs) == 1 {
		return nil, &orgs[0], nil
	}

	// Multiple organizations - inform user to select one
	return nil, map[string]interface{}{
		"message":       "Multiple organizations found. Use switch_organization to select one.",
		"organizations": orgs,
	}, nil
}

type CreateDecisionInput struct {
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	Status       string `json:"status,omitempty"`
	Area         string `json:"area,omitempty"`
	Context      string `json:"context,omitempty"`
	Consequences string `json:"consequences,omitempty"`
}

func handleCreateDecision(ctx context.Context, req *mcp.CallToolRequest, input CreateDecisionInput) (*mcp.CallToolResult, interface{}, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}

	// AGENT ENFORCEMENT: Force draft status and agent source
	// Agents can only create drafts - users must approve them
	status := "draft"
	source := "agent"

	decision, err := apiClient.CreateDecision(
		title,
		strings.TrimSpace(input.Description),
		status, // Always "draft" for agent-created decisions
		strings.TrimSpace(input.Area),
		strings.TrimSpace(input.Context),
		strings.TrimSpace(input.Consequences),
		source, // Always "agent" for MCP-created decisions
	)
	if err != nil {
		return nil, nil, err
	}
	return nil, decision, nil
}

type ListDecisionsInput struct {
	Status string  `json:"status,omitempty"`
	Area   string  `json:"area,omitempty"`
	Limit  float64 `json:"limit,omitempty"`
}

func handleListDecisions(ctx context.Context, req *mcp.CallToolRequest, input ListDecisionsInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	decisions, err := apiClient.ListDecisions(strings.TrimSpace(input.Status), strings.TrimSpace(input.Area), int(input.Limit))
	if err != nil {
		return nil, nil, err
	}
	// Wrap array response to fix "expected record, received array" error
	return nil, formatMCPResponse(decisions, getContextString()), nil
}

type GetStatsInput struct {
	Project string `json:"project"` // REQUIRED - project name or ID
}

func handleGetStats(ctx context.Context, req *mcp.CallToolRequest, input GetStatsInput) (*mcp.CallToolResult, interface{}, error) {
	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("❌ 'project' parameter is REQUIRED. Specify which project to get stats for.\n\nUse list_projects to see available projects.\nExample: get_stats(project=\"my-project\")")
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	// Get stats for specific project
	b, err := apiClient.Request("GET", fmt.Sprintf("/reports/stats?project_id=%s", projectID), nil)
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
	Format  string `json:"format,omitempty"`
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
			Description: "🔴 ESSENTIAL | Initialize agent session. ⚠️ CALL THIS FIRST! Provide your agent name and model for tracking. Returns current context, active project, pending tasks, and recommended actions.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_name":  map[string]interface{}{"type": "string", "description": "Your agent identifier (e.g., 'claude-code', 'cursor', 'gemini')"},
					"agent_model": map[string]interface{}{"type": "string", "description": "Model being used (e.g., 'claude-opus-4-5-20250514', 'gpt-4')"},
				},
			},
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
			Description: "⚠️ DEPRECATED | Set the active project. Use explicit 'project' parameter in tools instead.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectName": map[string]interface{}{"type": "string", "description": "Project name or ID"}}, "required": []string{"projectName"}},
		},

		// ============================================================================
		// 🔴 ESSENTIAL - Task Management (Core)
		// ============================================================================
		{
			Name:        "list_tasks",
			Description: "🔴 ESSENTIAL | List tasks with filtering. REQUIRED: project parameter. 💡 Call before create_task to check for duplicates.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "description": "Filter: TODO, IN_PROGRESS, COMPLETED"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID (REQUIRED)"}, "limit": map[string]interface{}{"type": "number", "description": "Max results"}}, "required": []string{"project"}},
		},
		{
			Name:        "create_task",
			Description: "🔴 ESSENTIAL | Create a new task. REQUIRED: project, description parameters. ⚠️ Always check list_tasks first to avoid duplicates!",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"description": map[string]interface{}{"type": "string", "description": "Task description - clear and actionable"}, "priority": map[string]interface{}{"type": "string", "description": "Priority: H=High, M=Medium, L=Low"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID (REQUIRED)"}}, "required": []string{"description", "project"}},
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
			Description: "🔴 ESSENTIAL | Get prioritized TODO tasks. REQUIRED: project parameter. 💡 Use at session start to see what needs attention.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"count": map[string]interface{}{"type": "number", "description": "Number of tasks (default: 5)"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID (REQUIRED)"}}, "required": []string{"project"}},
		},

		// ============================================================================
		// 🔴 ESSENTIAL - Memory Management (Core)
		// ============================================================================
		{
			Name:        "add_memory",
			Description: "🔴 ESSENTIAL | Store important information to knowledge base. REQUIRED: project, content parameters. 💡 If it matters later, add it here!",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"content": map[string]interface{}{"type": "string", "description": "Memory content - be descriptive"}, "project": map[string]interface{}{"type": "string", "description": "Project name or ID (REQUIRED)"}}, "required": []string{"content", "project"}},
		},
		{
			Name:        "list_memories",
			Description: "🔴 ESSENTIAL | List memories with filtering. REQUIRED: project parameter.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string", "description": "Project name or ID (REQUIRED)"}, "term": map[string]interface{}{"type": "string", "description": "Filter by keyword"}, "limit": map[string]interface{}{"type": "number"}}, "required": []string{"project"}},
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
		// 🟡 COMMON - Context Pack Management
		// ============================================================================
		{
			Name:        "list_context_packs",
			Description: "🟡 COMMON | List all context packs with optional filtering by type and status.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type":   map[string]interface{}{"type": "string", "description": "Filter by type: project, integration, decision, custom"},
					"status": map[string]interface{}{"type": "string", "description": "Filter by status: draft, published"},
					"query":  map[string]interface{}{"type": "string", "description": "Search query"},
					"limit":  map[string]interface{}{"type": "number", "description": "Max results (default: 50)"},
				},
			},
		},
		{
			Name:        "create_context_pack",
			Description: "🟡 COMMON | Create a new context pack. Use to bundle related contexts, memories, and tasks.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":        map[string]interface{}{"type": "string", "description": "Pack name"},
					"type":        map[string]interface{}{"type": "string", "description": "Pack type: project, integration, decision, custom"},
					"description": map[string]interface{}{"type": "string", "description": "Pack description"},
					"tags":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Tags"},
				},
				"required": []string{"name", "type"},
			},
		},
		{
			Name:        "get_context_pack",
			Description: "🟡 COMMON | Get detailed context pack info including linked memories, tasks, and contexts.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string", "description": "Context pack ID"}},
				"required":   []string{"packId"},
			},
		},
		{
			Name:        "update_context_pack",
			Description: "🟡 COMMON | Update a context pack's name, description, type, or status.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"packId":      map[string]interface{}{"type": "string", "description": "Context pack ID"},
					"name":        map[string]interface{}{"type": "string", "description": "New name"},
					"description": map[string]interface{}{"type": "string", "description": "New description"},
					"status":      map[string]interface{}{"type": "string", "description": "New status: draft, published"},
				},
				"required": []string{"packId"},
			},
		},
		{
			Name:        "delete_context_pack",
			Description: "🟢 ADVANCED | Delete a context pack. ⚠️ Requires explicit user approval.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string", "description": "Context pack ID"}},
				"required":   []string{"packId"},
			},
		},
		{
			Name:        "add_memory_to_pack",
			Description: "🟡 COMMON | Add an existing memory to a context pack.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"packId":   map[string]interface{}{"type": "string", "description": "Context pack ID"},
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID"},
				},
				"required": []string{"packId", "memoryId"},
			},
		},
		{
			Name:        "add_task_to_pack",
			Description: "🟡 COMMON | Add an existing task to a context pack.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"packId": map[string]interface{}{"type": "string", "description": "Context pack ID"},
					"taskId": map[string]interface{}{"type": "string", "description": "Task ID"},
				},
				"required": []string{"packId", "taskId"},
			},
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
			Description: "🟡 COMMON | Record an architectural decision (ADR). ⚠️ Agent creates DRAFTS only - user must approve. Use for important technical choices.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string", "description": "Decision title"}, "description": map[string]interface{}{"type": "string"}, "area": map[string]interface{}{"type": "string", "description": "Frontend, Backend, Architecture, etc."}, "context": map[string]interface{}{"type": "string", "description": "Why this decision?"}, "consequences": map[string]interface{}{"type": "string", "description": "What are the impacts?"}}, "required": []string{"title"}},
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
			Description: "🔴 ESSENTIAL | Get task statistics and completion rates. REQUIRED: project parameter.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string", "description": "Project name or ID (REQUIRED)"}}, "required": []string{"project"}},
		},

		// ============================================================================
		// 🟢 ADVANCED - Less frequently used
		// ============================================================================
		{
			Name:        "create_project",
			Description: "🟢 ADVANCED | Create a new project. ⚠️ Auto-checks for duplicates - returns similar projects if found. Use force=true to bypass.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "Project name - auto-checked for duplicates"}, "description": map[string]interface{}{"type": "string"}, "force": map[string]interface{}{"type": "boolean", "description": "Set true to bypass duplicate check and force creation"}, "org_id": map[string]interface{}{"type": "string", "description": "Organization ID to scope project to"}}, "required": []string{"name"}},
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
		{
			Name:        "move_task",
			Description: "🟡 COMMON | Move a task to a different project. Use this to fix tasks created in the wrong location.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"taskId":    map[string]interface{}{"type": "string", "description": "ID of the task to move"},
					"projectId": map[string]interface{}{"type": "string", "description": "Target project ID or name"},
				},
				"required": []string{"taskId", "projectId"},
			},
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

// ============================================================================
