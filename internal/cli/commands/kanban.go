package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/config"
	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/models"
)

// NewKanbanCmd creates the kanban command, fully API-driven.
func NewKanbanCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "kanban",
		Short: "Display tasks in a kanban board view",
		Long:  "Show a visual overview of tasks organized by status (TODO, IN_PROGRESS, COMPLETED).",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadCliConfig()
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			// Use active project if no specific project provided
			if projectID == "" && cfg.ActiveProjectID != "" {
				projectID = cfg.ActiveProjectID
			}

			client := api.NewClient()
			
			// Get tasks for each status
			todoTasks, err := client.ListTasks(projectID, "TODO")
			if err != nil {
				fmt.Printf("Error fetching TODO tasks: %v\n", err)
				os.Exit(1)
			}

			inProgressTasks, err := client.ListTasks(projectID, "IN_PROGRESS")
			if err != nil {
				fmt.Printf("Error fetching IN_PROGRESS tasks: %v\n", err)
				os.Exit(1)
			}

			completedTasks, err := client.ListTasks(projectID, "COMPLETED")
			if err != nil {
				fmt.Printf("Error fetching COMPLETED tasks: %v\n", err)
				os.Exit(1)
			}

			// Display kanban board
			displayKanbanBoard(todoTasks, inProgressTasks, completedTasks)
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Filter by project ID")

	return cmd
}

func displayKanbanBoard(todoTasks, inProgressTasks, completedTasks []models.Task) {
	fmt.Println("📋 Task Kanban Board")
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Println()

	// Calculate column width
	colWidth := 25

	// Headers
	fmt.Printf("%-*s | %-*s | %-*s\n", colWidth, "📝 TODO", colWidth, "🚀 IN PROGRESS", colWidth, "✅ COMPLETED")
	fmt.Printf("%s-+-%s-+-%s\n", 
		strings.Repeat("-", colWidth), 
		strings.Repeat("-", colWidth), 
		strings.Repeat("-", colWidth))

	// Find max rows needed
	maxRows := max(len(todoTasks), len(inProgressTasks), len(completedTasks))

	// Display tasks row by row
	for i := 0; i < maxRows; i++ {
		todoCell := ""
		inProgressCell := ""
		completedCell := ""

		if i < len(todoTasks) {
			task := todoTasks[i]
			priority := getPriorityIcon(task.Priority)
			todoCell = fmt.Sprintf("%s %s %s", 
				priority, 
				task.ID.String()[:8], 
				truncateString(task.Title, colWidth-12))
		}

		if i < len(inProgressTasks) {
			task := inProgressTasks[i]
			priority := getPriorityIcon(task.Priority)
			inProgressCell = fmt.Sprintf("%s %s %s", 
				priority, 
				task.ID.String()[:8], 
				truncateString(task.Title, colWidth-12))
		}

		if i < len(completedTasks) {
			task := completedTasks[i]
			priority := getPriorityIcon(task.Priority)
			completedCell = fmt.Sprintf("%s %s %s", 
				priority, 
				task.ID.String()[:8], 
				truncateString(task.Title, colWidth-12))
		}

		fmt.Printf("%-*s | %-*s | %-*s\n", colWidth, todoCell, colWidth, inProgressCell, colWidth, completedCell)
	}

	fmt.Println()
	fmt.Printf("Summary: %d TODO, %d IN PROGRESS, %d COMPLETED\n", 
		len(todoTasks), len(inProgressTasks), len(completedTasks))
	
	// Show priority legend
	fmt.Println()
	fmt.Println("Priority: 🔴 High | 🟡 Medium | 🟢 Low")
}

func getPriorityIcon(priority string) string {
	switch priority {
	case "H":
		return "🔴"
	case "M":
		return "🟡"
	case "L":
		return "🟢"
	default:
		return "⚪"
	}
}

func max(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}