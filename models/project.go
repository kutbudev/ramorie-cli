package models

import (
	"time"
	"gorm.io/gorm"
)

// Project proje modelini temsil eder
type Project struct {
	ID            string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null;uniqueIndex"`
	Description   *string        `json:"description" gorm:"type:text"`
	Path          *string        `json:"path" gorm:"type:varchar(500)"`
	IsActive      bool           `json:"is_active" gorm:"default:false"`
	Configuration *string        `json:"configuration" gorm:"type:jsonb"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Tasks         []Task         `json:"tasks,omitempty" gorm:"foreignKey:ProjectID"`
	Memories      []Memory       `json:"memories,omitempty" gorm:"foreignKey:ProjectID"`
}

// TableName GORM için tablo adını belirtir
func (Project) TableName() string {
	return "projects"
} 