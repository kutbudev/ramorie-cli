package models

import (
	"time"
	"gorm.io/gorm"
)

// Context bağlam modelini temsil eder
type Context struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null;uniqueIndex"`
	Description *string        `json:"description" gorm:"type:text"`
	Filter      *string        `json:"filter" gorm:"type:text"`
	IsActive    bool           `json:"is_active" gorm:"default:false"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Tasks       []Task         `json:"tasks,omitempty" gorm:"foreignKey:ContextID"`
}

// Tag etiket modelini temsil eder
type Tag struct {
	ID        string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Color     *string        `json:"color" gorm:"type:varchar(7)"` // hex color
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName GORM için tablo adını belirtir
func (Context) TableName() string {
	return "contexts"
}

func (Tag) TableName() string {
	return "tags"
} 