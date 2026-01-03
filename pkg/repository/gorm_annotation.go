package repository

import (
	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/pkg/models"
	"gorm.io/gorm"
)

type gormAnnotationRepository struct {
	db *gorm.DB
}

// NewAnnotationRepository creates a new GORM annotation repository
func NewAnnotationRepository(db *gorm.DB) AnnotationRepository {
	return &gormAnnotationRepository{db: db}
}

func (r *gormAnnotationRepository) Create(annotation *models.Annotation) error {
	return r.db.Create(annotation).Error
}

func (r *gormAnnotationRepository) GetByID(id uuid.UUID) (*models.Annotation, error) {
	var annotation models.Annotation
	err := r.db.Where("id = ?", id).First(&annotation).Error
	if err != nil {
		return nil, err
	}
	return &annotation, nil
}

func (r *gormAnnotationRepository) GetByTaskID(taskID uuid.UUID) ([]models.Annotation, error) {
	var annotations []models.Annotation
	err := r.db.Where("task_id = ?", taskID).Order("created_at DESC").Find(&annotations).Error
	return annotations, err
}

func (r *gormAnnotationRepository) GetAll() ([]models.Annotation, error) {
	var annotations []models.Annotation
	err := r.db.Order("created_at DESC").Find(&annotations).Error
	return annotations, err
}

func (r *gormAnnotationRepository) Update(annotation *models.Annotation) error {
	return r.db.Save(annotation).Error
}

func (r *gormAnnotationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Annotation{}, id).Error
}
