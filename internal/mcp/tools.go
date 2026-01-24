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

	// Use scope-aware decryption
	plaintext, err := crypto.DecryptContentWithScope(
		m.EncryptedContent, m.ContentNonce,
		m.EncryptionScope, m.EncryptionOrgID, true,
	)
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

	// Decrypt title using scope-aware decryption
	if t.EncryptedTitle != "" {
		decrypted, err := crypto.DecryptContentWithScope(
			t.EncryptedTitle, t.TitleNonce,
			t.EncryptionScope, t.EncryptionOrgID, true,
		)
		if err != nil {
			title = "[Decryption Failed]"
		} else {
			title = decrypted
		}
	} else {
		title = t.Title
	}

	// Decrypt description using scope-aware decryption
	if t.EncryptedDescription != "" {
		decrypted, err := crypto.DecryptContentWithScope(
			t.EncryptedDescription, t.DescriptionNonce,
			t.EncryptionScope, t.EncryptionOrgID, true,
		)
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

// boolPtr returns a pointer to a bool value (for optional ToolAnnotation fields)
func boolPtr(b bool) *bool {
	return &b
}

// registerTools registers all MCP tools with the server using go-sdk
// The SDK automatically infers InputSchema from the handler's input struct type
// v3: Simplified from 61 tools to 26 tools via removal and consolidation
func registerTools(server *mcp.Server) {
	// ============================================================================
	// 🔴 ESSENTIAL (7 tools)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "setup_agent",
		Description: "🔴 ESSENTIAL | Initialize agent session. ⚠️ CALL THIS FIRST! Provide your agent name and model for tracking. Returns current context, pending tasks, recommended actions, agent_directives, and system info.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Initialize Agent Session",
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, handleSetupAgent)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "🔴 ESSENTIAL | List all projects.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Projects",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListProjects)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "🔴 ESSENTIAL | List, search, or get prioritized tasks. REQUIRED: project. Optional: status, query (keyword search), next_priority (bool, returns top TODO tasks sorted by priority), limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Tasks",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListTasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_task",
		Description: "🔴 ESSENTIAL | Create a new task. REQUIRED: project, description. Optional: priority (L/M/H).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Task",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_memory",
		Description: "🔴 ESSENTIAL | Store knowledge. REQUIRED: project, content. Optional: type (general, decision, bug_fix, preference, pattern, reference, skill - use 'skill' for procedural knowledge: reusable procedures, workflows, and step-by-step patterns), force (bypass similarity check).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Add Memory",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleAddMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall",
		Description: "🔴 ESSENTIAL | Search memories. REQUIRED: term. Supports OR (space-separated) and AND (comma-separated) search. Optional: project, tag, type (filter by type e.g. 'skill' for learned procedures), min_score, limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Search Memories",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleRecall)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_focus",
		Description: "🔴 ESSENTIAL | Get, set, or clear active workspace focus. No params = get current focus. With pack_id = set focus. With clear=true = clear focus.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Manage Focus",
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, handleManageFocus)

	// ============================================================================
	// 🟡 COMMON (12 tools)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task",
		Description: "🟡 COMMON | Get full task details including notes and metadata. REQUIRED: taskId.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Task Details",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_task",
		Description: "🟡 COMMON | Start, complete, stop, or update task progress. REQUIRED: taskId, action (start|complete|stop|progress). For progress action: also requires progress (0-100).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Manage Task Status",
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, handleManageTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_task_note",
		Description: "🟡 COMMON | Add a note to a task. REQUIRED: taskId, note.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Add Task Note",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleAddTaskNote)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_memories",
		Description: "🟡 COMMON | List memories with filtering. REQUIRED: project. Optional: term, limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Memories",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListMemories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_memory",
		Description: "🟡 COMMON | Get memory details by ID. REQUIRED: memoryId.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Memory",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_context_packs",
		Description: "🟡 COMMON | List context packs. Optional: type, status, query, limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Context Packs",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListContextPacks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_context_pack",
		Description: "🟡 COMMON | Get context pack details with linked memories, tasks, and contexts. REQUIRED: packId.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Context Pack",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_context_pack",
		Description: "🟡 COMMON | Create, update, or link items to a context pack. REQUIRED: action (create|update|add_memory|add_task). For create: name, type required. For update: packId required. For add_memory/add_task: packId and memoryId/taskId required.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Manage Context Pack",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleManageContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_decision",
		Description: "🟡 COMMON | Record an architectural decision (ADR). REQUIRED: title. Optional: description, status, area, context, consequences.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Decision Record",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateDecision)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_decisions",
		Description: "🟡 COMMON | List architectural decisions. Optional: status, area, limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Decisions",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListDecisions)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stats",
		Description: "🟡 COMMON | Get task statistics and completion rates. REQUIRED: project.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Statistics",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetStats)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_agent_activity",
		Description: "🟡 COMMON | Get recent agent activity timeline. Optional: project, agent_name, event_type, limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Agent Activity",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetAgentActivity)

	// ============================================================================
	// 🟢 ADVANCED (7 tools)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_project",
		Description: "🟢 ADVANCED | Create a new project. REQUIRED: name, description. Optional: force (bypass duplicate check), org_id.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Project",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateProject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_task",
		Description: "🟢 ADVANCED | Move a task to a different project. REQUIRED: taskId, projectId.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Move Task",
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, handleMoveTask)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_subtasks",
		Description: "🟢 ADVANCED | CRUD for subtasks. REQUIRED: action (create|list|complete|update), task_id. For create: description required. For complete/update: subtask_id required.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Manage Subtasks",
			OpenWorldHint: boolPtr(false),
		},
	}, handleManageSubtasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_dependencies",
		Description: "🟢 ADVANCED | Manage task dependencies. REQUIRED: action (add|list|remove), task_id. For add/remove: depends_on_id required. For list: direction (deps|dependents, default: deps).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Manage Dependencies",
			OpenWorldHint: boolPtr(false),
		},
	}, handleManageDependencies)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_plan",
		Description: "🟢 ADVANCED | Multi-agent planning. REQUIRED: action (create|status|list|apply|cancel). For create: requirements required. For status/apply/cancel: plan_id required.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Manage Plan",
			OpenWorldHint: boolPtr(false),
		},
	}, handleManagePlan)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_organizations",
		Description: "🟢 ADVANCED | List user's organizations.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Organizations",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListOrganizations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "switch_organization",
		Description: "🟢 ADVANCED | Switch active organization. REQUIRED: orgId.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Switch Organization",
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, handleSwitchOrganization)
}

// ============================================================================
// PAGINATION HELPERS
// ============================================================================

// decodeCursor decodes a cursor string to an offset. Empty cursor returns 0.
func decodeCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	var offset int
	if _, err := fmt.Sscanf(cursor, "%d", &offset); err != nil || offset < 0 {
		return 0
	}
	return offset
}

// paginateSlice applies cursor-based pagination to a slice and returns
// the paginated items, nextCursor (empty if no more), and total count.
func paginateSlice[T any](items []T, cursor string, limit int) ([]T, string, int) {
	total := len(items)
	offset := decodeCursor(cursor)

	if limit <= 0 {
		limit = 20
	}

	if offset >= total {
		return []T{}, "", total
	}

	end := offset + limit
	hasMore := false
	if end >= total {
		end = total
	} else {
		hasMore = true
	}

	paginated := items[offset:end]
	nextCursor := ""
	if hasMore {
		nextCursor = fmt.Sprintf("%d", end)
	}

	return paginated, nextCursor, total
}

// ============================================================================
// TOOL HANDLER FUNCTIONS
// ============================================================================

type EmptyInput struct{}


type SetupAgentInput struct {
	AgentName  string `json:"agent_name,omitempty"`
	AgentModel string `json:"agent_model,omitempty"`
}

func handleSetupAgent(ctx context.Context, req *mcp.CallToolRequest, input SetupAgentInput) (*mcp.CallToolResult, any, error) {
	// Initialize session with agent info
	agentName := strings.TrimSpace(input.AgentName)
	if agentName == "" {
		agentName = "unknown-agent"
	}
	agentModel := strings.TrimSpace(input.AgentModel)

	// Initialize the session
	session := InitializeSession(agentName, agentModel)

	// Set agent info on API client so ALL subsequent requests include agent headers
	// This enables proper event tracking in the backend timeline
	apiClient.SetAgentInfo(session.AgentName, session.AgentModel, session.ID)

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
		"step_1": "✅ setup_agent called - session initialized",
		"step_2": "📋 Call list_projects to see available projects",
		"step_3": "📝 Specify 'project' parameter in create_task/add_memory calls",
		"note":   "Always pass the 'project' parameter explicitly when creating tasks or memories",
	}

	// Add agent directives for proactive behavior
	result["agent_directives"] = GetDirectivesAsMap()

	return mustTextResult(result), nil, nil
}

func handleListProjects(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
	// Pass active org ID from session to get organization-scoped projects
	var orgID string
	session := GetCurrentSession()
	if session != nil && session.ActiveOrgID != nil {
		orgID = session.ActiveOrgID.String()
	}

	projects, err := apiClient.ListProjects(orgID)
	if err != nil {
		return nil, nil, err
	}
	// Wrap array response to fix "expected record, received array" error
	return mustTextResult(formatMCPResponse(projects, getContextString())), nil, nil
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
				return mustTextResult(map[string]interface{}{
					"status":        "exists",
					"exact_match":   suggestions.ExactMatch,
					"_message":      "✅ Project already exists with this name. Use existing project instead of creating a duplicate.",
					"_action":       "Specify this project name in your create_task/add_memory calls.",
				}), nil, nil
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
					return mustTextResult(map[string]interface{}{
						"status":           "needs_confirmation",
						"similar_projects": similarList,
						"requested_name":   name,
						"_message":         "⚠️ Similar projects found. To avoid duplicates:\n1. Use an existing project from the list above, OR\n2. Call create_project with force=true to create anyway",
						"_action":          "Either specify 'project' parameter with an existing project name, or call create_project(name=\"...\", force=true)",
					}), nil, nil
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
	return mustTextResult(map[string]interface{}{
		"project":  project,
		"_message": "✅ Project created successfully: " + project.Name,
	}), nil, nil
}

type ListTasksInput struct {
	Status       string  `json:"status,omitempty"`
	Project      string  `json:"project"`                 // REQUIRED - project name or ID
	Query        string  `json:"query,omitempty"`          // Optional keyword search
	NextPriority bool    `json:"next_priority,omitempty"`  // If true, return top TODO tasks by priority
	Limit        float64 `json:"limit,omitempty"`
	Cursor       string  `json:"cursor,omitempty"`         // Pagination cursor from previous response
}

func handleListTasks(ctx context.Context, req *mcp.CallToolRequest, input ListTasksInput) (*mcp.CallToolResult, any, error) {
	// REQUIRED: project parameter must be specified
	if strings.TrimSpace(input.Project) == "" {
		return nil, nil, errors.New("'project' parameter is REQUIRED. Use list_projects to see available projects.")
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	// Determine which query mode to use
	query := strings.TrimSpace(input.Query)
	status := strings.TrimSpace(input.Status)

	var tasks []models.Task
	if input.NextPriority {
		// Priority mode: get TODO tasks sorted by priority
		if status == "" {
			status = "TODO"
		}
		tasks, err = apiClient.ListTasksQuery(projectID, status, query, nil, nil)
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
	} else if query != "" {
		// Search mode: keyword search across tasks
		tasks, err = apiClient.ListTasksQuery(projectID, status, query, nil, nil)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Standard list mode
		tasks, err = apiClient.ListTasks(projectID, status)
		if err != nil {
			return nil, nil, err
		}
	}

	limit := int(input.Limit)
	if input.NextPriority && limit <= 0 {
		limit = 5 // Default limit for priority mode
	}
	if limit <= 0 {
		limit = 20
	}

	// Apply cursor-based pagination
	paginatedTasks, nextCursor, total := paginateSlice(tasks, input.Cursor, limit)

	// Decrypt task fields before returning
	var decryptedTasks []map[string]interface{}
	for _, t := range paginatedTasks {
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

	return mustTextResult(formatPaginatedResponse(decryptedTasks, nextCursor, total, getContextString())), nil, nil
}

type CreateTaskInput struct {
	Description     string `json:"description"`
	Priority        string `json:"priority,omitempty"`
	Project         string `json:"project"`                    // REQUIRED - project name or ID
	EncryptionScope string `json:"encryption_scope,omitempty"` // "personal" or "organization" (auto-detected if not provided)
}

func handleCreateTask(ctx context.Context, req *mcp.CallToolRequest, input CreateTaskInput) (*mcp.CallToolResult, any, error) {
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
	projectID, orgID, err := resolveProjectWithOrg(apiClient, input.Project)
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

	// Determine encryption scope
	scope := strings.TrimSpace(input.EncryptionScope)
	if scope == "" {
		// Auto-detect: use org encryption if org vault is unlocked
		scope = determineEncryptionScope(orgID)
	}

	// Check if user has encryption enabled and vault is unlocked
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled {
		if scope == "organization" && orgID != "" {
			// Organization-scoped encryption
			if !crypto.IsOrgVaultUnlocked(orgID) {
				return nil, nil, errors.New("🔒 Organization vault is locked.\n\n" +
					"To unlock, the user must run:\n" +
					"  ramorie org unlock " + orgID[:8] + "\n\n" +
					"This only needs to be done once per session.\n" +
					"Please inform the user to unlock their org vault.")
			}

			encryptedDesc, nonce, isEncrypted, err := crypto.EncryptContentWithScope(description, "organization", orgID)
			if err != nil {
				return nil, nil, fmt.Errorf("encryption failed: %w", err)
			}

			if isEncrypted {
				task, err := apiClient.CreateEncryptedTaskWithMeta(projectID, encryptedDesc, nonce, priority, meta)
				if err != nil {
					return nil, nil, err
				}

				result := formatMCPResponse(task, getContextString())
				result["_created_in_project"] = projectID
				result["_encrypted"] = true
				result["_encryption_scope"] = "organization"
				if session != nil {
					result["_created_by_agent"] = session.AgentName
				}
				result["_message"] = "✅ Task created (org-encrypted) in project " + projectID[:8] + "..."
				return mustTextResult(result), nil, nil
			}
		} else {
			// Personal-scoped encryption
			if !crypto.IsVaultUnlocked() {
				return nil, nil, errors.New("🔒 Vault is locked. Your account has encryption enabled.\n\n" +
					"To unlock, the user must run:\n" +
					"  ramorie setup unlock\n\n" +
					"This only needs to be done once per session (until computer restarts).\n" +
					"Please inform the user to unlock their vault.")
			}

			encryptedDesc, nonce, isEncrypted, err := crypto.EncryptContent(description)
			if err != nil {
				return nil, nil, fmt.Errorf("encryption failed: %w", err)
			}

			if isEncrypted {
				task, err := apiClient.CreateEncryptedTaskWithMeta(projectID, encryptedDesc, nonce, priority, meta)
				if err != nil {
					return nil, nil, err
				}

				result := formatMCPResponse(task, getContextString())
				result["_created_in_project"] = projectID
				result["_encrypted"] = true
				result["_encryption_scope"] = "personal"
				if session != nil {
					result["_created_by_agent"] = session.AgentName
				}
				result["_message"] = "✅ Task created (encrypted) in project " + projectID[:8] + "..."
				return mustTextResult(result), nil, nil
			}
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

	return mustTextResult(result), nil, nil
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

	return mustTextResult(result), nil, nil
}


type MoveTaskInput struct {
	TaskID    string `json:"taskId"`
	ProjectID string `json:"projectId"`
}

func handleMoveTask(ctx context.Context, req *mcp.CallToolRequest, input MoveTaskInput) (*mcp.CallToolResult, any, error) {
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

	return mustTextResult(result), nil, nil
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
			return mustTextResult(annotation), nil, nil
		}
	}

	annotation, err := apiClient.CreateAnnotation(taskID, note)
	if err != nil {
		return nil, nil, err
	}
	return mustTextResult(annotation), nil, nil
}

type AddMemoryInput struct {
	Content         string `json:"content"`
	Project         string `json:"project"`                    // REQUIRED - project name or ID
	Type            string `json:"type,omitempty"`             // Memory type: general, decision, bug_fix, preference, pattern, reference (auto-detected if not provided)
	Force           bool   `json:"force,omitempty"`            // Skip similarity check and force creation
	EncryptionScope string `json:"encryption_scope,omitempty"` // "personal" or "organization" (auto-detected if not provided)
}

func handleAddMemory(ctx context.Context, req *mcp.CallToolRequest, input AddMemoryInput) (*mcp.CallToolResult, any, error) {
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

	projectID, orgID, err := resolveProjectWithOrg(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	// Check for similar memories unless force=true
	if !input.Force {
		// Fetch existing memories for this project
		existingMemories, err := apiClient.ListMemories(projectID, "")
		if err == nil && len(existingMemories) > 0 {
			// Check for similar content
			similarMemories := CheckSimilarMemories(existingMemories, content, SimilarityThreshold)
			if len(similarMemories) > 0 {
				// Return warning with similar memories
				result := map[string]interface{}{
					"status":           "similar_exists",
					"message":          "Similar memories already exist. Use force=true to create anyway.",
					"similar_count":    len(similarMemories),
					"similar_memories": similarMemories,
					"threshold":        SimilarityThreshold,
					"_context":         getContextString(),
				}
				return mustTextResult(result), nil, nil
			}
		}
	}

	// Determine encryption scope
	scope := strings.TrimSpace(input.EncryptionScope)
	if scope == "" {
		scope = determineEncryptionScope(orgID)
	}

	// Check if user has encryption enabled and vault is unlocked
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled {
		if scope == "organization" && orgID != "" {
			// Organization-scoped encryption
			if !crypto.IsOrgVaultUnlocked(orgID) {
				return nil, nil, errors.New("🔒 Organization vault is locked.\n\n" +
					"To unlock, the user must run:\n" +
					"  ramorie org unlock " + orgID[:8] + "\n\n" +
					"This only needs to be done once per session.\n" +
					"Please inform the user to unlock their org vault.")
			}

			encryptedContent, nonce, isEncrypted, err := crypto.EncryptContentWithScope(content, "organization", orgID)
			if err != nil {
				return nil, nil, fmt.Errorf("encryption failed: %w", err)
			}

			if isEncrypted {
				memory, err := apiClient.CreateEncryptedMemory(projectID, encryptedContent, nonce)
				if err != nil {
					return nil, nil, err
				}

				result := formatMCPResponse(memory, getContextString())
				result["_created_in_project"] = projectID
				result["_encrypted"] = true
				result["_encryption_scope"] = "organization"
				result["_message"] = "✅ Memory created (org-encrypted) in project " + projectID[:8] + "..."
				return mustTextResult(result), nil, nil
			}
		} else {
			// Personal-scoped encryption
			if !crypto.IsVaultUnlocked() {
				return nil, nil, errors.New("🔒 Vault is locked. Your account has encryption enabled.\n\n" +
					"To unlock, the user must run:\n" +
					"  ramorie setup unlock\n\n" +
					"This only needs to be done once per session (until computer restarts).\n" +
					"Please inform the user to unlock their vault.")
			}

			encryptedContent, nonce, isEncrypted, err := crypto.EncryptContent(content)
			if err != nil {
				return nil, nil, fmt.Errorf("encryption failed: %w", err)
			}

			if isEncrypted {
				memory, err := apiClient.CreateEncryptedMemory(projectID, encryptedContent, nonce)
				if err != nil {
					return nil, nil, err
				}

				result := formatMCPResponse(memory, getContextString())
				result["_created_in_project"] = projectID
				result["_encrypted"] = true
				result["_encryption_scope"] = "personal"
				result["_message"] = "✅ Memory created (encrypted) in project " + projectID[:8] + "..."
				return mustTextResult(result), nil, nil
			}
		}
	}

	// Determine memory type - use provided or auto-detect
	memoryType := strings.TrimSpace(input.Type)
	if memoryType == "" {
		memoryType = DetectMemoryType(content)
	} else if !IsValidMemoryType(memoryType) {
		return nil, nil, fmt.Errorf("invalid memory type '%s'. Valid types: %v", memoryType, ValidMemoryTypes())
	}

	// Non-encrypted memory creation (user doesn't have encryption enabled)
	memory, err := apiClient.CreateMemoryWithType(projectID, content, memoryType)
	if err != nil {
		return nil, nil, err
	}

	// Return with context info showing where memory was created
	result := formatMCPResponse(memory, getContextString())
	result["_created_in_project"] = projectID
	result["_type"] = memoryType
	result["_type_auto_detected"] = input.Type == ""
	result["_message"] = "✅ Memory created successfully in project " + projectID[:8] + "..."

	return mustTextResult(result), nil, nil
}

type ListMemoriesInput struct {
	Project string  `json:"project"` // REQUIRED - project name or ID
	Term    string  `json:"term,omitempty"`
	Limit   float64 `json:"limit,omitempty"`
	Cursor  string  `json:"cursor,omitempty"` // Pagination cursor
}

func handleListMemories(ctx context.Context, req *mcp.CallToolRequest, input ListMemoriesInput) (*mcp.CallToolResult, any, error) {
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
	if limit <= 0 {
		limit = 20
	}

	// Apply cursor-based pagination
	paginatedMemories, nextCursor, total := paginateSlice(memories, input.Cursor, limit)

	// Decrypt all memory content before returning
	var decryptedMemories []map[string]interface{}
	for _, m := range paginatedMemories {
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

	return mustTextResult(formatPaginatedResponse(decryptedMemories, nextCursor, total, getContextString())), nil, nil
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

	return mustTextResult(result), nil, nil
}

type RecallInput struct {
	Term             string  `json:"term"`
	Project          string  `json:"project,omitempty"`
	Tag              string  `json:"tag,omitempty"`
	Type             string  `json:"type,omitempty"` // Filter by memory type: general, decision, bug_fix, preference, pattern, reference
	LinkedTask       bool    `json:"linked_task,omitempty"`
	IncludeRelations bool    `json:"include_relations,omitempty"`
	Limit            float64 `json:"limit,omitempty"`
	MinScore         float64 `json:"min_score,omitempty"`
	Cursor           string  `json:"cursor,omitempty"` // Pagination cursor
}

func handleRecall(ctx context.Context, req *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, any, error) {
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
			"id":           m.ID.String(),
			"content":      decryptedContent, // Use decrypted content in result
			"score":        score,
			"access_count": m.AccessCount,
			"created_at":   m.CreatedAt,
		}
		if m.Importance != nil {
			result["importance"] = *m.Importance
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

	// Apply cursor-based pagination to scored results
	paginatedScored, nextCursor, totalFound := paginateSlice(scored, input.Cursor, limit)

	var results []interface{}
	for _, s := range paginatedScored {
		results = append(results, s.memory)
	}

	response := map[string]interface{}{
		"term":        term,
		"search_mode": map[bool]string{true: "AND", false: "OR"}[isAndSearch],
		"count":       len(results),
		"total_found": totalFound,
		"results":     results,
	}
	if nextCursor != "" {
		response["nextCursor"] = nextCursor
	}

	return mustTextResult(response), nil, nil
}

type ListContextPacksInput struct {
	Type   string  `json:"type,omitempty"`   // project, integration, decision, custom
	Status string  `json:"status,omitempty"` // draft, published
	Query  string  `json:"query,omitempty"`
	Limit  float64 `json:"limit,omitempty"`
	Cursor string  `json:"cursor,omitempty"` // Pagination cursor
}

func handleListContextPacks(ctx context.Context, req *mcp.CallToolRequest, input ListContextPacksInput) (*mcp.CallToolResult, any, error) {
	limit := int(input.Limit)
	if limit <= 0 {
		limit = 20
	}
	offset := decodeCursor(input.Cursor)
	result, err := apiClient.ListContextPacks(
		strings.TrimSpace(input.Type),
		strings.TrimSpace(input.Status),
		strings.TrimSpace(input.Query),
		limit+1, // Fetch one extra to detect if there are more
		offset,
	)
	if err != nil {
		return nil, nil, err
	}

	// The backend was asked for limit+1 items to detect if there are more pages
	packs := result.ContextPacks
	nextCursor := ""
	if len(packs) > limit {
		packs = packs[:limit]
		nextCursor = fmt.Sprintf("%d", offset+limit)
	}

	total := int(result.Total)
	if total == 0 {
		total = offset + len(packs)
		if nextCursor != "" {
			total++
		}
	}

	return mustTextResult(formatPaginatedResponse(packs, nextCursor, total, getContextString())), nil, nil
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
	return mustTextResult(pack), nil, nil
}



// ============================================================================
// ORGANIZATION HANDLERS
// ============================================================================

func handleListOrganizations(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
	orgs, err := apiClient.ListOrganizations()
	if err != nil {
		return nil, nil, err
	}
	// Wrap array response to fix "expected record, received array" error
	return mustTextResult(formatMCPResponse(orgs, getContextString())), nil, nil
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

	return mustTextResult(org), nil, nil
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
	return mustTextResult(decision), nil, nil
}

type ListDecisionsInput struct {
	Status string  `json:"status,omitempty"`
	Area   string  `json:"area,omitempty"`
	Limit  float64 `json:"limit,omitempty"`
	Cursor string  `json:"cursor,omitempty"` // Pagination cursor
}

func handleListDecisions(ctx context.Context, req *mcp.CallToolRequest, input ListDecisionsInput) (*mcp.CallToolResult, any, error) {
	// Fetch all matching decisions from API
	decisions, err := apiClient.ListDecisions(strings.TrimSpace(input.Status), strings.TrimSpace(input.Area), 0)
	if err != nil {
		return nil, nil, err
	}

	// Apply cursor-based pagination
	limit := int(input.Limit)
	if limit <= 0 {
		limit = 20
	}

	paginated, nextCursor, total := paginateSlice(decisions, input.Cursor, limit)
	return mustTextResult(formatPaginatedResponse(paginated, nextCursor, total, getContextString())), nil, nil
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
	return mustTextResult(out), nil, nil
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
		// 🔴 ESSENTIAL (7 tools)
		// ============================================================================
		{
			Name:        "setup_agent",
			Description: "🔴 ESSENTIAL | Initialize agent session. ⚠️ CALL THIS FIRST! Provide your agent name and model for tracking.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_name":  map[string]interface{}{"type": "string", "description": "Your agent identifier (e.g., 'claude-code', 'cursor', 'gemini')"},
					"agent_model": map[string]interface{}{"type": "string", "description": "Model being used (e.g., 'claude-opus-4-5-20250514', 'gpt-4')"},
				},
			},
		},
		{Name: "list_projects", Description: "🔴 ESSENTIAL | List all projects."},
		{Name: "list_tasks", Description: "🔴 ESSENTIAL | List/search/prioritize tasks. Supports query, next_priority params."},
		{Name: "create_task", Description: "🔴 ESSENTIAL | Create a new task."},
		{Name: "add_memory", Description: "🔴 ESSENTIAL | Store knowledge (use type='skill' for procedural memory)."},
		{Name: "recall", Description: "🔴 ESSENTIAL | Search memories (filter type='skill' for learned procedures)."},
		{Name: "manage_focus", Description: "🔴 ESSENTIAL | Get/set/clear active workspace focus."},
		// ============================================================================
		// 🟡 COMMON (12 tools)
		// ============================================================================
		{Name: "get_task", Description: "🟡 COMMON | Get task details including notes and metadata."},
		{Name: "manage_task", Description: "🟡 COMMON | Start/complete/stop/update progress on a task. Actions: start, complete, stop, progress."},
		{Name: "add_task_note", Description: "🟡 COMMON | Add a note/annotation to a task."},
		{Name: "list_memories", Description: "🟡 COMMON | List memories with filtering."},
		{Name: "get_memory", Description: "🟡 COMMON | Get memory details by ID."},
		{Name: "list_context_packs", Description: "🟡 COMMON | List all context packs."},
		{Name: "get_context_pack", Description: "🟡 COMMON | Get detailed context pack info."},
		{Name: "manage_context_pack", Description: "🟡 COMMON | Create/update/add_memory/add_task to context packs."},
		{Name: "create_decision", Description: "🟡 COMMON | Record an architectural decision (ADR)."},
		{Name: "list_decisions", Description: "🟡 COMMON | List architectural decisions."},
		{Name: "get_stats", Description: "🟡 COMMON | Get task statistics and completion rates."},
		{Name: "get_agent_activity", Description: "🟡 COMMON | Get recent agent activity timeline."},
		// ============================================================================
		// 🟢 ADVANCED (7 tools)
		// ============================================================================
		{Name: "create_project", Description: "🟢 ADVANCED | Create a new project."},
		{Name: "move_task", Description: "🟢 ADVANCED | Move a task to a different project."},
		{Name: "manage_subtasks", Description: "🟢 ADVANCED | Create/list/complete/update subtasks."},
		{Name: "manage_dependencies", Description: "🟢 ADVANCED | Add/list/remove task dependencies."},
		{Name: "manage_plan", Description: "🟢 ADVANCED | Create/status/list/apply/cancel multi-agent plans."},
		{Name: "list_organizations", Description: "🟢 ADVANCED | List user's organizations."},
		{Name: "switch_organization", Description: "🟢 ADVANCED | Switch active organization context."},
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

func getActiveOrgIDString() string {
	if activeOrg := GetSessionActiveOrgID(); activeOrg != nil {
		return activeOrg.String()
	}
	return ""
}

func resolveProjectID(client *api.Client, projectIdentifier string) (string, error) {
	projectIdentifier = strings.TrimSpace(projectIdentifier)
	if projectIdentifier == "" {
		return "", errors.New("project parameter is required")
	}

	projects, err := client.ListProjects(getActiveOrgIDString())
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

// resolveProjectWithOrg resolves project ID and returns the org ID if the project belongs to an org
func resolveProjectWithOrg(client *api.Client, projectIdentifier string) (projectID string, orgID string, err error) {
	projectIdentifier = strings.TrimSpace(projectIdentifier)
	if projectIdentifier == "" {
		return "", "", errors.New("project parameter is required")
	}

	projects, err := client.ListProjects(getActiveOrgIDString())
	if err != nil {
		return "", "", err
	}
	for _, p := range projects {
		if p.Name == projectIdentifier || strings.HasPrefix(p.ID.String(), projectIdentifier) {
			pid := p.ID.String()
			oid := ""
			if p.OrganizationID != nil {
				oid = p.OrganizationID.String()
			}
			return pid, oid, nil
		}
	}

	return "", "", errors.New("project not found")
}

// determineEncryptionScope decides the encryption scope for a project.
// Returns "organization" if the project belongs to an org with encryption enabled and the org vault is unlocked.
// Otherwise returns "personal".
func determineEncryptionScope(orgID string) string {
	if orgID != "" && crypto.IsOrgVaultUnlocked(orgID) {
		return "organization"
	}
	return "personal"
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



func setupAgent(client *api.Client) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"status":  "ready",
		"message": "🧠 Ramorie agent session initialized",
		"version": "3.19.0",
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

	// List projects (include org-scoped if active)
	projects, err := client.ListProjects(getActiveOrgIDString())
	if err == nil {
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

	// Get stats
	statsBytes, err := client.Request("GET", "/reports/stats", nil)
	if err == nil {
		var stats map[string]interface{}
		if json.Unmarshal(statsBytes, &stats) == nil {
			result["stats"] = stats
		}
	}

	// Context injection: provide recent memories, in-progress tasks, and last session events
	contextInjection := map[string]interface{}{}

	// Recent memories (max 5, truncated to 200 chars)
	if memories, err := client.ListMemories("", ""); err == nil && len(memories) > 0 {
		limit := 5
		if len(memories) < limit {
			limit = len(memories)
		}
		truncated := make([]map[string]interface{}, 0, limit)
		for i := 0; i < limit; i++ {
			m := memories[i]
			content := m.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			entry := map[string]interface{}{
				"id":         m.ID.String(),
				"content":    content,
				"created_at": m.CreatedAt.Format("2006-01-02 15:04"),
			}
			if m.Project != nil {
				entry["project"] = m.Project.Name
			}
			truncated = append(truncated, entry)
		}
		contextInjection["recent_memories"] = truncated
	}

	// In-progress tasks (max 3, title + description truncated)
	if tasks, err := client.ListTasks("", "IN_PROGRESS"); err == nil && len(tasks) > 0 {
		limit := 3
		if len(tasks) < limit {
			limit = len(tasks)
		}
		truncated := make([]map[string]interface{}, 0, limit)
		for i := 0; i < limit; i++ {
			t := tasks[i]
			desc := t.Description
			if len(desc) > 150 {
				desc = desc[:150] + "..."
			}
			entry := map[string]interface{}{
				"id":     t.ID.String(),
				"title":  t.Title,
				"status": t.Status,
			}
			if desc != "" {
				entry["description"] = desc
			}
			if t.Project != nil {
				entry["project"] = t.Project.Name
			}
			truncated = append(truncated, entry)
		}
		contextInjection["in_progress_tasks"] = truncated
	}

	// Last session events (max 5)
	if events, err := client.GetAgentEvents(api.AgentEventFilter{Limit: 5}); err == nil && len(events.Events) > 0 {
		sessionEvents := make([]map[string]interface{}, 0, len(events.Events))
		for _, e := range events.Events {
			entry := map[string]interface{}{
				"event_type":  e.EventType,
				"entity_type": e.EntityType,
				"created_at":  e.CreatedAt,
			}
			if e.EntityTitle != nil {
				entry["title"] = *e.EntityTitle
			}
			if e.EntityPreview != nil {
				preview := *e.EntityPreview
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				entry["preview"] = preview
			}
			if e.AgentName != "" {
				entry["agent"] = e.AgentName
			}
			sessionEvents = append(sessionEvents, entry)
		}
		contextInjection["last_session"] = sessionEvents
	}

	if len(contextInjection) > 0 {
		result["context_injection"] = contextInjection
	}

	// Recommendations
	recommendations := []string{}
	if result["active_focus"] == nil {
		recommendations = append(recommendations, "💡 Set an active focus: manage_focus with pack_id (for workspace context)")
	}
	if result["active_task"] == nil {
		recommendations = append(recommendations, "💡 Start a task for memory auto-linking: manage_task with action=start")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✅ Ready to work! Use list_tasks with next_priority=true to see priorities")
	}
	result["next_steps"] = recommendations

	return result, nil
}

// ============================================================================
// Agent Activity Timeline Handlers
// ============================================================================

type GetAgentActivityInput struct {
	Project   string  `json:"project,omitempty"`   // Optional: filter by project name or ID
	AgentName string  `json:"agent_name,omitempty"` // Optional: filter by agent name
	EventType string  `json:"event_type,omitempty"` // Optional: filter by event type (memory_created, task_created, etc.)
	Limit     float64 `json:"limit,omitempty"`      // Optional: max results (default 20, max 50)
	Cursor    string  `json:"cursor,omitempty"`     // Pagination cursor (offset)
}

func handleGetAgentActivity(ctx context.Context, req *mcp.CallToolRequest, input GetAgentActivityInput) (*mcp.CallToolResult, any, error) {
	// Build filter
	filter := api.AgentEventFilter{
		AgentName: strings.TrimSpace(input.AgentName),
		EventType: strings.TrimSpace(input.EventType),
	}

	// Set limit
	limit := int(input.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	filter.Limit = limit

	// Resolve project ID if provided
	if project := strings.TrimSpace(input.Project); project != "" {
		projectID, err := resolveProjectID(apiClient, project)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve project: %w", err)
		}
		filter.ProjectID = projectID
	}

	// Get agent events
	response, err := apiClient.GetAgentEvents(filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get agent activity: %w", err)
	}

	// Format events for agent-friendly output
	events := make([]map[string]interface{}, 0, len(response.Events))
	for _, event := range response.Events {
		e := map[string]interface{}{
			"event_type":  event.EventType,
			"entity_type": event.EntityType,
			"agent_name":  event.AgentName,
			"created_at":  event.CreatedAt,
			"created_via": event.CreatedVia,
		}
		if event.EntityTitle != nil {
			e["entity_title"] = *event.EntityTitle
		}
		if event.ProjectName != nil {
			e["project_name"] = *event.ProjectName
		}
		if event.AgentModel != nil {
			e["agent_model"] = *event.AgentModel
		}
		events = append(events, e)
	}

	// Build paginated response
	nextCursor := ""
	if response.HasMore {
		nextCursor = fmt.Sprintf("%d", decodeCursor(input.Cursor)+len(events))
	}

	result := map[string]interface{}{
		"events":   events,
		"count":    len(events),
		"_context": getContextString(),
	}

	if response.Total > 0 {
		result["total"] = response.Total
	}
	if nextCursor != "" {
		result["nextCursor"] = nextCursor
	}

	return mustTextResult(result), nil, nil
}

