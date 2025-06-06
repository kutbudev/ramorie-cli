package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Project represents a project in the system
type Project struct {
	ID            uuid.UUID       `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name          string          `json:"name" gorm:"not null;unique"`
	Description   *string         `json:"description,omitempty"`
	Path          *string         `json:"path,omitempty" gorm:"size:1024"`
	IsActive      bool            `json:"is_active" gorm:"default:false"`
	Configuration datatypes.JSON  `json:"configuration,omitempty" gorm:"type:jsonb"`
	CreatedAt     time.Time       `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time       `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt     gorm.DeletedAt  `json:"-" gorm:"index"`

	// One-to-Many Relations
	Tasks    []*Task   `json:"tasks,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Memories []*Memory `json:"memories,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:SET NULL"`
} 