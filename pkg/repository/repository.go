package repository

import (
	"gorm.io/gorm"
)

// NewRepository creates a new repository with all sub-repositories
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		Project:    NewProjectRepository(db),
		Task:       NewTaskRepository(db),
		Memory:     NewMemoryRepository(db),
		Context:    NewContextRepository(db),
		Tag:        NewTagRepository(db),
		Annotation: NewAnnotationRepository(db),
	}
}
