package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/urfave/cli/v2"
)

// NewTaskCommand creates all subcommands for the 'task' command group.
func NewTaskCommand() *cli.Command {
	return &cli.Command{
		Name:    "task",
		Aliases: []string{"t"},
		Usage:   "Manage tasks",
		Subcommands: []*cli.Command{
			taskListCmd(),
			taskCreateCmd(),
			taskShowCmd(),
			taskUpdateCmd(),
			taskStartCmd(),
			taskCompleteCmd(),
			taskDeleteCmd(),
		},
	}
}

// taskListCmd lists tasks.
func taskListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List tasks",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Filter by project ID. If not provided, active project is used."},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status (TODO, IN_PROGRESS, COMPLETED)"},
		},
		Action: func(c *cli.Context) error {
			projectID := c.String("project")
			status := c.String("status")

			client := api.NewClient()

			if projectID == "" {
				// If no project is specified, find the active one from the server
				projects, err := client.ListProjects()
				if err != nil {
					return fmt.Errorf("could not fetch projects to find active one: %w", err)
				}
				for _, p := range projects {
					if p.IsActive {
						projectID = p.ID.String()
						break
					}
				}
			}

			tasks, err := client.ListTasks(projectID, status)
			if err != nil {
				fmt.Printf("Error listing tasks: %v\n", err)
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found for the given criteria.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPRIORITY")
			fmt.Fprintln(w, "--\t-----\t------\t--------")

			for _, t := range tasks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					t.ID.String()[:8],
					truncateString(t.Title, 40),
					t.Status,
					t.Priority)
			}
			w.Flush()
			return nil
		},
	}
}

// taskCreateCmd creates a new task.
func taskCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new task",
		ArgsUsage: "[title]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID. Defaults to active project."},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Task description"},
			&cli.StringFlag{Name: "priority", Usage: "Priority (H, M, L)", Value: "M"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task title is required")
			}
			title := c.Args().First()
			projectID := c.String("project")
			description := c.String("description")
			priority := c.String("priority")

			client := api.NewClient()

			if projectID == "" {
				// If no project is specified, find the active one from the server
				projects, err := client.ListProjects()
				if err != nil {
					return fmt.Errorf("could not fetch projects to find active one: %w", err)
				}
				for _, p := range projects {
					if p.IsActive {
						projectID = p.ID.String()
						break
					}
				}
			}

			if projectID == "" {
				return fmt.Errorf("no active project set. Use 'jbraincli project use <id>' or specify --project")
			}

			task, err := client.CreateTask(projectID, title, description, priority)
			if err != nil {
				fmt.Printf("Error creating task: %v\n", err)
				return err
			}

			fmt.Printf("✅ Task '%s' created successfully!\n", task.Title)
			fmt.Printf("ID: %s\n", task.ID.String()[:8])
			return nil
		},
	}
}

// taskShowCmd shows details for a specific task.
func taskShowCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Aliases:   []string{"info"},
		Usage:     "Show details for a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Printf("Error getting task: %v\n", err)
				return err
			}

			fmt.Printf("Task Details: %s\n", task.Title)
			fmt.Println(strings.Repeat("-", 40))
			fmt.Printf("ID:          %s\n", task.ID.String())
			fmt.Printf("Title:       %s\n", task.Title)
			fmt.Printf("Description: %s\n", task.Description)
			fmt.Printf("Status:      %s\n", task.Status)
			fmt.Printf("Priority:    %s\n", task.Priority)
			fmt.Printf("Project ID:  %s\n", task.ProjectID.String())
			fmt.Printf("Created At:  %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated At:  %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

// taskUpdateCmd updates a task.
func taskUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a task's properties",
		ArgsUsage: "[task-id]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "New title"},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "New description"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "New status (TODO, IN_PROGRESS, COMPLETED)"},
			&cli.StringFlag{Name: "priority", Aliases: []string{"p"}, Usage: "New priority (H, M, L)"},
			&cli.IntFlag{Name: "progress", Usage: "New progress percentage (0-100)", Value: -1},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			updateData := map[string]interface{}{}
			if c.IsSet("title") {
				updateData["title"] = c.String("title")
			}
			if c.IsSet("description") {
				updateData["description"] = c.String("description")
			}
			if c.IsSet("status") {
				updateData["status"] = c.String("status")
			}
			if c.IsSet("priority") {
				updateData["priority"] = c.String("priority")
			}
			if c.Int("progress") >= 0 {
				updateData["progress"] = c.Int("progress")
			}

			if len(updateData) == 0 {
				return fmt.Errorf("at least one flag is required to update")
			}

			client := api.NewClient()
			task, err := client.UpdateTask(taskID, updateData)
			if err != nil {
				fmt.Printf("Error updating task: %v\n", err)
				return err
			}

			fmt.Printf("✅ Task '%s' updated successfully.\n", task.Title)
			return nil
		},
	}
}

// taskStartCmd starts a task.
func taskStartCmd() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Start a task (set status to IN_PROGRESS)",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()
			updateData := map[string]interface{}{"status": "IN_PROGRESS"}

			client := api.NewClient()
			_, err := client.UpdateTask(taskID, updateData)
			if err != nil {
				fmt.Printf("Error starting task: %v\n", err)
				return err
			}
			fmt.Printf("🚀 Task %s marked as IN_PROGRESS.\n", taskID[:8])
			return nil
		},
	}
}

// taskCompleteCmd completes a task.
func taskCompleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "complete",
		Aliases:   []string{"done"},
		Usage:     "Complete a task (set status to COMPLETED)",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()
			updateData := map[string]interface{}{"status": "COMPLETED"}

			client := api.NewClient()
			_, err := client.UpdateTask(taskID, updateData)
			if err != nil {
				fmt.Printf("Error completing task: %v\n", err)
				return err
			}
			fmt.Printf("✅ Task %s marked as COMPLETED.\n", taskID[:8])
			return nil
		},
	}
}

// taskDeleteCmd deletes a task.
func taskDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			err := client.DeleteTask(taskID)
			if err != nil {
				fmt.Printf("Error deleting task: %v\n", err)
				return err
			}

			fmt.Printf("🗑️ Task %s deleted successfully.\n", taskID[:8])
			return nil
		},
	}
}