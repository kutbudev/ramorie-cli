package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusTODO       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusInReview   TaskStatus = "IN_REVIEW"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
)

// TaskPriority represents the priority of a task
type TaskPriority string

const (
	TaskPriorityHigh   TaskPriority = "H"
	TaskPriorityMedium TaskPriority = "M"
	TaskPriorityLow    TaskPriority = "L"
)

// Task represents a task in the system
type Task struct {
	ID          uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ProjectID   uuid.UUID      `json:"project_id" gorm:"not null;type:uuid;index:idx_tasks_project_status"`
	ContextID   *uuid.UUID     `json:"context_id,omitempty" gorm:"type:uuid"`
	Description string         `json:"description" gorm:"not null"`
	Status      string         `json:"status" gorm:"not null;type:varchar(50)"`
	Priority    string         `json:"priority" gorm:"not null;type:varchar(1)"`
	Progress    int            `json:"progress" gorm:"default:0;check:progress >= 0 AND progress <= 100"`
	CreatedAt   time.Time      `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	DueDate     *time.Time     `json:"due_date,omitempty"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Foreign Key Relations
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Context *Context `json:"context,omitempty" gorm:"foreignKey:ContextID;constraint:OnDelete:SET NULL"`

	// One-to-Many Relations
	Annotations []*Annotation `json:"annotations,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`

	// Many-to-Many Relations
	Tags []*Tag `json:"tags,omitempty" gorm:"many2many:task_tags"`

	// Memory Relations
	Memories    []*TaskMemory     `json:"memories,omitempty" gorm:"foreignKey:TaskID"`
	MemoryLinks []*MemoryTaskLink `json:"memory_links,omitempty" gorm:"foreignKey:TaskID"`

	// Dependencies
	BlockingTasks []*Dependency `json:"blocking_tasks,omitempty" gorm:"foreignKey:BlockedTaskID"`
	BlockedTasks  []*Dependency `json:"blocked_tasks,omitempty" gorm:"foreignKey:BlockingTaskID"`
}

// Annotation represents a task annotation/note
type Annotation struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TaskID    uuid.UUID `json:"task_id" gorm:"not null;type:uuid;index:idx_annotations_task"`
	Content   string    `json:"content" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// Foreign Key Relations
	Task *Task `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
}

// Dependency represents task dependencies
type Dependency struct {
	ID             uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BlockingTaskID uuid.UUID `json:"blocking_task_id" gorm:"not null;type:uuid"`
	BlockedTaskID  uuid.UUID `json:"blocked_task_id" gorm:"not null;type:uuid"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;default:now()"`

	// Foreign Key Relations
	BlockingTask *Task `json:"blocking_task,omitempty" gorm:"foreignKey:BlockingTaskID;constraint:OnDelete:CASCADE"`
	BlockedTask  *Task `json:"blocked_task,omitempty" gorm:"foreignKey:BlockedTaskID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (Dependency) TableName() string {
	return "dependencies"
}
