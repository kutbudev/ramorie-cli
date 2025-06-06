package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/terzigolu/josepshbrain-go/api/database"
	"github.com/terzigolu/josepshbrain-go/api/models"
)

// CreateProjectInput DTO for creating a new project
type CreateProjectInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateProject creates a new project.
func CreateProject(c *gin.Context) {
	var input CreateProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := models.Project{
		Name:        input.Name,
		Description: &input.Description,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// ListProjects retrieves all projects.
func ListProjects(c *gin.Context) {
	var projects []models.Project
	if err := database.DB.Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}

	c.JSON(http.StatusOK, projects)
} 