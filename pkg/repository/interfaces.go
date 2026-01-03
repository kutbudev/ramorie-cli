package repository

import (
	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/pkg/models"
)

// ProjectRepository defines the interface for project operations
type ProjectRepository interface {
	Create(project *models.Project) error
	GetByID(id uuid.UUID) (*models.Project, error)
	GetByName(name string) (*models.Project, error)
	GetAll() ([]models.Project, error)
	GetByOrganizationID(orgID uuid.UUID) ([]models.Project, error)
	Update(project *models.Project) error
	Delete(id uuid.UUID) error
	SetActive(id uuid.UUID) error
	GetActive() (*models.Project, error)
}

// TaskRepository defines the interface for task operations
type TaskRepository interface {
	Create(task *models.Task) error
	GetByID(id uuid.UUID) (*models.Task, error)
	GetByProjectID(projectID uuid.UUID) ([]models.Task, error)
	GetAll() ([]models.Task, error)
	Update(task *models.Task) error
	Delete(id uuid.UUID) error
	GetByStatus(status models.TaskStatus) ([]models.Task, error)
	GetByPriority(priority models.TaskPriority) ([]models.Task, error)
	GetByContext(contextID uuid.UUID) ([]models.Task, error)
	GetByTags(tags []string) ([]models.Task, error)
	Search(query string) ([]models.Task, error)
}

// MemoryRepository defines the interface for memory operations
type MemoryRepository interface {
	Create(memory *models.Memory) error
	GetByID(id uuid.UUID) (*models.Memory, error)
	GetByProjectID(projectID uuid.UUID) ([]models.Memory, error)
	GetAll() ([]models.Memory, error)
	Update(memory *models.Memory) error
	Delete(id uuid.UUID) error
	Search(query string) ([]models.Memory, error)
	GetByTags(tags []string) ([]models.Memory, error)
}

// ContextRepository defines the interface for context operations
type ContextRepository interface {
	Create(context *models.Context) error
	GetByID(id uuid.UUID) (*models.Context, error)
	GetByName(name string) (*models.Context, error)
	GetByProjectID(projectID uuid.UUID) ([]models.Context, error)
	GetAll() ([]models.Context, error)
	Update(context *models.Context) error
	Delete(id uuid.UUID) error
}

// TagRepository defines the interface for tag operations
type TagRepository interface {
	Create(tag *models.Tag) error
	GetByID(id uuid.UUID) (*models.Tag, error)
	GetByName(name string) (*models.Tag, error)
	GetAll() ([]models.Tag, error)
	Update(tag *models.Tag) error
	Delete(id uuid.UUID) error
	GetOrCreate(name string) (*models.Tag, error)
}

// AnnotationRepository defines the interface for annotation operations
type AnnotationRepository interface {
	Create(annotation *models.Annotation) error
	GetByID(id uuid.UUID) (*models.Annotation, error)
	GetByTaskID(taskID uuid.UUID) ([]models.Annotation, error)
	GetAll() ([]models.Annotation, error)
	Update(annotation *models.Annotation) error
	Delete(id uuid.UUID) error
}

// OrganizationRepository defines the interface for organization operations
type OrganizationRepository interface {
	Create(org *models.Organization) error
	GetByID(id uuid.UUID) (*models.Organization, error)
	GetBySlug(slug string) (*models.Organization, error)
	GetByUserID(userID uuid.UUID) ([]models.Organization, error)
	GetAll() ([]models.Organization, error)
	Update(org *models.Organization) error
	Delete(id uuid.UUID) error
	AddMember(orgID, userID uuid.UUID, role models.OrganizationRole) error
	RemoveMember(orgID, userID uuid.UUID) error
	GetMembers(orgID uuid.UUID) ([]models.OrganizationMember, error)
	UpdateMemberRole(orgID, userID uuid.UUID, role models.OrganizationRole) error
}

// Repository aggregates all repository interfaces
type Repository struct {
	Project      ProjectRepository
	Task         TaskRepository
	Memory       MemoryRepository
	Context      ContextRepository
	Tag          TagRepository
	Annotation   AnnotationRepository
	Organization OrganizationRepository
}
