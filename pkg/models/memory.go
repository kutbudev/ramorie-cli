package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Memory represents the memories table
type Memory struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Content   string         `json:"content" gorm:"not null"`
	ProjectID *uuid.UUID     `json:"project_id,omitempty" gorm:"type:uuid"`
	ContextID *uuid.UUID     `json:"context_id,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time      `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// Foreign Key Relations
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:SET NULL"`
	Context *Context `json:"context,omitempty" gorm:"foreignKey:ContextID;constraint:OnDelete:SET NULL"`

	// Many-to-Many Relations
	Tags []*Tag `json:"tags,omitempty" gorm:"many2many:memory_tags"`
	Tasks []*TaskMemory `json:"tasks,omitempty" gorm:"foreignKey:MemoryID"`
}

// MemoryItem represents the memory_items table
type MemoryItem struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Content   string         `json:"content" gorm:"not null"`
	ContextID *uuid.UUID     `json:"context_id,omitempty" gorm:"type:uuid"`
	ProjectID *uuid.UUID     `json:"project_id,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time      `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Foreign Key Relations
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:SET NULL"`
	Context *Context `json:"context,omitempty" gorm:"foreignKey:ContextID;constraint:OnDelete:SET NULL"`

	// Many-to-Many Relations
	Tags []*Tag `json:"tags,omitempty" gorm:"many2many:memory_item_tags"`
	TaskLinks []*MemoryTaskLink `json:"task_links,omitempty" gorm:"foreignKey:MemoryID"`
}

// TaskMemory represents the task_memories table (memory -> task relation)
type TaskMemory struct {
	ID                    uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TaskID               uuid.UUID `json:"task_id" gorm:"not null;type:uuid;uniqueIndex:idx_task_memory"`
	MemoryID             uuid.UUID `json:"memory_id" gorm:"not null;type:uuid;uniqueIndex:idx_task_memory"`
	RelevanceScore       float32   `json:"relevance_score" gorm:"default:0"`
	RelationType         string    `json:"relation_type" gorm:"type:varchar(50);default:'similarity'"`
	RelevanceExplanation string    `json:"relevance_explanation"`
	CreatedAt            time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// Foreign Key Relations
	Task   *Task   `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Memory *Memory `json:"memory,omitempty" gorm:"foreignKey:MemoryID;constraint:OnDelete:CASCADE"`
}

// MemoryTaskLink represents the memory_task_links table (memory_item -> task relation)
type MemoryTaskLink struct {
	ID           uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TaskID       uuid.UUID `json:"task_id" gorm:"not null;type:uuid"`
	MemoryID     uuid.UUID `json:"memory_id" gorm:"not null;type:uuid"`
	Confidence   float32   `json:"confidence" gorm:"default:0"`
	RelationType string    `json:"relation_type" gorm:"default:'similarity'"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;default:now()"`

	// Foreign Key Relations
	Task       *Task       `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	MemoryItem *MemoryItem `json:"memory_item,omitempty" gorm:"foreignKey:MemoryID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (Memory) TableName() string {
	return "memories"
}

func (MemoryItem) TableName() string {
	return "memory_items"
}

func (TaskMemory) TableName() string {
	return "task_memories"
}

func (MemoryTaskLink) TableName() string {
	return "memory_task_links"
} 