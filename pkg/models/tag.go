package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tag represents a tag in the system
type Tag struct {
	ID       uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name     string         `json:"name" gorm:"not null;unique;index:idx_tags_name"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Many-to-Many Relations
	Tasks    []*Task   `json:"tasks,omitempty" gorm:"many2many:task_tags"`
	Memories []*Memory `json:"memories,omitempty" gorm:"many2many:memory_item_tags"`
} 