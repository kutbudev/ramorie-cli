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
	// Importance scoring
	Importance  *float64 `json:"importance,omitempty"`
	AccessCount int      `json:"access_count"`
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
