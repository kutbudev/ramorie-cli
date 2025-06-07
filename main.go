package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/terzigolu/josepshbrain-go/api/config"
	"github.com/terzigolu/josepshbrain-go/api/database"
	"github.com/terzigolu/josepshbrain-go/api/handlers"
	"github.com/terzigolu/josepshbrain-go/api/repository"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/terzigolu/josepshbrain-go/api/docs" // This is required for swag to find docs
)

// @title JosephsBrain API
// @version 1.0
// @description This is the API server for the JosephsBrain CLI.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email development@terzigolu.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /v1
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.Initialize(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	projectRepo := repository.NewProjectRepository(db)

	taskHandler := handlers.NewTaskHandler(taskRepo)
	projectHandler := handlers.NewProjectHandler(projectRepo)

	r := gin.Default()

	// Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/v1")
	{
		// Task routes
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", taskHandler.ListTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.GET("/:id", taskHandler.GetTask)
			tasks.PUT("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", taskHandler.DeleteTask)
			tasks.POST("/:id/annotations", taskHandler.CreateAnnotation)

			// Task status change routes
			tasks.POST("/:id/start", taskHandler.SetTaskStatus)
			tasks.POST("/:id/done", taskHandler.SetTaskStatus)
		}

		// Project routes
		projects := v1.Group("/projects")
		{
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", projectHandler.CreateProject)
		}
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
} 