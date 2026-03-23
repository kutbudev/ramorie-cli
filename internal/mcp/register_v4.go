package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerToolsV4 registers 16 simplified MCP tools (down from 49)
// This is the v4 tool set optimized for agent compliance
func registerToolsV4(server *mcp.Server) {
	// ============================================================================
	// 🔴 CORE (5 tools) - Every Session
	// ============================================================================

	// 1. setup_agent - Initialize session (KEEP)
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

	// 2. list_projects - List accessible projects (KEEP)
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

	// 3. remember - Store memories (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name: "remember",
		Description: `🔴 ESSENTIAL | Ultra-simple memory storage with duplicate prevention.

REQUIRED: content (what to remember), project (name or ID)
OPTIONAL: force (bool) - Skip similarity check and save anyway

⚠️ DUPLICATE PREVENTION: By default, remember() checks for similar existing memories.
If similar content exists (>80% match), you'll get a warning with the existing memory.

The type is auto-detected from content:
- "decided X" → decision
- "fixed bug" → bug_fix
- "prefer X" → preference
- "todo: X" / "later: X" → auto-creates TASK instead of memory

💡 Best Practice: Always call recall(term) BEFORE remember() to check existing knowledge.

Example: remember(content: "API uses JWT authentication", project: "my-project")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Remember",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleRemember)

	// 4. recall - Search memories AND decisions (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name: "recall",
		Description: `🔴 ESSENTIAL | Search memories AND architectural decisions using full-text search with relevance scoring.

REQUIRED: term
Returns ranked results from both memories and decisions.

Optional: project, tag, type, purpose (coding/research/review), min_score, limit, include_decisions (default: true)

Uses backend PostgreSQL full-text search with ts_rank scoring.
Project filters are applied server-side. Entity expansion is opt-in via entity_hops > 0.
Falls back to client-side search if backend unavailable.

Example: recall(term: "authentication") - finds auth-related memories and decisions`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Recall",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleRecall)

	// 5. task - Unified task management (NEW - replaces 6 tools)
	mcp.AddTool(server, &mcp.Tool{
		Name: "task",
		Description: `🔴 ESSENTIAL | Unified task management.

REQUIRED: action (list|get|create|start|complete|stop|progress|note|move)

Actions:
- list: List tasks. Requires: project
- get: Get task details. Requires: taskId
- create: Create task. Requires: project, description. Optional: priority (L/M/H)
- start: Start working on task. Requires: taskId
- complete: Mark task complete. Requires: taskId
- stop: Stop working on task. Requires: taskId
- progress: Update progress. Requires: taskId, progress (0-100)
- note: Add note to task. Requires: taskId, description
- move: Move task to project. Requires: taskId, projectId

Examples:
- task(action: "list", project: "my-project")
- task(action: "create", project: "my-project", description: "Add login", priority: "H")
- task(action: "start", taskId: "uuid")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Task",
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedTask)

	// ============================================================================
	// 🟡 COMMON (6 tools) - When Relevant
	// ============================================================================

	// 6. memory - Unified memory operations (NEW - replaces 2 tools)
	mcp.AddTool(server, &mcp.Tool{
		Name: "memory",
		Description: `🟡 COMMON | Get memory details with related entities.

REQUIRED: action (list|get)

Actions:
- list: List memories. Requires: project. Optional: term, limit
- get: Get memory details with related decisions, tasks, and memories. Requires: memoryId

Examples:
- memory(action: "list", project: "my-project")
- memory(action: "get", memoryId: "uuid")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Memory",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedMemory)

	// 7. decision - Unified decision management (NEW - replaces 2 tools)
	mcp.AddTool(server, &mcp.Tool{
		Name: "decision",
		Description: `🟡 COMMON | Record architectural decisions (ADRs). Searchable via recall().

REQUIRED: action (create|list)

Actions:
- create: Record decision. Requires: title. Optional: project, description, status, area, context, consequences
- list: List decisions. Optional: project, status, area, limit

Decisions are indexed with full-text search. Use recall(term) to find relevant decisions by keyword.

Examples:
- decision(action: "create", title: "Use PostgreSQL", project: "my-project")
- decision(action: "list", project: "my-project")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Decision",
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedDecision)

	// 8. skill - Unified procedural skill management (NEW - replaces 8 tools)
	mcp.AddTool(server, &mcp.Tool{
		Name: "skill",
		Description: `🟡 COMMON | Manage procedural skills (how-to knowledge).

REQUIRED: action (list|create|surface|execute|complete|stats|generate|update)

Actions:
- list: List skills. Optional: project, limit
- create: Create skill. Requires: project, trigger, description, steps[]. Optional: validation, tags[]
- surface: Find relevant skills. Requires: context. Optional: project, limit
- execute: Start skill execution. Requires: skill_id. Optional: context
- complete: Complete execution. Requires: execution_id, success. Optional: notes
- stats: Get skill stats. Requires: skill_id
- generate: AI-generate skill. Requires: description. Optional: project, auto_save
- update: Update skill. Requires: skill_id. Optional: trigger, description, steps[], validation, tags[]

Examples:
- skill(action: "surface", context: "deploying to production")
- skill(action: "create", project: "my-project", trigger: "When deploying", description: "Deploy procedure", steps: ["Build", "Test", "Deploy"])`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Skill",
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedSkill)

	// 9. manage_focus - Workspace focus management (KEEP - already unified)
	mcp.AddTool(server, &mcp.Tool{
		Name: "manage_focus",
		Description: `🟡 COMMON | Get, set, or clear active workspace focus.

No params = get current focus
With pack_id = set focus
With clear=true = clear focus`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Manage Focus",
			OpenWorldHint: boolPtr(false),
		},
	}, handleManageFocus)

	// 10. get_stats - Task statistics (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stats",
		Description: "🟡 COMMON | Get task statistics and completion rates. REQUIRED: project.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Stats",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetStats)

	// 11. get_agent_activity - Activity timeline (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_agent_activity",
		Description: "🟡 COMMON | Get recent agent activity timeline. Optional: project, agent_name, event_type, limit.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get Agent Activity",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleGetAgentActivity)

	// 12. surface_context - Proactive context surfacing (NEW)
	mcp.AddTool(server, &mcp.Tool{
		Name: "surface_context",
		Description: `🟡 COMMON | Proactively surface relevant decisions and memories based on your current work context.

CALL THIS before writing code to see relevant decisions, bug fixes, and patterns.

REQUIRED: At least one of:
- file_paths: Array of file paths being edited (e.g., ["src/store/api/userApi.ts"])
- domains: Array of domains/modules (e.g., ["admin", "auth", "api"])
- code_patterns: Array of code patterns about to be used (e.g., ["fetch(", "useEffect"])

OPTIONAL:
- project: Project name or ID
- purpose: "coding" (default), "research", "review"
- limit: Max results (default 10)

Returns decisions, bug fixes, and patterns relevant to your current work.`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Surface Context",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleSurfaceContext)

	// ============================================================================
	// 🟢 ADVANCED (4 tools) - Explicit Need
	// ============================================================================

	// 13. create_project - Create new project (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name: "create_project",
		Description: `🟢 ADVANCED | Create a new project.

REQUIRED: name, description
OPTIONAL: force (bypass duplicate check), org_id (organization UUID)

Example: create_project(name: "my-project", description: "My awesome project")`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Project",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleCreateProject)

	// 14. subtask - Unified subtask and dependency management (KEEP - uses existing merged handlers)
	mcp.AddTool(server, &mcp.Tool{
		Name: "manage_subtasks",
		Description: `🟢 ADVANCED | CRUD for subtasks.

REQUIRED: action (create|list|complete|update), task_id
For create: description required
For complete/update: subtask_id required`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Manage Subtasks",
			OpenWorldHint: boolPtr(false),
		},
	}, handleManageSubtasks)

	// 15. entity - Unified knowledge graph operations (NEW - replaces 10+ tools)
	mcp.AddTool(server, &mcp.Tool{
		Name: "entity",
		Description: `🟢 ADVANCED | Knowledge graph entity operations.

REQUIRED: action (list|get|create|create_rel|graph|memories|entity_memories|stats|traverse|extract)

Actions:
- list: List entities. Optional: type, project, query, limit
- get: Get entity details. Requires: entity_id
- create: Create entity. Requires: name, type (person|tool|concept|project|organization|location|event|document|api|other). Optional: description, aliases[], project, confidence
- create_rel: Create relationship. Requires: source_entity_id, target_entity_id, relationship_type. Optional: label, description, strength
- graph: Get entity graph. Requires: entity_id. Optional: hops (1-3)
- memories: Get memories for entity. Requires: entity_id. Optional: hops, limit
- entity_memories: Get entities from memory. Requires: memory_id
- stats: Get knowledge graph stats
- traverse: Advanced graph traversal. Requires: start_entity_id. Optional: target_entity_id, max_depth, mode (paths|cluster|neighbors)
- extract: Preview entity extraction from content. Requires: content

Examples:
- entity(action: "list", type: "tool")
- entity(action: "graph", entity_id: "uuid", hops: 2)`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Entity",
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedEntity)

	// 16. admin - Unified administrative operations (NEW - replaces maintenance tools)
	mcp.AddTool(server, &mcp.Tool{
		Name: "admin",
		Description: `🟢 ADVANCED | Administrative and maintenance operations.

REQUIRED: action (consolidate|cleanup|orgs|switch_org|export|import|plan|analyze)

Actions:
- consolidate: Memory consolidation. Requires: project. Optional: stale_days, promote_threshold, archive_threshold
- cleanup: Clean expired memories. Optional: project, dry_run, batch_size
- orgs: List organizations
- switch_org: Switch organization. Requires: orgId
- export: Export context pack. Requires: pack_id
- import: Import context pack. Requires: bundle, project, conflict_mode
- plan: Multi-agent planning. Requires for create: requirements. Requires for status/apply/cancel: plan_id
- analyze: Analyze project files. Requires: project, files[]. Optional: auto_apply

Examples:
- admin(action: "orgs")
- admin(action: "consolidate", project: "my-project")`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Admin",
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedAdmin)
}
