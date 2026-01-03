package repository

import (
	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/pkg/models"
	"gorm.io/gorm"
)

type gormProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new GORM project repository
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &gormProjectRepository{db: db}
}

func (r *gormProjectRepository) Create(project *models.Project) error {
	return r.db.Create(project).Error
}

func (r *gormProjectRepository) GetByID(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.Where("id = ?", id).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *gormProjectRepository) GetByName(name string) (*models.Project, error) {
	var project models.Project
	err := r.db.Where("name = ?", name).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *gormProjectRepository) GetAll() ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Preload("Organization").Find(&projects).Error
	return projects, err
}

func (r *gormProjectRepository) GetByOrganizationID(orgID uuid.UUID) ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Where("organization_id = ?", orgID).Find(&projects).Error
	return projects, err
}

func (r *gormProjectRepository) Update(project *models.Project) error {
	return r.db.Save(project).Error
}

func (r *gormProjectRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Project{}, id).Error
}

func (r *gormProjectRepository) SetActive(id uuid.UUID) error {
	// First, set all projects to inactive
	if err := r.db.Model(&models.Project{}).Update("is_active", false).Error; err != nil {
		return err
	}
	// Then set the specified project as active
	return r.db.Model(&models.Project{}).Where("id = ?", id).Update("is_active", true).Error
}

func (r *gormProjectRepository) GetActive() (*models.Project, error) {
	var project models.Project
	err := r.db.Where("is_active = ?", true).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}
