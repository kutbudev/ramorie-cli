package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
	"gorm.io/gorm"
)

// NewTaskCmd creates the task command with all subcommands
func NewTaskCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
		Long:  "Create, list, update, and manage tasks",
	}

	// Add subcommands with database
	cmd.AddCommand(newTaskCreateCmd(db))
	cmd.AddCommand(newTaskListCmd(db))
	cmd.AddCommand(newTaskStartCmd(db))
	cmd.AddCommand(newTaskDoneCmd(db))

	return cmd
}

// task create
func newTaskCreateCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "create [description]",
		Short:   "Create a new task",
		Aliases: []string{"add"},
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			description := args[0]
			
			// Get active project - require one to exist
			var project models.Project
			result := db.Where("is_active = ? AND deleted_at IS NULL", true).First(&project)
			if result.Error != nil {
				fmt.Println("❌ No active project found")
				fmt.Println("💡 Use 'jbraincli init <name>' to create a project first")
				return
			}

			// Create new task
			task := models.Task{
				ProjectID:   project.ID,
				Description: description,
				Status:      string(models.TaskStatusTODO),
				Priority:    string(models.TaskPriorityMedium),
				Progress:    0,
			}

			if err := db.Create(&task).Error; err != nil {
				log.Fatalf("Failed to create task: %v", err)
			}

			fmt.Printf("🔄 Created task: %s\n", description)
			fmt.Printf("✅ Task ID: %s\n", task.ID.String())
		},
	}
}

// task list
func newTaskListCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all tasks",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			var tasks []models.Task
			if err := db.Preload("Project").Find(&tasks).Error; err != nil {
				log.Fatalf("Failed to fetch tasks: %v", err)
			}

			if len(tasks) == 0 {
				fmt.Println("📋 No tasks found. Create one with 'jbraincli task create <description>'")
				return
			}

			fmt.Println("📋 Task List:")
			fmt.Println("┌─────────────────────────────────────────┬───────────────────────────┬──────────────┬──────────┐")
			fmt.Println("│ ID                                      │ Description               │ Status       │ Priority │")
			fmt.Println("├─────────────────────────────────────────┼───────────────────────────┼──────────────┼──────────┤")
			
			for _, task := range tasks {
				fmt.Printf("│ %-39s │ %-25s │ %-12s │ %-8s │\n", 
					task.ID.String()[:8]+"...", 
					truncateString(task.Description, 25),
					string(task.Status),
					string(task.Priority))
			}
			fmt.Println("└─────────────────────────────────────────┴───────────────────────────┴──────────────┴──────────┘")
		},
	}
}

// task start
func newTaskStartCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "start [id]",
		Short: "Start working on a task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			
			var task models.Task
			if err := db.Where("id::text LIKE ?", taskID+"%").First(&task).Error; err != nil {
				log.Fatalf("Task not found: %v", err)
			}

			task.Status = string(models.TaskStatusInProgress)
			if err := db.Save(&task).Error; err != nil {
				log.Fatalf("Failed to update task: %v", err)
			}

			fmt.Printf("▶️ Started task: %s\n", task.Description)
			fmt.Println("✅ Task status updated to IN_PROGRESS!")
		},
	}
}

// task done
func newTaskDoneCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "done [id]",
		Short: "Mark task as completed",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			
			var task models.Task
			if err := db.Where("id::text LIKE ?", taskID+"%").First(&task).Error; err != nil {
				log.Fatalf("Task not found: %v", err)
			}

			task.Status = string(models.TaskStatusCompleted)
			task.Progress = 100
			if err := db.Save(&task).Error; err != nil {
				log.Fatalf("Failed to update task: %v", err)
			}

			fmt.Printf("✅ Task completed: %s\n", task.Description)
			fmt.Println("🎉 Great job!")
		},
	}
}

 