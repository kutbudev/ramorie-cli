package models

import (
	"time"
	"gorm.io/gorm"
)

// Memory hafıza modelini temsil eder
type Memory struct {
	ID          string         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Content     string         `json:"content" gorm:"type:text;not null"`
	ProjectID   string         `json:"project_id" gorm:"type:uuid;not null;index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Project     *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Tags        []Tag          `json:"tags,omitempty" gorm:"many2many:memory_item_tags;"`
	TaskLinks   []MemoryTaskLink `json:"task_links,omitempty" gorm:"foreignKey:MemoryID"`
}

// MemoryTaskLink görev-hafıza ilişkisini temsil eder
type MemoryTaskLink struct {
	ID           string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID       string    `json:"task_id" gorm:"type:uuid;not null;index"`
	MemoryID     string    `json:"memory_id" gorm:"type:uuid;not null;index"`
	Score        float64   `json:"score" gorm:"type:decimal(5,4);default:0.0"`
	RelationType string    `json:"relation_type" gorm:"type:varchar(50);default:'SEMANTIC'"`
	Explanation  *string   `json:"explanation" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
	
	// Relationships
	Task         *Task     `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Memory       *Memory   `json:"memory,omitempty" gorm:"foreignKey:MemoryID"`
}

// TableName GORM için tablo adını belirtir
func (Memory) TableName() string {
	return "memory_items"
}

func (MemoryTaskLink) TableName() string {
	return "memory_task_links"
} 