package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Context represents a context/filter in the system
type Context struct {
	ID          uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name        string         `json:"name" gorm:"not null;unique"`
	Description *string        `json:"description,omitempty"`
	Filter      *string        `json:"filter,omitempty"`
	IsActive    bool           `json:"is_active" gorm:"default:false"`
	CreatedAt   time.Time      `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// One-to-Many Relations
	Tasks    []*Task   `json:"tasks,omitempty" gorm:"foreignKey:ContextID;constraint:OnDelete:SET NULL"`
	Memories []*Memory `json:"memories,omitempty" gorm:"foreignKey:ContextID;constraint:OnDelete:SET NULL"`
} 