package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/terzigolu/josepshbrain-go/api/models"
	"github.com/terzigolu/josepshbrain-go/api/repository"
	"github.com/terzigolu/josepshbrain-go/api/utils"
)

type TaskHandler struct {
	repo repository.TaskRepository
}

func NewTaskHandler(repo repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// ListTasks godoc
// @Summary List all tasks
// @Description Get a list of all tasks, optionally filtered by status.
// @Tags tasks
// @Produce json
// @Param status query string false "Filter by status (e.g., 'TODO', 'IN_PROGRESS')"
// @Success 200 {array} models.Task
// @Router /tasks [get]
func (h *TaskHandler) ListTasks(c *gin.Context) {
	status := c.Query("status")
	tasks, err := h.repo.GetTasks(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// CreateTask godoc
// @Summary Create a new task
// @Description Creates a new task with the provided data.
// @Tags tasks
// @Accept json
// @Produce json
// @Param task body models.CreateTaskDTO true "Task to create"
// @Success 201 {object} models.Task
// @Router /tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var createTaskDTO models.CreateTaskDTO
	if err := c.ShouldBindJSON(&createTaskDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if createTaskDTO.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title cannot be empty"})
		return
	}

	task, err := h.repo.CreateTask(createTaskDTO)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")

	task, err := h.repo.GetTaskByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id := c.Param("id")

	var taskUpdates models.UpdateTaskDTO
	if err := c.ShouldBindJSON(&taskUpdates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedTask, err := h.repo.UpdateTask(id, taskUpdates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		}
		return
	}

	c.JSON(http.StatusOK, updatedTask)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id := c.Param("id")

	err := h.repo.DeleteTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *TaskHandler) CreateAnnotation(c *gin.Context) {
	taskID := c.Param("id")

	var annotationDTO models.CreateAnnotationDTO
	if err := c.ShouldBindJSON(&annotationDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	annotationDTO.TaskID = taskID // Ensure association

	if annotationDTO.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Annotation content cannot be empty"})
		return
	}

	newAnnotation, err := h.repo.CreateAnnotation(annotationDTO)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create annotation"})
		return
	}

	c.JSON(http.StatusCreated, newAnnotation)
}

func (h *TaskHandler) SetTaskStatus(c *gin.Context) {
	id := c.Param("id")
	path := c.FullPath() // e.g., "/v1/tasks/:id/start"

	var newStatus string
	if strings.HasSuffix(path, "/start") {
		newStatus = string(models.TaskStatusInProgress)
	} else if strings.HasSuffix(path, "/done") {
		newStatus = string(models.TaskStatusDone)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status transition"})
		return
	}

	priority := utils.NormalizePriority(newStatus)
	task, err := h.repo.UpdateTaskStatus(id, newStatus, priority)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task status"})
		}
		return
	}

	c.JSON(http.StatusOK, task)
} 