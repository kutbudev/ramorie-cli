package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
	PriorityHigh   Priority = "H"
	PriorityMedium Priority = "M"
	PriorityLow    Priority = "L"
)

// Task görev modelini temsil eder
type Task struct {
	ID          string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title       string         `gorm:"not null" json:"title"`
	Description string         `json:"description"`
	Status      TaskStatus     `gorm:"type:varchar(20);not null;default:'TODO'" json:"status"`
	Priority    Priority       `gorm:"type:char(1);default:'M'" json:"priority"`
	Progress    int            `gorm:"default:0" json:"progress"`
	ProjectID   string         `gorm:"type:uuid;not null" json:"project_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	Annotations []Annotation   `gorm:"foreignkey:TaskID" json:"annotations"`
	Tags        datatypes.JSON `json:"tags" swaggertype:"object,string" example:"{\\\"key\\\":\\\"value\\\"}"`
	
	// Foreign key relationships
	Project     *Project       `json:"-" gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" swaggerignore:"true"`
	
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Annotation represents a task annotation in the database
type Annotation struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	TaskID    string    `json:"task_id" gorm:"type:uuid;not null"`
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

// Data Transfer Objects (DTOs) for API interactions

// CreateTaskDTO defines the structure for creating a new task.
type CreateTaskDTO struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	ProjectID   string `json:"project_id" binding:"required"`
	Priority    string `json:"priority"` // Will be normalized (e.g., "High", "H")
}

// UpdateTaskDTO defines the structure for updating an existing task.
type UpdateTaskDTO struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	Progress    *int    `json:"progress"`
}

// CreateAnnotationDTO defines the structure for creating a new annotation.
type CreateAnnotationDTO struct {
	Content string `json:"content" binding:"required"`
	TaskID  string `json:"-"` // Ignored in JSON, set from URL param
} 