package repository

import (
	"strings"

	"github.com/google/uuid"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
	"gorm.io/gorm"
)

type gormMemoryRepository struct {
	db *gorm.DB
}

// NewMemoryRepository creates a new GORM memory repository
func NewMemoryRepository(db *gorm.DB) MemoryRepository {
	return &gormMemoryRepository{db: db}
}

func (r *gormMemoryRepository) Create(memory *models.Memory) error {
	return r.db.Create(memory).Error
}

func (r *gormMemoryRepository) GetByID(id uuid.UUID) (*models.Memory, error) {
	var memory models.Memory
	err := r.db.Preload("Tags").Where("id = ?", id).First(&memory).Error
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

func (r *gormMemoryRepository) GetByProjectID(projectID uuid.UUID) ([]models.Memory, error) {
	var memories []models.Memory
	err := r.db.Preload("Tags").Where("project_id = ?", projectID).Order("created_at DESC").Find(&memories).Error
	return memories, err
}

func (r *gormMemoryRepository) GetAll() ([]models.Memory, error) {
	var memories []models.Memory
	err := r.db.Preload("Tags").Order("created_at DESC").Find(&memories).Error
	return memories, err
}

func (r *gormMemoryRepository) Update(memory *models.Memory) error {
	return r.db.Save(memory).Error
}

func (r *gormMemoryRepository) Delete(id uuid.UUID) error {
	return r.db.Select("Tags").Delete(&models.Memory{}, id).Error
}

func (r *gormMemoryRepository) Search(query string) ([]models.Memory, error) {
	var memories []models.Memory
	searchTerm := "%" + strings.ToLower(query) + "%"

	err := r.db.Preload("Tags").Where(
		"LOWER(content) LIKE ? OR LOWER(title) LIKE ?",
		searchTerm, searchTerm,
	).Order("created_at DESC").Find(&memories).Error

	return memories, err
}

func (r *gormMemoryRepository) GetByTags(tags []string) ([]models.Memory, error) {
	var memories []models.Memory
	query := r.db.Preload("Tags").
		Joins("JOIN memory_tags ON memories.id = memory_tags.memory_id").
		Joins("JOIN tags ON memory_tags.tag_id = tags.id").
		Where("tags.name IN ?", tags).
		Group("memories.id").
		Order("memories.created_at DESC")

	err := query.Find(&memories).Error
	return memories, err
}
