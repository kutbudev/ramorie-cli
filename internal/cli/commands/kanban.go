package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/urfave/cli/v2"
)

// NewKanbanCmd creates the kanban command using urfave/cli.
func NewKanbanCmd() *cli.Command {
	return &cli.Command{
		Name:  "kanban",
		Usage: "Display tasks in a kanban board view",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID",
			},
		},
		Action: func(c *cli.Context) error {
			projectID := c.String("project")
			cfg, err := config.LoadConfig()
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			if projectID == "" && cfg.ActiveProjectID != "" {
				projectID = cfg.ActiveProjectID
			}

			client := api.NewClient()

			todoTasks, err := client.ListTasks(projectID, "TODO")
			if err != nil {
				return fmt.Errorf("error fetching TODO tasks: %w", err)
			}

			inProgressTasks, err := client.ListTasks(projectID, "IN_PROGRESS")
			if err != nil {
				return fmt.Errorf("error fetching IN_PROGRESS tasks: %w", err)
			}

			completedTasks, err := client.ListTasks(projectID, "COMPLETED")
			if err != nil {
				return fmt.Errorf("error fetching COMPLETED tasks: %w", err)
			}

			displayKanbanBoard(todoTasks, inProgressTasks, completedTasks)
			return nil
		},
	}
}

func displayKanbanBoard(todoTasks, inProgressTasks, completedTasks []models.Task) {
	fmt.Println("📋 Task Kanban Board")
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Println()

	colWidth := 25

	fmt.Printf("%-*s | %-*s | %-*s\n", colWidth, "📝 TODO", colWidth, "🚀 IN PROGRESS", colWidth, "✅ COMPLETED")
	fmt.Printf("%s-+-%s-+-%s\n",
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth))

	maxRows := max(len(todoTasks), len(inProgressTasks), len(completedTasks))

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
