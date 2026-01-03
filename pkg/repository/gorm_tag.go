package repository

import (
	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/pkg/models"
	"gorm.io/gorm"
)

type gormTagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new GORM tag repository
func NewTagRepository(db *gorm.DB) TagRepository {
	return &gormTagRepository{db: db}
}

func (r *gormTagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *gormTagRepository) GetByID(id uuid.UUID) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *gormTagRepository) GetByName(name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *gormTagRepository) GetAll() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *gormTagRepository) Update(tag *models.Tag) error {
	return r.db.Save(tag).Error
}

func (r *gormTagRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Tag{}, id).Error
}

func (r *gormTagRepository) GetOrCreate(name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("name = ?", name).First(&tag).Error
	if err == gorm.ErrRecordNotFound {
		// Tag doesn't exist, create it
		tag = models.Tag{
			ID:   uuid.New(),
			Name: name,
		}
		if err := r.db.Create(&tag).Error; err != nil {
			return nil, err
		}
		return &tag, nil
	} else if err != nil {
		return nil, err
	}
	// Tag exists, return it
	return &tag, nil
}
