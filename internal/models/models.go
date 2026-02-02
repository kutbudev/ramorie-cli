package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents an organization
type Organization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Project represents a project in the system
type Project struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Configuration  map[string]interface{} `json:"configuration,omitempty"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	Organization   *Organization          `json:"organization,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Task represents a task in the system
type Task struct {
	ID          uuid.UUID    `json:"id"`
	ProjectID   uuid.UUID    `json:"project_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      string       `json:"status"`   // TODO, IN_PROGRESS, COMPLETED
	Priority    string       `json:"priority"` // L, M, H
	Tags        interface{}  `json:"tags"`     // Can be array or object from backend
	Annotations []Annotation `json:"annotations"`
	Project     *Project     `json:"project,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	// Encryption fields for zero-knowledge encryption
	EncryptedTitle       string `json:"encrypted_title,omitempty"`
	TitleNonce           string `json:"title_nonce,omitempty"`
	EncryptedDescription string `json:"encrypted_description,omitempty"`
	DescriptionNonce     string `json:"description_nonce,omitempty"`
	IsEncrypted          bool   `json:"is_encrypted"`
	// Encryption scope (personal vs organization)
	EncryptionScope string `json:"encryption_scope,omitempty"`
	EncryptionOrgID string `json:"encryption_org_id,omitempty"`
	KeyVersion      int    `json:"key_version,omitempty"`
}

// Memory represents a memory/knowledge item
type Memory struct {
	ID           uuid.UUID   `json:"id"`
	ProjectID    uuid.UUID   `json:"project_id"`
	Content      string      `json:"content"`
	Tags         interface{} `json:"tags"`                     // Can be array or object from backend
	LinkedTaskID *uuid.UUID  `json:"linked_task_id,omitempty"` // Active task it was linked to
	Project      *Project    `json:"project,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	// Memory categorization
	Type string `json:"type"` // general, decision, bug_fix, preference, pattern, reference, skill
	// Importance scoring
	Importance  *float64 `json:"importance,omitempty"`
	AccessCount int      `json:"access_count"`
	// Procedural memory fields (for type='skill')
	Trigger    *string  `json:"trigger,omitempty"`    // Conditions when this skill should be activated
	Steps      []string `json:"steps,omitempty"`      // Array of steps to follow
	Validation *string  `json:"validation,omitempty"` // How to verify the skill was applied
	// Access control fields
	Visibility string   `json:"visibility,omitempty"` // private, project, organization, public
	Readers    []string `json:"readers,omitempty"`    // User IDs with read access
	Writers    []string `json:"writers,omitempty"`    // User IDs with write access
	// Encryption fields for zero-knowledge encryption
	EncryptedContent string `json:"encrypted_content,omitempty"`
	ContentNonce     string `json:"content_nonce,omitempty"`
	IsEncrypted      bool   `json:"is_encrypted"`
	// Encryption scope (personal vs organization)
	EncryptionScope string `json:"encryption_scope,omitempty"`
	EncryptionOrgID string `json:"encryption_org_id,omitempty"`
	KeyVersion      int    `json:"key_version,omitempty"`
}

type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Annotation represents a task annotation
type Annotation struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Subtask struct {
	ID              uuid.UUID  `json:"id"`
	TaskID          uuid.UUID  `json:"task_id"`
	Description     string     `json:"description"`
	Completed       int        `json:"completed"`
	Status          string     `json:"status,omitempty"`
	Priority        string     `json:"priority,omitempty"`
	ParentSubtaskID *uuid.UUID `json:"parent_subtask_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Context represents a context in the system
type Context struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// API Response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ProjectListResponse struct {
	Success bool      `json:"success"`
	Data    []Project `json:"data"`
}

type TaskListResponse struct {
	Success bool   `json:"success"`
	Data    []Task `json:"data"`
}

type MemoryListResponse struct {
	Success bool     `json:"success"`
	Data    []Memory `json:"data"`
}

type AnnotationListResponse struct {
	Success bool         `json:"success"`
	Data    []Annotation `json:"data"`
}

// ============================================================================
// Entity & Knowledge Graph Types
// ============================================================================

// GraphEntityType represents the type of an entity
type GraphEntityType string

const (
	GraphEntityTypePerson       GraphEntityType = "person"
	GraphEntityTypeTool         GraphEntityType = "tool"
	GraphEntityTypeConcept      GraphEntityType = "concept"
	GraphEntityTypeProject      GraphEntityType = "project"
	GraphEntityTypeOrganization GraphEntityType = "organization"
	GraphEntityTypeLocation     GraphEntityType = "location"
	GraphEntityTypeEvent        GraphEntityType = "event"
	GraphEntityTypeDocument     GraphEntityType = "document"
	GraphEntityTypeAPI          GraphEntityType = "api"
	GraphEntityTypeOther        GraphEntityType = "other"
)

// RelationshipType represents the type of relationship between entities
type RelationshipType string

const (
	RelationshipUses        RelationshipType = "uses"
	RelationshipWorksOn     RelationshipType = "works_on"
	RelationshipRelatedTo   RelationshipType = "related_to"
	RelationshipDependsOn   RelationshipType = "depends_on"
	RelationshipPartOf      RelationshipType = "part_of"
	RelationshipCreatedBy   RelationshipType = "created_by"
	RelationshipBelongsTo   RelationshipType = "belongs_to"
	RelationshipConnectsTo  RelationshipType = "connects_to"
	RelationshipReplaces    RelationshipType = "replaces"
	RelationshipSimilarTo   RelationshipType = "similar_to"
	RelationshipContradicts RelationshipType = "contradicts"
	RelationshipReferences  RelationshipType = "references"
	RelationshipImplements  RelationshipType = "implements"
	RelationshipExtends     RelationshipType = "extends"
)

// Entity represents an extracted entity from memories (knowledge graph node)
type Entity struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	OrgID          *uuid.UUID             `json:"org_id,omitempty"`
	ProjectID      *uuid.UUID             `json:"project_id,omitempty"`
	Name           string                 `json:"name"`
	NormalizedName string                 `json:"normalized_name"`
	Type           GraphEntityType        `json:"type"`
	Description    *string                `json:"description,omitempty"`
	Aliases        []string               `json:"aliases,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Confidence     float64                `json:"confidence"`
	MentionCount   int                    `json:"mention_count"`
	ValidFrom      time.Time              `json:"valid_from"`
	ValidUntil     *time.Time             `json:"valid_until,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// EntityRelationship represents a relationship between two entities (knowledge graph edge)
type EntityRelationship struct {
	ID               uuid.UUID              `json:"id"`
	UserID           uuid.UUID              `json:"user_id"`
	OrgID            *uuid.UUID             `json:"org_id,omitempty"`
	SourceEntityID   uuid.UUID              `json:"source_entity_id"`
	TargetEntityID   uuid.UUID              `json:"target_entity_id"`
	RelationshipType RelationshipType       `json:"relationship_type"`
	Label            *string                `json:"label,omitempty"`
	Description      *string                `json:"description,omitempty"`
	Strength         float64                `json:"strength"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	SourceMemoryID   *uuid.UUID             `json:"source_memory_id,omitempty"`
	ValidFrom        time.Time              `json:"valid_from"`
	ValidUntil       *time.Time             `json:"valid_until,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	// Loaded relationships
	SourceEntity *Entity `json:"source_entity,omitempty"`
	TargetEntity *Entity `json:"target_entity,omitempty"`
}

// CreateEntityRequest represents the request to create a new entity
type CreateEntityRequest struct {
	Name        string          `json:"name"`
	Type        GraphEntityType `json:"type"`
	Description *string         `json:"description,omitempty"`
	Aliases     []string        `json:"aliases,omitempty"`
	ProjectID   *string         `json:"project_id,omitempty"`
	Confidence  *float64        `json:"confidence,omitempty"`
}

// CreateRelationshipRequest represents the request to create a new relationship
type CreateRelationshipRequest struct {
	SourceEntityID   string           `json:"source_entity_id"`
	TargetEntityID   string           `json:"target_entity_id"`
	RelationshipType RelationshipType `json:"relationship_type"`
	Label            *string          `json:"label,omitempty"`
	Description      *string          `json:"description,omitempty"`
	Strength         *float64         `json:"strength,omitempty"`
	SourceMemoryID   *string          `json:"source_memory_id,omitempty"`
}

// EntityListResponse represents the response from list entities endpoint
type EntityListResponse struct {
	Entities []Entity `json:"entities"`
	Total    int64    `json:"total"`
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
}

// EntityGraphResponse represents the response from get entity graph endpoint
type EntityGraphResponse struct {
	RootEntity *Entity                  `json:"root_entity"`
	Nodes      []map[string]interface{} `json:"nodes"`
	Edges      []map[string]interface{} `json:"edges"`
	Hops       int                      `json:"hops"`
	NodeCount  int                      `json:"node_count"`
	EdgeCount  int                      `json:"edge_count"`
}

// EntityRelationshipsResponse represents the response from get entity relationships endpoint
type EntityRelationshipsResponse struct {
	Entity           *Entity              `json:"entity"`
	RelationshipsOut []EntityRelationship `json:"relationships_out"`
	RelationshipsIn  []EntityRelationship `json:"relationships_in"`
}

// EntityMemoriesResponse represents the response from get entity memories endpoint
type EntityMemoriesResponse struct {
	MemoryIDs []string `json:"memory_ids"`
	EntityID  string   `json:"entity_id"`
	Hops      int      `json:"hops"`
	Total     int      `json:"total"`
}

// MemoryEntitiesResponse represents the response from get memory entities endpoint
type MemoryEntitiesResponse struct {
	Entities []Entity `json:"entities"`
	MemoryID string   `json:"memory_id"`
	Total    int      `json:"total"`
}

// EntityStatsResponse represents the response from get entity stats endpoint
type EntityStatsResponse struct {
	TotalEntities      int64            `json:"total_entities"`
	TotalRelationships int64            `json:"total_relationships"`
	EntitiesByType     map[string]int64 `json:"entities_by_type"`
}

// ExtractedEntity represents an entity extracted during memory processing
type ExtractedEntity struct {
	Name       string          `json:"name"`
	Type       GraphEntityType `json:"type"`
	Confidence float64         `json:"confidence"`
	Position   int             `json:"position"`
	Context    string          `json:"context"`
}

// ExtractedRelationship represents a relationship extracted during processing
type ExtractedRelationship struct {
	SourceName       string           `json:"source_name"`
	TargetName       string           `json:"target_name"`
	RelationshipType RelationshipType `json:"relationship_type"`
	Confidence       float64          `json:"confidence"`
}

// ExtractionResult represents the result of entity extraction
type ExtractionResult struct {
	Entities      []ExtractedEntity       `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
	UsedAI        bool                    `json:"used_ai"`
}

// ============================================================================
// Skills & Execution Types
// ============================================================================

// SkillExecution represents an execution of a procedural skill
type SkillExecution struct {
	ID             uuid.UUID  `json:"id"`
	SkillID        uuid.UUID  `json:"skill_id"`
	UserID         uuid.UUID  `json:"user_id"`
	OrgID          *uuid.UUID `json:"org_id,omitempty"`
	AgentName      *string    `json:"agent_name,omitempty"`
	AgentModel     *string    `json:"agent_model,omitempty"`
	AgentSessionID *string    `json:"agent_session_id,omitempty"`
	Context        *string    `json:"context,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Success        *bool      `json:"success,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// Loaded relationships
	Skill *Memory `json:"skill,omitempty"`
}

// SkillStats represents aggregated statistics for a skill
type SkillStats struct {
	SkillID         uuid.UUID  `json:"skill_id"`
	TotalExecutions int64      `json:"total_executions"`
	SuccessCount    int64      `json:"success_count"`
	FailureCount    int64      `json:"failure_count"`
	SuccessRate     float64    `json:"success_rate"`
	LastExecutedAt  *time.Time `json:"last_executed_at,omitempty"`
}

// GeneratedSkill represents an AI-generated skill
type GeneratedSkill struct {
	Trigger       string   `json:"trigger"`
	Description   string   `json:"description"`
	Steps         []string `json:"steps"`
	Validation    string   `json:"validation"`
	Confidence    float64  `json:"confidence"`
	SuggestedTags []string `json:"suggested_tags"`
}

// GenerateSkillResponse is the response for skill generation
type GenerateSkillResponse struct {
	Skill     GeneratedSkill `json:"skill"`
	SavedID   *uuid.UUID     `json:"saved_id,omitempty"`
	AIModel   string         `json:"ai_model"`
	LatencyMs int            `json:"latency_ms"`
}

// ============================================================================
// Project Analysis & Bootstrap Types
// ============================================================================

// FileInput represents a file to analyze
type FileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ManualProjectInput represents manual project description for analysis
type ManualProjectInput struct {
	Description string   `json:"description"`
	TechStack   []string `json:"tech_stack,omitempty"`
	Conventions string   `json:"conventions,omitempty"`
	SetupSteps  []string `json:"setup_steps,omitempty"`
}

// AnalyzeProjectRequest is the request for project analysis
type AnalyzeProjectRequest struct {
	Files       []FileInput         `json:"files,omitempty"`
	ManualInput *ManualProjectInput `json:"manual_input,omitempty"`
}

// TechStackItem represents a detected technology in the project
type TechStackItem struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Detected string `json:"detected"`
}

// SuggestedMemory represents an AI-suggested memory for bootstrap
type SuggestedMemory struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags"`
	Source     string   `json:"source"`
	Confidence float64  `json:"confidence"`
	Selected   bool     `json:"selected"`
}

// SuggestedDecision represents an AI-suggested decision/ADR for bootstrap
type SuggestedDecision struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Status       string  `json:"status"`
	Area         string  `json:"area"`
	Context      string  `json:"context,omitempty"`
	Consequences string  `json:"consequences,omitempty"`
	Source       string  `json:"source"`
	Confidence   float64 `json:"confidence"`
	Selected     bool    `json:"selected"`
}

// SuggestedTask represents an AI-suggested task for bootstrap
type SuggestedTask struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    string   `json:"priority"`
	Source      string   `json:"source"`
	Tags        []string `json:"tags"`
	Selected    bool     `json:"selected"`
}

// AnalysisResult is the response from project analysis
type AnalysisResult struct {
	ProjectID          string              `json:"project_id"`
	AnalyzedAt         string              `json:"analyzed_at"`
	Source             string              `json:"source"` // github, mcp, manual
	TechStack          []TechStackItem     `json:"tech_stack"`
	FileStructure      []string            `json:"file_structure,omitempty"`
	SuggestedMemories  []SuggestedMemory   `json:"suggested_memories"`
	SuggestedDecisions []SuggestedDecision `json:"suggested_decisions"`
	SuggestedTasks     []SuggestedTask     `json:"suggested_tasks"`
	Confidence         float64             `json:"confidence"`
	FilesAnalyzed      int                 `json:"files_analyzed"`
	AIModel            string              `json:"ai_model"`
	LatencyMs          int                 `json:"latency_ms"`
}

// BootstrapMemoryInput represents a memory to create during bootstrap
type BootstrapMemoryInput struct {
	Content    string   `json:"content"`
	Type       string   `json:"type,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Source     string   `json:"source,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
}

// BootstrapDecisionInput represents a decision to create during bootstrap
type BootstrapDecisionInput struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Status       string `json:"status,omitempty"`
	Area         string `json:"area,omitempty"`
	Context      string `json:"context,omitempty"`
	Consequences string `json:"consequences,omitempty"`
}

// BootstrapTaskInput represents a task to create during bootstrap
type BootstrapTaskInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Status      string   `json:"status,omitempty"`
	Source      string   `json:"source,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// BootstrapProjectRequest is the request for project bootstrapping
type BootstrapProjectRequest struct {
	Memories          []BootstrapMemoryInput   `json:"memories"`
	Decisions         []BootstrapDecisionInput `json:"decisions"`
	Tasks             []BootstrapTaskInput     `json:"tasks"`
	CreateContextPack bool                     `json:"create_context_pack,omitempty"`
	MarkAsOnboarded   bool                     `json:"mark_as_onboarded,omitempty"`
}

// BootstrapResult is the response from project bootstrapping
type BootstrapResult struct {
	ProjectID     string `json:"project_id"`
	ContextPackID string `json:"context_pack_id,omitempty"`
	Created       struct {
		Memories  int `json:"memories"`
		Decisions int `json:"decisions"`
		Tasks     int `json:"tasks"`
	} `json:"created"`
	Errors []string `json:"errors,omitempty"`
}
