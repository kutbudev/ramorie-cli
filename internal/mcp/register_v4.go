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
		Name: "setup_agent",
		Description: `🔴 ESSENTIAL | Initialize the agent session. ⚠️ CALL THIS FIRST every conversation.

Returns a compact session context (~500 tokens):
- session info + agent tracking identity
- project auto-detected from cwd (git remote / dir name)
- top-5 active user preferences (surfaced so you don't have to remember them)
- task stats + project count
- next_action nudge based on current state

OPTIONAL:
- agent_name (string) — tracking label (e.g. "claude-code", "cursor")
- agent_model (string) — model identifier
- full (bool) — verbose payload with context_injection + recommended_actions. Default false.

Durable operating directives live in the server's instructions block (loaded once per session).`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Initialize Agent Session",
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, handleSetupAgent)

	// 2. list_projects - List accessible projects (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_projects",
		Description: `🔴 ESSENTIAL | List ALL accessible projects (personal + organization-scoped).

Returns a compact shape: [{id, name, org?}].

OPTIONAL:
- verbose (bool) — include full nested organization metadata, timestamps, description. Default false.`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Projects",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleListProjects)

	// 3. remember - Store memories (KEEP)
	mcp.AddTool(server, &mcp.Tool{
		Name: "remember",
		Description: `🔴 ESSENTIAL | Store a memory (durable fact / decision / pattern / preference).

REQUIRED: content, project (name or ID)
OPTIONAL: force (bool) — skip similarity check and save anyway

Type is auto-detected from content words:
- "decided X" / "chose X"  → decision
- "fixed bug" / "root cause" → bug_fix
- "prefer X" / "always" / "never" → preference
- "skill:" / "how to"      → skill

TASK vs MEMORY promotion:
Content that STARTS with one of these prefixes is promoted to a task instead:
  todo: | TODO: | later: | task: | action: | reminder: | followup:
Middle-of-sentence mentions do NOT promote ("fixed 2 TODOs" stays as memory).
To force task creation, explicitly open the content with "todo:" (or call task(action=create)).

Duplicate prevention: if cosine similarity to an existing memory is >0.9, a warning
is returned with the existing memory instead of creating a duplicate. Override with force=true.

Best practice: find(term) BEFORE remember() to check existing knowledge. The backend also
runs a post-write contradiction check — if your new memory supersedes an older one, the
older is marked superseded and hidden from default searches.

Example:
  remember(content: "API uses JWT auth stored in httpOnly cookie", project: "my-project")
  remember(content: "todo: document the retry policy", project: "my-project")   // → task`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Remember",
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handleRemember)

	// 4. find - Hybrid search (semantic + lexical) — preferred in v4.
	mcp.AddTool(server, &mcp.Tool{
		Name: "find",
		Description: `🔴 ESSENTIAL | Hybrid memory + decision retrieval — preferred over recall().

Under the hood the server runs a multi-stage pipeline and picks what to apply per query:
  1. HyDE — query is rewritten into a hypothetical answer first, then embedded (catches queries that share no keywords with the memory)
  2. Hybrid scan — pgvector cosine + PostgreSQL FTS + recency + access_count, weighted
  3. Entity graph bonus — memories linked to entities matching your query get a small lift
  4. Propositional boost — long memories are split into atomic claims; the best-matching claim boosts its parent
  5. Intent routing — "how do I X" narrows to skill/pattern; "why did we X" auto-includes decisions; "recent X" favors fresh content
  6. Gemini rerank — top candidates are re-scored by an LLM for pair-wise relevance
  7. Supersede filter — memories marked as out-of-date by a newer contradicting memory are hidden by default

All of the above are ON by default with graceful fallback if a stage fails — agents
rarely need to tune these flags. Override only when you have a specific reason.

REQUIRED: term

OPTIONAL (common):
- project — name or UUID. Omit to auto-scope via cwd.
- types ([]string) — general | decision | bug_fix | preference | pattern | reference | skill
- tags ([]string)
- limit (default 5, max 50)
- budget_tokens (default 2000) — response trimmed to fit
- include_decisions (bool, default true)
- purpose ("coding" | "research" | "review") — type-preference nudge

OPTIONAL (tuning):
- hyde ("on" | "off" | "default") — query expansion. Off = benchmark raw embedding.
- rerank ("on" | "off" | "default") — LLM reranker. Off = keep hybrid order.
- intent ("auto" | "how_to" | "why" | "recent" | "owner" | "generic") — pin intent.
- entity_hops (0-3) — multi-hop entity expansion. 0 = direct matches only.
- include_superseded (bool) — show memories marked superseded (audit/debug).

Returns: {items[], _meta: {total, intent, hyde_used, rerank_used, ranking_mode, latency_ms, ...}}
Each item: {id, type, title, preview, score, breakdown, access_count, project}

Examples:
  find(term: "RTK query cache invalidation")
  find(term: "why did we migrate from Material UI", include_decisions: true)
  find(term: "yarn rule", types: ["preference"])
  find(term: "ui framework", include_superseded: true)   // see history
  find(term: "bootstrap auth flow", hyde: "off", rerank: "off")  // raw hybrid`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Find",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handleFind)

	// 4b. recall - legacy keyword search (KEEP for backwards compat).
	mcp.AddTool(server, &mcp.Tool{
		Name: "recall",
		Description: `🟡 LEGACY | Full-text ts_rank search — kept for backwards compatibility, will be removed in v5.

⚠️ Prefer find(): find() wraps the same query set but adds HyDE expansion, Gemini rerank,
propositional boost, entity-graph bonus, intent routing, and supersede filtering on top.
recall() only does lexical ts_rank — you will miss results that don't share keywords with
your query even when they're semantically spot-on.

Use recall() only when (a) you need to replicate exact pre-v4 agent behavior, or
(b) you're benchmarking search quality and want a lexical-only baseline.

REQUIRED: term
Optional: project, tag, type, purpose, min_score, limit, include_decisions (default true)`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Recall (legacy — prefer find)",
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

	// 12. surface_context - File/domain/pattern-scoped context surfacing
	mcp.AddTool(server, &mcp.Tool{
		Name: "surface_context",
		Description: `🟡 COMMON | Pull relevant decisions + memories based on which FILES you're about to edit
(not a natural-language query — that's find()).

When to use which:
- find(term) — when you have a concrete question or topic ("how do I invalidate the RTK cache")
- surface_context — when you're opening a file and want to see what's been decided/learned
  about that module, even before you know what to ask

The call signature accepts file paths, directory/domain names, or code patterns you're about
to use. The server maps those to search terms, finds matching decisions + bug_fix + pattern
memories, and returns a compact list of "things you should know before editing here".

REQUIRED (at least one):
- file_paths ([]string) — "src/store/api/userApi.ts"
- domains ([]string)    — "admin", "auth", "api"
- code_patterns ([]string) — "fetch(", "useEffect", "dangerouslySetInnerHTML"

OPTIONAL:
- project — scope override (cwd auto-detect otherwise)
- purpose ("coding" | "research" | "review") — type-boost nudge
- limit (default 10)

Returns memories and decisions ranked by relevance to the current work scope.`,
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
		Description: `🟢 ADVANCED | Administrative + maintenance operations.

REQUIRED: action (consolidate|cleanup|orgs|switch_org|export|import|plan|analyze)

Actions:
- consolidate: Run memory consolidation. Requires: project.
  Optional: stale_days, promote_threshold, archive_threshold,
            mode ("score" default | "merge" | "both"), dry_run (preview merge clusters),
            cluster_threshold (default 0.92 cosine).
  "merge" requires FEATURE_MERGE_ENABLED=true on the server — it's off-by-default because
  merging is irreversible. Use dry_run first to preview.
- cleanup: Clean expired / TTL'd memories. Optional: project, dry_run, batch_size
- orgs: List accessible organizations
- switch_org: Switch active organization. Requires: orgId
- export: Export context pack. Requires: pack_id
- import: Import context pack. Requires: bundle, project, conflict_mode (skip|overwrite|rename)
- plan: Multi-agent planning. For create: requirements. For status/apply/cancel: plan_id
- analyze: Analyze project files. Requires: project, files[]. Optional: auto_apply

Examples:
  admin(action: "orgs")
  admin(action: "consolidate", project: "my-project", mode: "merge", dry_run: true)`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Admin",
			OpenWorldHint: boolPtr(false),
		},
	}, handleUnifiedAdmin)
}
