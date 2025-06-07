package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/config"
	"github.com/terzigolu/josepshbrain-go/internal/api"
)

// NewTaskCmd creates the task command with subcommands, fully API-driven.
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
		Long:  "Manage tasks with create, list, info, start, done, and delete operations.",
	}

	cmd.AddCommand(newTaskCreateCmd())
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskInfoCmd())
	cmd.AddCommand(newTaskStartCmd())
	cmd.AddCommand(newTaskDoneCmd())
	cmd.AddCommand(newTaskDeleteCmd())

	return cmd
}

// task create
func newTaskCreateCmd() *cobra.Command {
	var priority string
	var description string
	var tags []string

	cmd := &cobra.Command{
		Use:   "create [title]",
		Short: "Create a new task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			title := args[0]

			cfg, err := config.LoadCliConfig()
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			if cfg.ActiveProjectID == "" {
				fmt.Println("No active project set. Use 'jbraincli project use <name>' to set an active project.")
				os.Exit(1)
			}

			client := api.NewClient()
			task, err := client.CreateTask(cfg.ActiveProjectID, title, description, priority, tags)
			if err != nil {
				fmt.Printf("Error creating task: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Task created successfully!\n")
			fmt.Printf("ID: %s\n", task.ID.String())
			fmt.Printf("Title: %s\n", task.Title)
			fmt.Printf("Status: %s\n", task.Status)
			fmt.Printf("Priority: %s\n", task.Priority)
			if task.Project != nil {
				fmt.Printf("Project: %s (%s)\n", task.Project.Name, task.ProjectID.String()[:8])
			}
		},
	}

	cmd.Flags().StringVarP(&priority, "priority", "p", "M", "Task priority (L, M, H)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Task description")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Task tags (comma-separated)")

	return cmd
}

// task list
func newTaskListCmd() *cobra.Command {
	var status string
	var projectID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
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
			tasks, err := client.ListTasks(projectID, status)
			if err != nil {
				fmt.Printf("Error listing tasks: %v\n", err)
				os.Exit(1)
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPRIORITY\tCREATED")
			fmt.Fprintln(w, "--\t-----\t------\t--------\t-------")

			for _, task := range tasks {
				shortID := task.ID.String()[:8]
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					shortID,
					truncateString(task.Title, 30),
					task.Status,
					task.Priority,
					task.CreatedAt.Format("2006-01-02"))
			}
			w.Flush()
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (TODO, IN_PROGRESS, COMPLETED)")
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Filter by project ID")

	return cmd
}

// task info
func newTaskInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [task-id]",
		Short: "Show detailed task information",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			client := api.NewClient()
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Printf("Error getting task: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("📋 Task Details\n")
			fmt.Printf("================\n")
			fmt.Printf("ID: %s\n", task.ID.String())
			fmt.Printf("Title: %s\n", task.Title)
			fmt.Printf("Description: %s\n", task.Description)
			fmt.Printf("Status: %s\n", task.Status)
			fmt.Printf("Priority: %s\n", task.Priority)
			if task.Project != nil {
				fmt.Printf("Project: %s (%s)\n", task.Project.Name, task.ProjectID.String()[:8])
			} else {
				fmt.Printf("Project ID: %s\n", task.ProjectID.String())
			}
			if len(task.Tags) > 0 {
				fmt.Printf("Tags: %s\n", strings.Join(task.Tags, ", "))
			}
			fmt.Printf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
			
			// Display annotations if any exist
			if len(task.Annotations) > 0 {
				fmt.Printf("\n📝 Annotations (%d)\n", len(task.Annotations))
				fmt.Printf("==================\n")
				for i, annotation := range task.Annotations {
					fmt.Printf("%d. %s\n", i+1, annotation.Content)
					fmt.Printf("   Created: %s\n", annotation.CreatedAt.Format("2006-01-02 15:04:05"))
					if i < len(task.Annotations)-1 {
						fmt.Printf("\n")
					}
				}
			}
		},
	}

	return cmd
}

// task start
func newTaskStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [task-id]",
		Short: "Start a task (set status to IN_PROGRESS)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			client := api.NewClient()
			task, err := client.UpdateTaskStatus(taskID, "IN_PROGRESS")
			if err != nil {
				fmt.Printf("Error starting task: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🚀 Task started!\n")
			fmt.Printf("ID: %s\n", task.ID.String())
			fmt.Printf("Title: %s\n", task.Title)
			fmt.Printf("Status: %s\n", task.Status)
			if task.Project != nil {
				fmt.Printf("Project: %s (%s)\n", task.Project.Name, task.ProjectID.String()[:8])
			}
		},
	}

	return cmd
}

// task done
func newTaskDoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done [task-id]",
		Short: "Mark a task as completed",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			client := api.NewClient()
			task, err := client.UpdateTaskStatus(taskID, "COMPLETED")
			if err != nil {
				fmt.Printf("Error completing task: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Task completed!\n")
			fmt.Printf("ID: %s\n", task.ID.String())
			fmt.Printf("Title: %s\n", task.Title)
			fmt.Printf("Status: %s\n", task.Status)
			if task.Project != nil {
				fmt.Printf("Project: %s (%s)\n", task.Project.Name, task.ProjectID.String()[:8])
			}
		},
	}

	return cmd
}

// task delete
func newTaskDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [task-id]",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			// Confirmation prompt
			confirm := false
			prompt := &survey.Confirm{
				Message: "Are you sure you want to delete this task?",
			}
			survey.AskOne(prompt, &confirm)

			if !confirm {
				fmt.Println("Task deletion cancelled.")
				return
			}

			client := api.NewClient()
			err := client.DeleteTask(taskID)
			if err != nil {
				fmt.Printf("Error deleting task: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🗑️  Task deleted successfully!\n")
		},
	}

	return cmd
}

// Helper function to parse partial UUID
func parsePartialUUID(partial string) string {
	// If it looks like a full UUID, return as is
	if len(partial) == 36 {
		return partial
	}
	// For now, return as is - the API should handle partial matches
	return partial
}