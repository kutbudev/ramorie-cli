package repository

import (
	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/pkg/models"
	"gorm.io/gorm"
)

type gormContextRepository struct {
	db *gorm.DB
}

// NewContextRepository creates a new GORM context repository
func NewContextRepository(db *gorm.DB) ContextRepository {
	return &gormContextRepository{db: db}
}

func (r *gormContextRepository) Create(context *models.Context) error {
	return r.db.Create(context).Error
}

func (r *gormContextRepository) GetByID(id uuid.UUID) (*models.Context, error) {
	var context models.Context
	err := r.db.Where("id = ?", id).First(&context).Error
	if err != nil {
		return nil, err
	}
	return &context, nil
}

func (r *gormContextRepository) GetByName(name string) (*models.Context, error) {
	var context models.Context
	err := r.db.Where("name = ?", name).First(&context).Error
	if err != nil {
		return nil, err
	}
	return &context, nil
}

func (r *gormContextRepository) GetByProjectID(projectID uuid.UUID) ([]models.Context, error) {
	var contexts []models.Context
	err := r.db.Where("project_id = ?", projectID).Find(&contexts).Error
	return contexts, err
}

func (r *gormContextRepository) GetAll() ([]models.Context, error) {
	var contexts []models.Context
	err := r.db.Find(&contexts).Error
	return contexts, err
}

func (r *gormContextRepository) Update(context *models.Context) error {
	return r.db.Save(context).Error
}

func (r *gormContextRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Context{}, id).Error
}
