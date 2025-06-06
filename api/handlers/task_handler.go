package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/terzigolu/josepshbrain-go/api/database"
	"github.com/terzigolu/josepshbrain-go/api/models"
)

// CreateTaskInput DTO for creating a new task
type CreateTaskInput struct {
	Description string `json:"description" binding:"required"`
	ProjectID   string `json:"project_id" binding:"required"`
	Priority    string `json:"priority"`
}

// CreateTask creates a new task in the database.
func CreateTask(c *gin.Context) {
	var input CreateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := models.Task{
		Description: input.Description,
		ProjectID:   input.ProjectID,
		Status:      models.TaskStatusTODO,
		Priority:    NormalizePriority(input.Priority),
	}

	if input.Priority == "" {
		task.Priority = models.PriorityMedium
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// GetTask retrieves a single task by its ID.
func GetTask(c *gin.Context) {
	id := c.Param("id")
	var task models.Task

	if err := database.DB.Preload("Annotations").First(&task, "id::text LIKE ?", id+"%").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTaskInput DTO for updating a task
type UpdateTaskInput struct {
	Description *string `json:"description"`
	Priority    *string `json:"priority"`
	Progress    *int    `json:"progress"`
}

// UpdateTask updates an existing task.
func UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var task models.Task
	if err := database.DB.First(&task, "id::text LIKE ?", id+"%").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	var input UpdateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Priority != nil {
		task.Priority = NormalizePriority(*input.Priority)
	}
	if input.Progress != nil {
		task.Progress = *input.Progress
	}

	if err := database.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask deletes a task from the database.
func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	var task models.Task
	if err := database.DB.First(&task, "id::text LIKE ?", id+"%").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if err := database.DB.Delete(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

// SetTaskStatusInput DTO for updating a task's status
type SetTaskStatusInput struct {
	Status models.TaskStatus `json:"status" binding:"required"`
}

// SetTaskStatus updates the status of a task.
func SetTaskStatus(c *gin.Context) {
	id := c.Param("id")
	var task models.Task
	if err := database.DB.First(&task, "id::text LIKE ?", id+"%").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	var input SetTaskStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task.Status = input.Status
	// Optionally update completed_at or started_at timestamps
	if input.Status == models.TaskStatusCompleted {
		now := time.Now()
		task.CompletedAt = &now
	}

	if err := database.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task status"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// CreateAnnotationInput DTO for adding an annotation
type CreateAnnotationInput struct {
	Content string `json:"content" binding:"required"`
}

// CreateAnnotation adds a new annotation to a task.
func CreateAnnotation(c *gin.Context) {
	taskID := c.Param("id")
	var task models.Task
	if err := database.DB.First(&task, "id::text LIKE ?", taskID+"%").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	var input CreateAnnotationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	annotation := models.Annotation{
		TaskID:  task.ID,
		Content: input.Content,
	}

	if err := database.DB.Create(&annotation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create annotation"})
		return
	}

	c.JSON(http.StatusCreated, annotation)
}

// ListTasks retrieves all tasks from the database.
func ListTasks(c *gin.Context) {
	var tasks []models.Task
	if err := database.DB.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
} 