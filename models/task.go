package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskStatus görev durumunu temsil eder
type TaskStatus string

const (
	TaskStatusTODO        TaskStatus = "TODO"
	TaskStatusInProgress  TaskStatus = "IN_PROGRESS"
	TaskStatusInReview    TaskStatus = "IN_REVIEW"
	TaskStatusCompleted   TaskStatus = "COMPLETED"
)

// Priority görev önceligini temsil eder
type Priority string

const (
	PriorityHigh   Priority = "HIGH"
	PriorityMedium Priority = "MEDIUM"
	PriorityLow    Priority = "LOW"
)

// Task görev modelini temsil eder
type Task struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   string         `json:"project_id" gorm:"type:uuid;not null;index"`
	ContextID   *string        `json:"context_id" gorm:"type:uuid;index"`
	Description string         `json:"description" gorm:"type:text;not null"`
	Status      TaskStatus     `json:"status" gorm:"type:varchar(20);default:'TODO'"`
	Priority    Priority       `json:"priority" gorm:"type:varchar(10);default:'MEDIUM'"`
	Progress    int            `json:"progress" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	DueDate     *time.Time     `json:"due_date,omitempty"`
	Tags        []Tag          `json:"tags,omitempty" gorm:"many2many:task_tags;"`
	Annotations []Annotation   `json:"annotations,omitempty" gorm:"foreignKey:TaskID"`
	
	// Foreign key relationships
	Project     *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Context     *Context       `json:"context,omitempty" gorm:"foreignKey:ContextID"`
	
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Annotation represents a task annotation in the database
type Annotation struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	TaskID    uuid.UUID `json:"task_id" gorm:"type:uuid;not null"`
	Task      *Task     `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName GORM için tablo adını belirtir
func (Task) TableName() string {
	return "tasks"
}

func (Annotation) TableName() string {
	return "annotations"
} 