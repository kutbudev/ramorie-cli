package repository

import (
	"strings"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/pkg/models"
	"gorm.io/gorm"
)

type gormTaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new GORM task repository
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &gormTaskRepository{db: db}
}

func (r *gormTaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *gormTaskRepository) GetByID(id uuid.UUID) (*models.Task, error) {
	var task models.Task
	err := r.db.Preload("Tags").Preload("Annotations").Preload("Dependencies").Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *gormTaskRepository) GetByProjectID(projectID uuid.UUID) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Tags").Where("project_id = ?", projectID).Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) GetAll() ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Tags").Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *gormTaskRepository) Delete(id uuid.UUID) error {
	return r.db.Select("Tags", "Annotations", "Dependencies").Delete(&models.Task{}, id).Error
}

func (r *gormTaskRepository) GetByStatus(status models.TaskStatus) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Tags").Where("status = ?", status).Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) GetByPriority(priority models.TaskPriority) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Tags").Where("priority = ?", priority).Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) GetByContext(contextID uuid.UUID) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Tags").Where("context_id = ?", contextID).Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) GetByTags(tags []string) ([]models.Task, error) {
	var tasks []models.Task
	query := r.db.Preload("Tags").
		Joins("JOIN task_tags ON tasks.id = task_tags.task_id").
		Joins("JOIN tags ON task_tags.tag_id = tags.id").
		Where("tags.name IN ?", tags).
		Group("tasks.id")

	err := query.Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) Search(query string) ([]models.Task, error) {
	var tasks []models.Task
	searchTerm := "%" + strings.ToLower(query) + "%"

	err := r.db.Preload("Tags").Where(
		"LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
		searchTerm, searchTerm,
	).Find(&tasks).Error

	return tasks, err
}
