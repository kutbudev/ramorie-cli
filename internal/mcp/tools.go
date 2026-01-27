package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// emptyObjectSchema returns an explicit JSON schema for tools with no input parameters
// This fixes the "expected record, received array" error when the SDK infers an empty schema
func emptyObjectSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
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
		Description: "🔴 ESSENTIAL | List ALL accessible projects (personal + all organizations you're a member of). No org switch required.",
		InputSchema: emptyObjectSchema(),
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
		Name: "create_task",
		Description: `🔴 ESSENTIAL | Create a new task.

REQUIRED: project (name or ID), description
OPTIONAL: priority (L/M/H, default: M)

IMPORTANT:
- Call 'setup_agent' first if you get session errors
- Use 'list_projects' to see available project names
- Project names work across all your organizations - no switch required
- Project names are case-insensitive

Example: create_task(project: "my-project", description: "Implement login feature", priority: "H")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Task",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateTask)

	mcp.AddTool(server, &mcp.Tool{
		Name: "remember",
		Description: `🔴 ESSENTIAL | Ultra-simple memory storage.

Just tell me what to remember - I'll figure out the rest.

REQUIRED: content (what to remember)
OPTIONAL: project (auto-detected from last used if not provided)

The type is auto-detected from content:
- "decided X" → decision
- "fixed bug" → bug_fix
- "prefer X" → preference
- "todo: X" / "later: X" → auto-creates TASK instead of memory

Example: remember(content: "API uses JWT authentication")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Remember",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleRemember)

	mcp.AddTool(server, &mcp.Tool{
		Name: "recall",
		Description: `🔴 ESSENTIAL | Search memories with optional knowledge graph traversal.

REQUIRED: term
Supports OR (space-separated) and AND (comma-separated) search.

Optional:
- project: Scope to project
- tag: Filter by tag
- type: Filter by memory type (general, decision, bug_fix, preference, pattern, reference, skill)
- entity_hops: (0-3) Include memories from related entities via knowledge graph (default 0)
- min_score: Minimum relevance score
- limit: Max results
- valid_at: RFC3339 timestamp to query memories valid at that time
- include_expired: Include TTL-expired memories

When entity_hops > 0, recall will:
1. Find entities matching your search term
2. Traverse the knowledge graph by N hops
3. Include memories linked to discovered entities
4. Boost scores for directly matching memories

Example: recall(term: "React", entity_hops: 2) - finds React memories + memories about related tools`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Search Memories",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleRecall)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "surface_skills",
		Description: "🟡 COMMON | Find relevant procedural skills based on context. REQUIRED: context (describe current task/situation). Returns skills with matching triggers, steps, and validation. Use before starting new tasks to check if learned procedures apply.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Surface Relevant Skills",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleSurfaceSkills)

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
		Description: "🟡 COMMON | Create, update, or link items to a context pack. REQUIRED: action (create|update|link_memory|link_task). For create: name, type required. For update: packId required. For link_memory/link_task: packId and memoryId/taskId required.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Manage Context Pack",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleManageContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "export_context_pack",
		Description: "🟢 ADVANCED | Export a context pack as a portable JSON bundle with all linked memories, tasks, and contexts. REQUIRED: pack_id. Returns downloadable bundle for migration or backup.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Export Context Pack",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleExportContextPack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "import_context_pack",
		Description: "🟢 ADVANCED | Import a previously exported context pack bundle. REQUIRED: bundle (JSON), project (target project). Optional: conflict_mode (skip|overwrite|rename). Recreates context pack with all linked items.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Import Context Pack",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleImportContextPack)

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
	// 🟢 ADVANCED (9 tools)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name: "create_project",
		Description: `🟢 ADVANCED | Create a new project.

REQUIRED: name, description
OPTIONAL: force (bypass duplicate check), org_id (organization UUID)

IMPORTANT:
- If org_id is provided, you must be a member of that organization
- Use 'list_organizations' to see available organizations
- Project names must be unique per user (or per org if org_id provided)

Example: create_project(name: "my-project", description: "My awesome project")`,
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
		InputSchema: emptyObjectSchema(),
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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "consolidate_memories",
		Description: "🟢 ADVANCED | Trigger memory consolidation for a project. Scores memories by importance/access/recency, promotes high-value and archives stale ones. REQUIRED: project. Optional: stale_days (default 30), promote_threshold (0-1, default 0.8), archive_threshold (0-1, default 0.2).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Consolidate Memories",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleConsolidateMemories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "cleanup_memories",
		Description: "🟢 ADVANCED | Clean up expired memories (TTL-based). Removes memories past their TTL expiration. Optional: project (scope to project), dry_run (preview without deleting), batch_size (default 100).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Cleanup Expired Memories",
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCleanupMemories)

	// ============================================================================
	// 🟡 COMMON - Entity & Knowledge Graph Tools
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_entities",
		Description: `🟡 COMMON | List entities (knowledge graph nodes) with filtering.

Optional: type (person|tool|concept|project|organization|location|event|document|api|other), project, query, limit, offset.

Returns entities sorted by mention count (most mentioned first).

Example: list_entities(type: "tool", project: "Ramorie Frontend")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Entities",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListEntities)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_entity",
		Description: "🟡 COMMON | Get entity details by ID. REQUIRED: entity_id. Returns entity with full metadata including confidence, aliases, mention count.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Entity",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetEntity)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_entity_graph",
		Description: `🟡 COMMON | Get knowledge graph visualization data for an entity.

REQUIRED: entity_id
Optional: hops (1-3, default 1) - how many relationship hops to traverse

Returns nodes and edges for graph visualization, including the root entity and all connected entities within N hops.

Example: get_entity_graph(entity_id: "uuid", hops: 2)`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Entity Graph",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetEntityGraph)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_memory_entities",
		Description: `🟡 COMMON | Get entities extracted from a specific memory.

REQUIRED: memory_id

Returns all entities that were automatically extracted from the memory content during processing.`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Memory Entities",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetMemoryEntities)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_entity_memories",
		Description: `🟡 COMMON | Get memories that mention a specific entity.

REQUIRED: entity_id
Optional: hops (0-3, default 0) - include memories from related entities within N hops
Optional: limit (default 50)

Use hops > 0 to discover related context through the knowledge graph.

Example: get_entity_memories(entity_id: "uuid", hops: 2)`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Entity Memories",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetEntityMemories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_entity_stats",
		Description: "🟡 COMMON | Get knowledge graph statistics. Returns total entities, total relationships, and breakdown by entity type.",
		InputSchema: emptyObjectSchema(),
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Entity Stats",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetEntityStats)

	// ============================================================================
	// 🟢 ADVANCED - Entity Management Tools
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name: "create_entity",
		Description: `🟢 ADVANCED | Create a new entity (knowledge graph node) manually.

REQUIRED: name, type (person|tool|concept|project|organization|location|event|document|api|other)
Optional: description, aliases (array), project, confidence (0-1)

Use this when you want to explicitly add an entity rather than relying on auto-extraction.

Example: create_entity(name: "React", type: "tool", description: "JavaScript UI library")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Entity",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateEntity)

	mcp.AddTool(server, &mcp.Tool{
		Name: "create_relationship",
		Description: `🟢 ADVANCED | Create a relationship (knowledge graph edge) between two entities.

REQUIRED: source_entity_id, target_entity_id, relationship_type
Optional: label, description, strength (0-1, default 1)

Relationship types: uses, works_on, related_to, depends_on, part_of, created_by, belongs_to, connects_to, replaces, similar_to, contradicts, references, implements, extends

Example: create_relationship(source_entity_id: "uuid1", target_entity_id: "uuid2", relationship_type: "uses")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Relationship",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateRelationship)

	// ============================================================================
	// 🟡 COMMON - Skills Tools (Procedural Memory)
	// ============================================================================
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_skills",
		Description: `🟡 COMMON | List procedural skills (memories with type='skill').

Optional: project (filter by project name or ID), limit (default 50)

Returns skills with their triggers, steps, and validation criteria.

Example: list_skills(project: "Ramorie Frontend")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Skills",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListSkills)

	mcp.AddTool(server, &mcp.Tool{
		Name: "create_skill",
		Description: `🟡 COMMON | Create a new procedural skill (how-to knowledge).

REQUIRED: project (name or ID), trigger, description, steps (array of strings)
Optional: validation (how to verify success), tags (array)

A skill is a procedural memory that can be surfaced and executed by agents.

Example: create_skill(project: "Ramorie", trigger: "When deploying to production", description: "Production deployment procedure", steps: ["Run tests", "Build", "Deploy"])`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Skill",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateSkill)

	mcp.AddTool(server, &mcp.Tool{
		Name: "execute_skill",
		Description: `🟡 COMMON | Start tracking execution of a skill. Returns the skill's steps to follow.

REQUIRED: skill_id
Optional: context (what triggered the execution)

Call this before following a skill's steps. Use complete_execution when done.

Example: execute_skill(skill_id: "uuid", context: "Production deploy requested")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Execute Skill",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleExecuteSkill)

	mcp.AddTool(server, &mcp.Tool{
		Name: "complete_execution",
		Description: `🟡 COMMON | Mark a skill execution as complete.

REQUIRED: execution_id, success (boolean)
Optional: notes (outcome details)

Call this after following a skill's steps to track effectiveness.

Example: complete_execution(execution_id: "uuid", success: true, notes: "Deployment successful")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Complete Execution",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCompleteExecution)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_skill_stats",
		Description: `🟡 COMMON | Get execution statistics for a skill.

REQUIRED: skill_id

Returns total executions, success/failure counts, success rate, and last execution time.

Example: get_skill_stats(skill_id: "uuid")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Skill Stats",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetSkillStats)

	mcp.AddTool(server, &mcp.Tool{
		Name: "generate_skill",
		Description: `🟡 COMMON | Generate a procedural skill from a natural language description using AI.

REQUIRED: description (what the skill should accomplish)
Optional: project (save to this project), auto_save (save automatically if true)

Uses Gemini to generate trigger, steps, and validation criteria.

Example: generate_skill(description: "How to deploy a Next.js app to Vercel", project: "Ramorie Frontend", auto_save: true)`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Generate Skill",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleGenerateSkill)
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
		"step_3": "📝 Use remember(content) or create_task(project, description)",
		"note":   "Always pass the 'project' parameter explicitly when creating tasks or memories",
	}

	// Add agent directives for proactive behavior
	result["agent_directives"] = GetDirectivesAsMap()

	return mustTextResult(result), nil, nil
}

func handleListProjects(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
	// Get ALL accessible projects (no org filtering) - enables cross-org project access
	projects, err := apiClient.ListProjects("")
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
					"_action":       "Use remember(content) for memories or create_task(project, description) for tasks.",
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

// ============================================================================
// REMEMBER TOOL - Ultra-simple memory storage (replaces add_memory)
// ============================================================================

type RememberInput struct {
	Content string `json:"content"`           // REQUIRED - what to remember
	Project string `json:"project,omitempty"` // OPTIONAL - auto-detected from last used
}

func handleRemember(ctx context.Context, req *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, any, error) {
	// Check session initialization
	if err := checkSessionInit("remember"); err != nil {
		return nil, nil, err
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, nil, errors.New("content is required - tell me what to remember")
	}

	// Check if this should be a task instead
	if shouldBeTask(content) {
		taskDesc := extractTaskDescription(content)

		// Resolve project (auto-detect if empty)
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
		scope := determineEncryptionScope(orgID)

		// Check if user has encryption enabled
		cfg, _ := config.LoadConfig()
		if cfg != nil && cfg.EncryptionEnabled {
			if scope == "organization" && orgID != "" && crypto.IsOrgVaultUnlocked(orgID) {
				encryptedDesc, nonce, isEncrypted, err := crypto.EncryptContentWithScope(taskDesc, "organization", orgID)
				if err == nil && isEncrypted {
					task, err := apiClient.CreateEncryptedTaskWithMeta(projectID, encryptedDesc, nonce, "M", meta)
					if err != nil {
						return nil, nil, err
					}
					return mustTextResult(map[string]interface{}{
						"action":    "task_created",
						"message":   "📋 Created task instead of memory (detected TODO)",
						"task":      task,
						"encrypted": true,
					}), nil, nil
				}
			} else if scope == "personal" && crypto.IsVaultUnlocked() {
				encryptedDesc, nonce, isEncrypted, err := crypto.EncryptContent(taskDesc)
				if err == nil && isEncrypted {
					task, err := apiClient.CreateEncryptedTaskWithMeta(projectID, encryptedDesc, nonce, "M", meta)
					if err != nil {
						return nil, nil, err
					}
					return mustTextResult(map[string]interface{}{
						"action":    "task_created",
						"message":   "📋 Created task instead of memory (detected TODO)",
						"task":      task,
						"encrypted": true,
					}), nil, nil
				}
			}
		}

		// Create non-encrypted task (title=taskDesc, empty description)
		task, err := apiClient.CreateTaskWithMeta(projectID, taskDesc, "", "M", meta)
		if err != nil {
			return nil, nil, err
		}

		return mustTextResult(map[string]interface{}{
			"action":  "task_created",
			"message": "📋 Created task instead of memory (detected TODO)",
			"task":    task,
		}), nil, nil
	}

	// Resolve project (auto-detect if empty)
	projectID, orgID, err := resolveProjectWithOrg(apiClient, input.Project)
	if err != nil {
		return nil, nil, err
	}

	// Auto-detect memory type
	memoryType := DetectMemoryType(content)

	// Determine encryption scope
	scope := determineEncryptionScope(orgID)

	// Check if user has encryption enabled
	cfg, _ := config.LoadConfig()
	if cfg != nil && cfg.EncryptionEnabled {
		if scope == "organization" && orgID != "" && crypto.IsOrgVaultUnlocked(orgID) {
			encryptedContent, nonce, isEncrypted, err := crypto.EncryptContentWithScope(content, "organization", orgID)
			if err == nil && isEncrypted {
				memory, err := apiClient.CreateEncryptedMemory(projectID, encryptedContent, nonce)
				if err != nil {
					return nil, nil, err
				}
				return mustTextResult(map[string]interface{}{
					"action":         "memory_saved",
					"message":        fmt.Sprintf("💾 Remembered as %s (org-encrypted)", memoryType),
					"memory":         memory,
					"type":           memoryType,
					"auto_detected":  true,
					"encrypted":      true,
					"project_id":     projectID,
				}), nil, nil
			}
		} else if scope == "personal" && crypto.IsVaultUnlocked() {
			encryptedContent, nonce, isEncrypted, err := crypto.EncryptContent(content)
			if err == nil && isEncrypted {
				memory, err := apiClient.CreateEncryptedMemory(projectID, encryptedContent, nonce)
				if err != nil {
					return nil, nil, err
				}
				return mustTextResult(map[string]interface{}{
					"action":         "memory_saved",
					"message":        fmt.Sprintf("💾 Remembered as %s (encrypted)", memoryType),
					"memory":         memory,
					"type":           memoryType,
					"auto_detected":  true,
					"encrypted":      true,
					"project_id":     projectID,
				}), nil, nil
			}
		}
	}

	// Create non-encrypted memory with auto-detected type
	memory, err := apiClient.CreateMemoryWithOptions(api.CreateMemoryOptions{
		ProjectID: projectID,
		Content:   content,
		Type:      memoryType,
	})
	if err != nil {
		return nil, nil, err
	}

	return mustTextResult(map[string]interface{}{
		"action":        "memory_saved",
		"message":       fmt.Sprintf("💾 Remembered as %s", memoryType),
		"memory":        memory,
		"type":          memoryType,
		"auto_detected": true,
		"project_id":    projectID,
	}), nil, nil
}

// shouldBeTask checks if the content looks like a TODO/task rather than a memory
func shouldBeTask(content string) bool {
	lower := strings.ToLower(content)
	taskIndicators := []string{
		"todo:", "todo ", "later:", "later ",
		"need to ", "should ", "must ", "reminder:",
		"task:", "action:", "followup:", "follow-up:",
	}
	for _, indicator := range taskIndicators {
		if strings.HasPrefix(lower, indicator) || strings.Contains(lower, " "+indicator) {
			return true
		}
	}
	return false
}

// extractTaskDescription extracts the task description from TODO-like content
func extractTaskDescription(content string) string {
	lower := strings.ToLower(content)
	prefixes := []string{
		"todo:", "todo ", "later:", "later ",
		"need to ", "should ", "must ", "reminder:",
		"task:", "action:", "followup:", "follow-up:",
	}

	for _, prefix := range prefixes {
		idx := strings.Index(lower, prefix)
		if idx != -1 {
			// Return everything after the prefix
			result := strings.TrimSpace(content[idx+len(prefix):])
			if result != "" {
				return result
			}
		}
	}

	// Fallback: return original content
	return content
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

	// Knowledge graph integration
	EntityHops float64 `json:"entity_hops,omitempty"` // 0-3: Include memories from related entities via knowledge graph

	// Temporal query support
	ValidAt        string `json:"valid_at,omitempty"`        // RFC3339 timestamp - query memories valid at this time (default: now)
	IncludeExpired bool   `json:"include_expired,omitempty"` // Include TTL-expired memories (default: false)
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

	// Check if entity_hops is enabled for knowledge graph traversal
	entityHops := int(input.EntityHops)
	if entityHops < 0 {
		entityHops = 0
	}
	if entityHops > 3 {
		entityHops = 3
	}

	// Collect memory IDs to fetch via entity graph traversal
	var entityMemoryIDs map[string]int // memory ID -> hop distance
	var matchedEntities []map[string]interface{}

	if entityHops > 0 {
		entityMemoryIDs = make(map[string]int)

		// Search for entities matching the term
		entityResp, err := apiClient.ListEntities("", projectID, term, 20, 0)
		if err == nil && len(entityResp.Entities) > 0 {
			for _, entity := range entityResp.Entities {
				matchedEntities = append(matchedEntities, map[string]interface{}{
					"id":   entity.ID.String(),
					"name": entity.Name,
					"type": entity.Type,
				})

				// Get memories for this entity with hops
				memResp, err := apiClient.GetEntityMemories(entity.ID.String(), entityHops, 50)
				if err == nil {
					for _, mid := range memResp.MemoryIDs {
						// Track hop distance (direct=0, related=hops)
						if _, exists := entityMemoryIDs[mid]; !exists {
							entityMemoryIDs[mid] = entityHops
						}
					}
				}
			}
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

		// Check if memory is linked to an entity (knowledge graph boost)
		memoryIDStr := m.ID.String()
		entityBoost := 0
		fromEntity := false
		if entityMemoryIDs != nil {
			if hopDistance, exists := entityMemoryIDs[memoryIDStr]; exists {
				fromEntity = true
				// Boost based on hop distance (closer = higher boost)
				// Direct (0 hops) = +30, 1 hop = +20, 2 hops = +10, 3 hops = +5
				entityBoost = max(30-hopDistance*10, 5)
			}
		}

		if isAndSearch && matchCount < len(searchTerms) {
			// If entity_hops is enabled and memory is from entity graph, still include it
			if !fromEntity {
				continue
			}
		}
		if !isAndSearch && matchCount == 0 {
			// If entity_hops is enabled and memory is from entity graph, still include it
			if !fromEntity {
				continue
			}
		}

		// Apply entity boost
		score += entityBoost

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
		if fromEntity {
			result["from_entity"] = true
			result["entity_boost"] = entityBoost
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
	if entityHops > 0 {
		response["entity_hops"] = entityHops
		response["matched_entities"] = matchedEntities
	}

	return mustTextResult(response), nil, nil
}

// SurfaceSkillsInput for proactive skill surfacing
type SurfaceSkillsInput struct {
	Context string `json:"context"`          // Current task/situation description
	Project string `json:"project,omitempty"` // Optional: limit to specific project
	Limit   int    `json:"limit,omitempty"`   // Max skills to return (default 5)
}

// handleSurfaceSkills finds relevant procedural skills based on the given context
func handleSurfaceSkills(ctx context.Context, req *mcp.CallToolRequest, input SurfaceSkillsInput) (*mcp.CallToolResult, any, error) {
	context := strings.TrimSpace(input.Context)
	if context == "" {
		return nil, nil, errors.New("context is required - describe the current task or situation")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}

	projectID := ""
	if strings.TrimSpace(input.Project) != "" {
		pid, err := resolveProjectID(apiClient, input.Project)
		if err == nil {
			projectID = pid
		}
	}

	// Fetch all memories and filter for skills
	memories, err := apiClient.ListMemories(projectID, "")
	if err != nil {
		return nil, nil, err
	}

	contextLower := strings.ToLower(context)
	contextWords := strings.Fields(contextLower)

	type scoredSkill struct {
		skill interface{}
		score int
	}
	var scored []scoredSkill

	for _, m := range memories {
		// Only consider skill-type memories
		if m.Type != "skill" {
			continue
		}

		// Decrypt content for matching
		decryptedContent := decryptMemoryContent(&m)

		// Score based on trigger field matching (if available)
		score := 0
		triggerMatches := 0

		// Check trigger field for matches
		triggerStr := ""
		if m.Trigger != nil {
			triggerStr = strings.ToLower(*m.Trigger)
		}

		// Match context words against trigger
		for _, word := range contextWords {
			if len(word) < 3 {
				continue // Skip short words
			}
			if triggerStr != "" && strings.Contains(triggerStr, word) {
				triggerMatches++
				score += 30 // High weight for trigger matches
			}
			// Also match against content
			if strings.Contains(strings.ToLower(decryptedContent), word) {
				score += 10
			}
		}

		// Require at least some relevance
		if score < 10 {
			continue
		}

		// Build skill result
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
		if m.Importance != nil {
			result["importance"] = *m.Importance
		}
		if m.Project != nil {
			result["project"] = map[string]interface{}{
				"id":   m.Project.ID.String(),
				"name": m.Project.Name,
			}
		}

		scored = append(scored, scoredSkill{skill: result, score: score})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Limit results
	if len(scored) > limit {
		scored = scored[:limit]
	}

	var results []interface{}
	for _, s := range scored {
		results = append(results, s.skill)
	}

	response := map[string]interface{}{
		"context":       context,
		"skills_found":  len(results),
		"skills":        results,
		"_message":      fmt.Sprintf("Found %d relevant skills for the given context", len(results)),
	}

	if len(results) == 0 {
		response["_hint"] = "No matching skills found. Consider creating a skill memory with remember(content) including trigger, steps, and validation info."
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
// CONTEXT PACK EXPORT/IMPORT HANDLERS
// ============================================================================

type ExportContextPackInput struct {
	PackID string `json:"pack_id"` // REQUIRED
}

func handleExportContextPack(ctx context.Context, req *mcp.CallToolRequest, input ExportContextPackInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("export_context_pack"); err != nil {
		return nil, nil, err
	}

	packID := strings.TrimSpace(input.PackID)
	if packID == "" {
		return nil, nil, errors.New("pack_id is required")
	}

	// Call export endpoint
	respBody, err := apiClient.Request("GET", "/context-packs/"+packID+"/export", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("export failed: %w", err)
	}

	var bundle map[string]interface{}
	if err := json.Unmarshal(respBody, &bundle); err != nil {
		return nil, nil, fmt.Errorf("failed to parse export bundle: %w", err)
	}

	result := formatMCPResponse(bundle, getContextString())
	result["_message"] = "✅ Context pack exported successfully. Use import_context_pack to import this bundle elsewhere."
	result["_format"] = "json"
	result["_version"] = bundle["version"]

	return mustTextResult(result), nil, nil
}

type ImportContextPackInput struct {
	Bundle       map[string]interface{} `json:"bundle"`        // REQUIRED: The export bundle (JSON object)
	Project      string                 `json:"project"`       // REQUIRED: Target project name or ID
	ConflictMode string                 `json:"conflict_mode"` // Optional: skip, overwrite, rename (default)
}

func handleImportContextPack(ctx context.Context, req *mcp.CallToolRequest, input ImportContextPackInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("import_context_pack"); err != nil {
		return nil, nil, err
	}

	if input.Bundle == nil {
		return nil, nil, errors.New("bundle is required")
	}
	project := strings.TrimSpace(input.Project)
	if project == "" {
		return nil, nil, errors.New("project is required")
	}

	projectID, err := resolveProjectID(apiClient, project)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve project: %w", err)
	}

	// Build import request
	importReq := map[string]interface{}{
		"bundle":     input.Bundle,
		"project_id": projectID,
	}
	if input.ConflictMode != "" {
		importReq["conflict_mode"] = input.ConflictMode
	}

	// Call import endpoint
	respBody, err := apiClient.Request("POST", "/context-packs/import", importReq)
	if err != nil {
		return nil, nil, fmt.Errorf("import failed: %w", err)
	}

	var importResult map[string]interface{}
	if err := json.Unmarshal(respBody, &importResult); err != nil {
		return nil, nil, fmt.Errorf("failed to parse import result: %w", err)
	}

	result := formatMCPResponse(importResult, getContextString())
	result["_message"] = "✅ Context pack imported successfully."

	return mustTextResult(result), nil, nil
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

type ConsolidateMemoriesInput struct {
	Project          string  `json:"project"`
	StaleDays        int     `json:"stale_days,omitempty"`
	PromoteThreshold float64 `json:"promote_threshold,omitempty"`
	ArchiveThreshold float64 `json:"archive_threshold,omitempty"`
}

func handleConsolidateMemories(ctx context.Context, req *mcp.CallToolRequest, input ConsolidateMemoriesInput) (*mcp.CallToolResult, interface{}, error) {
	project := strings.TrimSpace(input.Project)
	if project == "" {
		return nil, nil, errors.New("project is required")
	}

	projectID, err := resolveProjectID(apiClient, project)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve project: %w", err)
	}

	payload := map[string]interface{}{
		"project_id": projectID,
	}
	if input.StaleDays > 0 {
		payload["stale_days"] = input.StaleDays
	}
	if input.PromoteThreshold > 0 {
		payload["promote_threshold"] = input.PromoteThreshold
	}
	if input.ArchiveThreshold > 0 {
		payload["archive_threshold"] = input.ArchiveThreshold
	}

	// Add org_id if session has one
	if session := GetCurrentSession(); session != nil && session.ActiveOrgID != nil {
		payload["org_id"] = session.ActiveOrgID.String()
	}

	result, err := apiClient.EnqueueJob("memory_consolidate", payload)
	if err != nil {
		return nil, nil, fmt.Errorf("enqueue consolidation job: %w", err)
	}

	return mustTextResult(result), nil, nil
}

type CleanupMemoriesInput struct {
	Project   string `json:"project,omitempty"`   // Optional: scope to specific project
	DryRun    bool   `json:"dry_run,omitempty"`   // Preview without deleting
	BatchSize int    `json:"batch_size,omitempty"` // Batch size for deletion (default 100)
}

func handleCleanupMemories(ctx context.Context, req *mcp.CallToolRequest, input CleanupMemoriesInput) (*mcp.CallToolResult, interface{}, error) {
	payload := map[string]interface{}{}

	// Resolve project if provided
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

	// Add org_id if session has one
	if session := GetCurrentSession(); session != nil && session.ActiveOrgID != nil {
		payload["org_id"] = session.ActiveOrgID.String()
	}

	result, err := apiClient.EnqueueJob("memory_cleanup", payload)
	if err != nil {
		return nil, nil, fmt.Errorf("enqueue cleanup job: %w", err)
	}

	return mustTextResult(result), nil, nil
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
		{Name: "remember", Description: "🔴 ESSENTIAL | Ultra-simple memory storage with auto-detection."},
		{Name: "recall", Description: "🔴 ESSENTIAL | Search memories (filter type='skill' for learned procedures)."},
		{Name: "surface_skills", Description: "🟡 COMMON | Find relevant procedural skills based on context."},
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
		{Name: "manage_context_pack", Description: "🟡 COMMON | Create/update/link_memory/link_task to context packs."},
		{Name: "create_decision", Description: "🟡 COMMON | Record an architectural decision (ADR)."},
		{Name: "list_decisions", Description: "🟡 COMMON | List architectural decisions."},
		{Name: "get_stats", Description: "🟡 COMMON | Get task statistics and completion rates."},
		{Name: "get_agent_activity", Description: "🟡 COMMON | Get recent agent activity timeline."},
		// ============================================================================
		// 🟢 ADVANCED (9 tools)
		// ============================================================================
		{Name: "create_project", Description: "🟢 ADVANCED | Create a new project."},
		{Name: "move_task", Description: "🟢 ADVANCED | Move a task to a different project."},
		{Name: "manage_subtasks", Description: "🟢 ADVANCED | Create/list/complete/update subtasks."},
		{Name: "manage_dependencies", Description: "🟢 ADVANCED | Add/list/remove task dependencies."},
		{Name: "manage_plan", Description: "🟢 ADVANCED | Create/status/list/apply/cancel multi-agent plans."},
		{Name: "list_organizations", Description: "🟢 ADVANCED | List user's organizations."},
		{Name: "switch_organization", Description: "🟢 ADVANCED | Switch active organization context."},
		{Name: "consolidate_memories", Description: "🟢 ADVANCED | Trigger memory consolidation (scores, promotes, archives)."},
		{Name: "cleanup_memories", Description: "🟢 ADVANCED | Clean up TTL-expired memories."},
		{Name: "export_context_pack", Description: "🟢 ADVANCED | Export context pack as portable JSON bundle."},
		{Name: "import_context_pack", Description: "🟢 ADVANCED | Import context pack bundle with all linked items."},
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

// normalizeForMatch removes spaces, hyphens, underscores, dots for fuzzy path matching
// "Ramorie Frontend" -> "ramoriefrontend"
// "ramorie-frontend" -> "ramoriefrontend"
// "orkai.io" -> "orkaiio"
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

func resolveProjectID(client *api.Client, projectIdentifier string) (string, error) {
	projectIdentifier = strings.TrimSpace(projectIdentifier)

	// Get ALL accessible projects (no org filtering) - enables cross-org project access
	projects, err := client.ListProjects("")
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	// AUTO-DETECT: If project not provided, try to auto-detect
	if projectIdentifier == "" {
		// Try 1: Use last used project from session
		if lastProject := GetSessionLastProjectID(); lastProject != nil {
			// Verify it still exists
			for _, p := range projects {
				if p.ID == *lastProject {
					return lastProject.String(), nil
				}
			}
		}

		// Try 2: CWD Match - Current working directory contains project name
		if cwd, err := os.Getwd(); err == nil && cwd != "" {
			cwdLower := strings.ToLower(cwd)
			// Extract path segments for matching
			pathSegments := strings.Split(cwdLower, "/")

			for _, p := range projects {
				projectNameLower := strings.ToLower(p.Name)
				// Normalize: remove spaces, hyphens, underscores, dots for fuzzy matching
				projectNorm := normalizeForMatch(projectNameLower)

				// Check each path segment
				for _, segment := range pathSegments {
					if segment == "" {
						continue
					}
					segmentNorm := normalizeForMatch(segment)

					// Exact normalized match: "ramorie-frontend" == "Ramorie Frontend"
					if segmentNorm == projectNorm {
						SetSessionLastProject(p.ID)
						return p.ID.String(), nil
					}

					// Prefix match: "ramorie-frontend-app" contains "ramorie-frontend"
					if len(projectNorm) >= 4 && strings.HasPrefix(segmentNorm, projectNorm) {
						SetSessionLastProject(p.ID)
						return p.ID.String(), nil
					}
				}
			}
		}

		// Try 3: Use single project if user has only one
		if len(projects) == 1 {
			projectID := projects[0].ID.String()
			// Track this as last used
			SetSessionLastProject(projects[0].ID)
			return projectID, nil
		}

		// No auto-detection possible - require explicit project
		if len(projects) == 0 {
			return "", errors.New("❌ Project not specified and no projects found.\n\n" +
				"Create a project first:\n" +
				"  create_project(name=\"my-project\", description=\"...\")")
		}

		names := make([]string, 0, len(projects))
		for _, p := range projects {
			name := p.Name
			if p.Organization != nil && p.Organization.Name != "" {
				name = fmt.Sprintf("%s (%s)", p.Name, p.Organization.Name)
			}
			names = append(names, name)
		}
		return "", fmt.Errorf("❌ Project not specified.\n\n"+
			"Available projects: %s\n\n"+
			"💡 Specify project explicitly, or use a project first to set default.",
			strings.Join(names, ", "))
	}

	// Check for exact name match or ID prefix match
	for _, p := range projects {
		if strings.EqualFold(p.Name, projectIdentifier) || strings.HasPrefix(p.ID.String(), projectIdentifier) {
			// Track this as last used project
			SetSessionLastProject(p.ID)
			return p.ID.String(), nil
		}
	}

	// Build helpful error message with available projects
	if len(projects) == 0 {
		return "", fmt.Errorf("❌ Project '%s' not found.\n\n"+
			"No projects found. Create one with:\n"+
			"  create_project(name=\"%s\", description=\"...\")",
			projectIdentifier, projectIdentifier)
	}

	// List available projects (include org name for clarity)
	availableNames := make([]string, 0, len(projects))
	for _, p := range projects {
		name := p.Name
		if p.Organization != nil && p.Organization.Name != "" {
			name = fmt.Sprintf("%s (%s)", p.Name, p.Organization.Name)
		}
		availableNames = append(availableNames, name)
	}

	return "", fmt.Errorf("❌ Project '%s' not found.\n\n"+
		"Available projects: %s\n\n"+
		"💡 Tip: Use exact project name (case-insensitive)",
		projectIdentifier, strings.Join(availableNames, ", "))
}

// resolveProjectWithOrg resolves project ID and returns the org ID if the project belongs to an org
func resolveProjectWithOrg(client *api.Client, projectIdentifier string) (projectID string, orgID string, err error) {
	projectIdentifier = strings.TrimSpace(projectIdentifier)

	// Get ALL accessible projects (no org filtering) - enables cross-org project access
	projects, err := client.ListProjects("")
	if err != nil {
		return "", "", fmt.Errorf("failed to list projects: %w", err)
	}

	// AUTO-DETECT: If project not provided, try to auto-detect
	if projectIdentifier == "" {
		// Try 1: Use last used project from session
		if lastProject := GetSessionLastProjectID(); lastProject != nil {
			// Verify it still exists and get org info
			for _, p := range projects {
				if p.ID == *lastProject {
					oid := ""
					if p.OrganizationID != nil {
						oid = p.OrganizationID.String()
					}
					return lastProject.String(), oid, nil
				}
			}
		}

		// Try 2: CWD Match - Current working directory contains project name
		if cwd, err := os.Getwd(); err == nil && cwd != "" {
			cwdLower := strings.ToLower(cwd)
			pathSegments := strings.Split(cwdLower, "/")

			for _, p := range projects {
				projectNameLower := strings.ToLower(p.Name)
				projectNorm := normalizeForMatch(projectNameLower)

				for _, segment := range pathSegments {
					if segment == "" {
						continue
					}
					segmentNorm := normalizeForMatch(segment)

					if segmentNorm == projectNorm || (len(projectNorm) >= 4 && strings.HasPrefix(segmentNorm, projectNorm)) {
						SetSessionLastProject(p.ID)
						oid := ""
						if p.OrganizationID != nil {
							oid = p.OrganizationID.String()
						}
						return p.ID.String(), oid, nil
					}
				}
			}
		}

		// Try 3: Use single project if user has only one
		if len(projects) == 1 {
			p := projects[0]
			// Track this as last used
			SetSessionLastProject(p.ID)
			oid := ""
			if p.OrganizationID != nil {
				oid = p.OrganizationID.String()
			}
			return p.ID.String(), oid, nil
		}

		// No auto-detection possible - require explicit project
		if len(projects) == 0 {
			return "", "", errors.New("❌ Project not specified and no projects found.\n\n" +
				"Create a project first:\n" +
				"  create_project(name=\"my-project\", description=\"...\")")
		}

		names := make([]string, 0, len(projects))
		for _, p := range projects {
			name := p.Name
			if p.Organization != nil && p.Organization.Name != "" {
				name = fmt.Sprintf("%s (%s)", p.Name, p.Organization.Name)
			}
			names = append(names, name)
		}
		return "", "", fmt.Errorf("❌ Project not specified.\n\n"+
			"Available projects: %s\n\n"+
			"💡 Specify project explicitly, or use a project first to set default.",
			strings.Join(names, ", "))
	}

	// Check for exact name match or ID prefix match
	for _, p := range projects {
		if strings.EqualFold(p.Name, projectIdentifier) || strings.HasPrefix(p.ID.String(), projectIdentifier) {
			// Track this as last used project
			SetSessionLastProject(p.ID)
			pid := p.ID.String()
			oid := ""
			if p.OrganizationID != nil {
				oid = p.OrganizationID.String()
			}
			return pid, oid, nil
		}
	}

	// Build helpful error message with available projects
	if len(projects) == 0 {
		return "", "", fmt.Errorf("❌ Project '%s' not found.\n\n"+
			"No projects found. Create one with:\n"+
			"  create_project(name=\"%s\", description=\"...\")",
			projectIdentifier, projectIdentifier)
	}

	// List available projects (include org name for clarity)
	availableNames := make([]string, 0, len(projects))
	for _, p := range projects {
		name := p.Name
		if p.Organization != nil && p.Organization.Name != "" {
			name = fmt.Sprintf("%s (%s)", p.Name, p.Organization.Name)
		}
		availableNames = append(availableNames, name)
	}

	return "", "", fmt.Errorf("❌ Project '%s' not found.\n\n"+
		"Available projects: %s\n\n"+
		"💡 Tip: Use exact project name (case-insensitive)",
		projectIdentifier, strings.Join(availableNames, ", "))
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

// ============================================================================
// ENTITY & KNOWLEDGE GRAPH TOOL HANDLERS
// ============================================================================

type ListEntitiesInput struct {
	Type    string  `json:"type,omitempty"`    // Entity type filter
	Project string  `json:"project,omitempty"` // Project name or ID
	Query   string  `json:"query,omitempty"`   // Search query
	Limit   float64 `json:"limit,omitempty"`
	Offset  float64 `json:"offset,omitempty"`
}

func handleListEntities(ctx context.Context, req *mcp.CallToolRequest, input ListEntitiesInput) (*mcp.CallToolResult, any, error) {
	entityType := strings.TrimSpace(input.Type)
	query := strings.TrimSpace(input.Query)

	// Resolve project if provided
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

	// Format entities for output
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
		"_context": getContextString(),
	}), nil, nil
}

type GetEntityInput struct {
	EntityID string `json:"entity_id"`
}

func handleGetEntity(ctx context.Context, req *mcp.CallToolRequest, input GetEntityInput) (*mcp.CallToolResult, any, error) {
	entityID := strings.TrimSpace(input.EntityID)
	if entityID == "" {
		return nil, nil, errors.New("entity_id is required")
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
	if entity.ProjectID != nil {
		result["project_id"] = entity.ProjectID.String()
	}

	return mustTextResult(result), nil, nil
}

type GetEntityGraphInput struct {
	EntityID string  `json:"entity_id"`
	Hops     float64 `json:"hops,omitempty"`
}

func handleGetEntityGraph(ctx context.Context, req *mcp.CallToolRequest, input GetEntityGraphInput) (*mcp.CallToolResult, any, error) {
	entityID := strings.TrimSpace(input.EntityID)
	if entityID == "" {
		return nil, nil, errors.New("entity_id is required")
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
		"_context":    getContextString(),
	}), nil, nil
}

type GetMemoryEntitiesInput struct {
	MemoryID string `json:"memory_id"`
}

func handleGetMemoryEntities(ctx context.Context, req *mcp.CallToolRequest, input GetMemoryEntitiesInput) (*mcp.CallToolResult, any, error) {
	memoryID := strings.TrimSpace(input.MemoryID)
	if memoryID == "" {
		return nil, nil, errors.New("memory_id is required")
	}

	response, err := apiClient.GetMemoryEntities(memoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get memory entities: %w", err)
	}

	// Format entities for output
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
		"_context":  getContextString(),
	}), nil, nil
}

type GetEntityMemoriesInput struct {
	EntityID string  `json:"entity_id"`
	Hops     float64 `json:"hops,omitempty"`
	Limit    float64 `json:"limit,omitempty"`
}

func handleGetEntityMemories(ctx context.Context, req *mcp.CallToolRequest, input GetEntityMemoriesInput) (*mcp.CallToolResult, any, error) {
	entityID := strings.TrimSpace(input.EntityID)
	if entityID == "" {
		return nil, nil, errors.New("entity_id is required")
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
		"_context":   getContextString(),
	}), nil, nil
}

func handleGetEntityStats(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
	response, err := apiClient.GetEntityStats()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get entity stats: %w", err)
	}

	return mustTextResult(map[string]interface{}{
		"total_entities":      response.TotalEntities,
		"total_relationships": response.TotalRelationships,
		"entities_by_type":    response.EntitiesByType,
		"_context":            getContextString(),
	}), nil, nil
}

type CreateEntityInput struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Project     string   `json:"project,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
}

func handleCreateEntity(ctx context.Context, req *mcp.CallToolRequest, input CreateEntityInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("create_entity"); err != nil {
		return nil, nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}

	entityType := strings.TrimSpace(input.Type)
	if entityType == "" {
		return nil, nil, errors.New("type is required")
	}

	// Validate entity type
	validTypes := map[string]bool{
		"person": true, "tool": true, "concept": true, "project": true,
		"organization": true, "location": true, "event": true,
		"document": true, "api": true, "other": true,
	}
	if !validTypes[entityType] {
		return nil, nil, fmt.Errorf("invalid entity type: %s. Valid types: person, tool, concept, project, organization, location, event, document, api, other", entityType)
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
		"action":   "entity_created",
		"entity":   entity,
		"_message": fmt.Sprintf("✅ Entity '%s' created successfully", entity.Name),
	}), nil, nil
}

type CreateRelationshipInput struct {
	SourceEntityID   string  `json:"source_entity_id"`
	TargetEntityID   string  `json:"target_entity_id"`
	RelationshipType string  `json:"relationship_type"`
	Label            string  `json:"label,omitempty"`
	Description      string  `json:"description,omitempty"`
	Strength         float64 `json:"strength,omitempty"`
}

func handleCreateRelationship(ctx context.Context, req *mcp.CallToolRequest, input CreateRelationshipInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("create_relationship"); err != nil {
		return nil, nil, err
	}

	sourceID := strings.TrimSpace(input.SourceEntityID)
	if sourceID == "" {
		return nil, nil, errors.New("source_entity_id is required")
	}

	targetID := strings.TrimSpace(input.TargetEntityID)
	if targetID == "" {
		return nil, nil, errors.New("target_entity_id is required")
	}

	relType := strings.TrimSpace(input.RelationshipType)
	if relType == "" {
		return nil, nil, errors.New("relationship_type is required")
	}

	// Validate relationship type
	validTypes := map[string]bool{
		"uses": true, "works_on": true, "related_to": true, "depends_on": true,
		"part_of": true, "created_by": true, "belongs_to": true, "connects_to": true,
		"replaces": true, "similar_to": true, "contradicts": true, "references": true,
		"implements": true, "extends": true,
	}
	if !validTypes[relType] {
		return nil, nil, fmt.Errorf("invalid relationship type: %s. Valid types: uses, works_on, related_to, depends_on, part_of, created_by, belongs_to, connects_to, replaces, similar_to, contradicts, references, implements, extends", relType)
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
		"action":       "relationship_created",
		"relationship": relationship,
		"_message":     fmt.Sprintf("✅ Relationship '%s' created between entities", relType),
	}), nil, nil
}

// ============================================================================
// SKILLS TOOLS HANDLERS
// ============================================================================

type ListSkillsInput struct {
	Project string `json:"project,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

func handleListSkills(ctx context.Context, req *mcp.CallToolRequest, input ListSkillsInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("list_skills"); err != nil {
		return nil, nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	// Get skills via memory list with type=skill filter
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

	// Format skills for output
	var result []map[string]interface{}
	for _, skill := range skills {
		item := map[string]interface{}{
			"id":          skill.ID,
			"content":     skill.Content,
			"trigger":     skill.Trigger,
			"steps":       skill.Steps,
			"validation":  skill.Validation,
			"tags":        skill.Tags,
			"created_at":  skill.CreatedAt,
		}
		if skill.Project != nil {
			item["project_name"] = skill.Project.Name
		}
		result = append(result, item)
	}

	return mustTextResult(map[string]interface{}{
		"action":   "skills_listed",
		"skills":   result,
		"count":    len(result),
		"_message": fmt.Sprintf("Found %d skills", len(result)),
	}), nil, nil
}

type CreateSkillInput struct {
	Project     string   `json:"project"`
	Trigger     string   `json:"trigger"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Validation  string   `json:"validation,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func handleCreateSkill(ctx context.Context, req *mcp.CallToolRequest, input CreateSkillInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("create_skill"); err != nil {
		return nil, nil, err
	}

	projectID, err := resolveProjectID(apiClient, input.Project)
	if err != nil || projectID == "" {
		return nil, nil, errors.New("project is required")
	}

	if input.Trigger == "" {
		return nil, nil, errors.New("trigger is required")
	}

	if input.Description == "" {
		return nil, nil, errors.New("description is required")
	}

	if len(input.Steps) == 0 {
		return nil, nil, errors.New("steps array is required and must not be empty")
	}

	// Create skill via memory API with type=skill
	skill, err := apiClient.CreateSkill(projectID, input.Trigger, input.Description, input.Steps, input.Validation, input.Tags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create skill: %w", err)
	}

	return mustTextResult(map[string]interface{}{
		"action":   "skill_created",
		"skill":    skill,
		"_message": fmt.Sprintf("✅ Skill created: %s", skill.Content),
	}), nil, nil
}

type ExecuteSkillInput struct {
	SkillID string `json:"skill_id"`
	Context string `json:"context,omitempty"`
}

func handleExecuteSkill(ctx context.Context, req *mcp.CallToolRequest, input ExecuteSkillInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("execute_skill"); err != nil {
		return nil, nil, err
	}

	if input.SkillID == "" {
		return nil, nil, errors.New("skill_id is required")
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

	// Extract steps from the skill (Steps is already []string in CLI models)
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
		"_message":     fmt.Sprintf("⚡ Execution started. Follow these %d steps, then call complete_execution.", len(steps)),
	}), nil, nil
}

type CompleteExecutionInput struct {
	ExecutionID string `json:"execution_id"`
	Success     bool   `json:"success"`
	Notes       string `json:"notes,omitempty"`
}

func handleCompleteExecution(ctx context.Context, req *mcp.CallToolRequest, input CompleteExecutionInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("complete_execution"); err != nil {
		return nil, nil, err
	}

	if input.ExecutionID == "" {
		return nil, nil, errors.New("execution_id is required")
	}

	execution, err := apiClient.CompleteSkillExecution(input.ExecutionID, input.Success, input.Notes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to complete execution: %w", err)
	}

	status := "✅ succeeded"
	if !input.Success {
		status = "❌ failed"
	}

	return mustTextResult(map[string]interface{}{
		"action":       "execution_completed",
		"execution_id": execution.ID,
		"success":      input.Success,
		"_message":     fmt.Sprintf("Execution %s", status),
	}), nil, nil
}

type GetSkillStatsInput struct {
	SkillID string `json:"skill_id"`
}

func handleGetSkillStats(ctx context.Context, req *mcp.CallToolRequest, input GetSkillStatsInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("get_skill_stats"); err != nil {
		return nil, nil, err
	}

	if input.SkillID == "" {
		return nil, nil, errors.New("skill_id is required")
	}

	stats, err := apiClient.GetSkillStats(input.SkillID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get skill stats: %w", err)
	}

	return mustTextResult(map[string]interface{}{
		"action":           "skill_stats_retrieved",
		"skill_id":         stats.SkillID,
		"total_executions": stats.TotalExecutions,
		"success_count":    stats.SuccessCount,
		"failure_count":    stats.FailureCount,
		"success_rate":     fmt.Sprintf("%.1f%%", stats.SuccessRate*100),
		"last_executed_at": stats.LastExecutedAt,
		"_message":         fmt.Sprintf("📊 %d executions (%.1f%% success rate)", stats.TotalExecutions, stats.SuccessRate*100),
	}), nil, nil
}

type GenerateSkillInput struct {
	Description string `json:"description"`
	Project     string `json:"project,omitempty"`
	AutoSave    bool   `json:"auto_save,omitempty"`
}

func handleGenerateSkill(ctx context.Context, req *mcp.CallToolRequest, input GenerateSkillInput) (*mcp.CallToolResult, any, error) {
	if err := checkSessionInit("generate_skill"); err != nil {
		return nil, nil, err
	}

	if input.Description == "" {
		return nil, nil, errors.New("description is required")
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
		"action":     "skill_generated",
		"skill":      response.Skill,
		"ai_model":   response.AIModel,
		"latency_ms": response.LatencyMs,
		"_message":   fmt.Sprintf("🤖 Skill generated (%.1f%% confidence)", response.Skill.Confidence*100),
	}

	if response.SavedID != nil {
		result["saved_id"] = response.SavedID
		result["_message"] = fmt.Sprintf("🤖 Skill generated and saved (ID: %s)", response.SavedID)
	}

	return mustTextResult(result), nil, nil
}

