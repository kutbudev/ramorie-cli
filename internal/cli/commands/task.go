package commands

import (
	"fmt"
	"os"
	"strconv"
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
			taskElaborateCmd(),
			taskDuplicateCmd(),
			taskMoveCmd(),
			taskNextCmd(),
			taskProgressCmd(),
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
			&cli.StringFlag{Name: "priority", Aliases: []string{"P"}, Usage: "Priority (H, M, L)", Value: "M"},
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
				return fmt.Errorf("no active project set. Use 'ramorie project use <id>' or specify --project")
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

			if len(task.Annotations) > 0 {
				fmt.Println(strings.Repeat("-", 40))
				fmt.Println("Annotations:")
				for _, an := range task.Annotations {
					fmt.Printf("  - [%s] %s\n", an.CreatedAt.Format("2006-01-02 15:04"), an.Content)
				}
			}
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
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Usage:   "New title",
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "New description",
			},
			&cli.StringFlag{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "New status (TODO, IN_PROGRESS, COMPLETED)",
			},
			&cli.StringFlag{
				Name:    "priority",
				Aliases: []string{"P"},
				Usage:   "New priority (H, M, L)",
			},
			&cli.IntFlag{
				Name:  "progress",
				Usage: "New progress percentage (0-100)",
				Value: -1,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}

			args := c.Args().Slice()
			taskID := args[0]

			updateData := map[string]interface{}{}

			// Manual flag parsing since urfave/cli seems to have issues
			for i := 1; i < len(args); i++ {
				if args[i] == "--title" || args[i] == "-t" {
					if i+1 < len(args) {
						updateData["title"] = args[i+1]
						i++ // Skip next argument as it's the value
					}
				} else if args[i] == "--description" || args[i] == "-d" {
					if i+1 < len(args) {
						updateData["description"] = args[i+1]
						i++
					}
				} else if args[i] == "--status" || args[i] == "-s" {
					if i+1 < len(args) {
						updateData["status"] = args[i+1]
						i++
					}
				} else if args[i] == "--priority" || args[i] == "-P" {
					if i+1 < len(args) {
						updateData["priority"] = args[i+1]
						i++
					}
				} else if args[i] == "--progress" {
					if i+1 < len(args) {
						if progress, err := strconv.Atoi(args[i+1]); err == nil && progress >= 0 && progress <= 100 {
							updateData["progress"] = progress
						}
						i++
					}
				}
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

			fmt.Printf("✅ Task %s deleted successfully.\n", taskID[:8])
			return nil
		},
	}
}

// taskElaborateCmd uses AI to elaborate on a task's description and saves it as an annotation.
func taskElaborateCmd() *cli.Command {
	return &cli.Command{
		Name:      "elaborate",
		Aliases:   []string{"elab"},
		Usage:     "Use AI to elaborate on a task and save as a note",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			_, err := client.ElaborateTask(taskID)
			if err != nil {
				// The error from the API client is already quite descriptive
				fmt.Printf("Error elaborating task: %v\n", err)
				return err
			}

			fmt.Printf("✅ Successfully elaborated on task %s and saved it as a new note.\n", taskID)
			fmt.Printf("Use 'ramorie task show %s' to see the results.\n", taskID)
			return nil
		},
	}
}

// taskDuplicateCmd duplicates a task with its tags and notes.
func taskDuplicateCmd() *cli.Command {
	return &cli.Command{
		Name:      "duplicate",
		Aliases:   []string{"dup", "copy"},
		Usage:     "Duplicate a task (copies tags and notes, resets status to TODO)",
		ArgsUsage: "[task-id]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Usage:   "New title for the duplicated task (optional)",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()
			newTitle := c.String("title")

			client := api.NewClient()

			// Get original task
			original, err := client.GetTask(taskID)
			if err != nil {
				fmt.Printf("Error getting task: %v\n", err)
				return err
			}

			// Create new task with same properties
			title := original.Title
			if newTitle != "" {
				title = newTitle
			} else {
				title = title + " (copy)"
			}

			newTask, err := client.CreateTask(
				original.ProjectID.String(),
				title,
				original.Description,
				original.Priority,
			)
			if err != nil {
				fmt.Printf("Error creating duplicate task: %v\n", err)
				return err
			}

			// Copy annotations
			for _, ann := range original.Annotations {
				_, _ = client.CreateAnnotation(newTask.ID.String(), ann.Content)
			}

			fmt.Printf("✅ Task duplicated successfully!\n")
			fmt.Printf("Original: %s - %s\n", original.ID.String()[:8], original.Title)
			fmt.Printf("New:      %s - %s\n", newTask.ID.String()[:8], newTask.Title)
			return nil
		},
	}
}

// taskMoveCmd moves tasks to another project.
func taskMoveCmd() *cli.Command {
	return &cli.Command{
		Name:      "move",
		Usage:     "Move task(s) to another project",
		ArgsUsage: "[task-ids...]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "Target project ID or name",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("at least one task ID is required")
			}

			targetProject := c.String("project")
			if targetProject == "" {
				return fmt.Errorf("target project is required (--project)")
			}

			taskIDs := c.Args().Slice()
			client := api.NewClient()

			// Resolve project name to ID if needed
			projects, err := client.ListProjects()
			if err != nil {
				return fmt.Errorf("could not fetch projects: %w", err)
			}

			var projectID string
			for _, p := range projects {
				if p.ID.String() == targetProject || strings.HasPrefix(p.ID.String(), targetProject) || strings.EqualFold(p.Name, targetProject) {
					projectID = p.ID.String()
					break
				}
			}

			if projectID == "" {
				return fmt.Errorf("project '%s' not found", targetProject)
			}

			// Move each task
			movedCount := 0
			for _, taskID := range taskIDs {
				updateData := map[string]interface{}{"project_id": projectID}
				_, err := client.UpdateTask(taskID, updateData)
				if err != nil {
					fmt.Printf("⚠️  Failed to move task %s: %v\n", taskID[:8], err)
					continue
				}
				movedCount++
			}

			fmt.Printf("✅ Moved %d/%d task(s) to project.\n", movedCount, len(taskIDs))
			return nil
		},
	}
}

// taskNextCmd shows next tasks by priority.
func taskNextCmd() *cli.Command {
	return &cli.Command{
		Name:  "next",
		Usage: "Show next tasks by priority (optimized for agents)",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "count",
				Aliases: []string{"n"},
				Usage:   "Number of tasks to show",
				Value:   5,
			},
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID",
			},
		},
		Action: func(c *cli.Context) error {
			count := c.Int("count")
			projectID := c.String("project")

			client := api.NewClient()

			if projectID == "" {
				// Find active project
				projects, err := client.ListProjects()
				if err != nil {
					return fmt.Errorf("could not fetch projects: %w", err)
				}
				for _, p := range projects {
					if p.IsActive {
						projectID = p.ID.String()
						break
					}
				}
			}

			// Get all tasks
			tasks, err := client.ListTasks(projectID, "")
			if err != nil {
				return fmt.Errorf("could not fetch tasks: %w", err)
			}

			// Filter pending tasks and calculate priority score
			type scoredTask struct {
				idx   int
				score int
			}
			var scored []scoredTask
			priorityMap := map[string]int{"H": 3, "M": 2, "L": 1}

			for i, t := range tasks {
				if t.Status == "TODO" || t.Status == "IN_PROGRESS" {
					score := priorityMap[t.Priority]
					if score == 0 {
						score = 2 // Default to Medium
					}
					// IN_PROGRESS tasks get a boost
					if t.Status == "IN_PROGRESS" {
						score += 10
					}
					scored = append(scored, scoredTask{idx: i, score: score})
				}
			}

			// Sort by score (descending)
			for i := 0; i < len(scored)-1; i++ {
				for j := i + 1; j < len(scored); j++ {
					if scored[j].score > scored[i].score {
						scored[i], scored[j] = scored[j], scored[i]
					}
				}
			}

			// Limit results
			if len(scored) > count {
				scored = scored[:count]
			}

			if len(scored) == 0 {
				fmt.Println(" No pending tasks! You're all caught up.")
				return nil
			}

			fmt.Printf(" Next %d task(s):\n", len(scored))
			fmt.Println(strings.Repeat("-", 60))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "#	ID	PRIORITY	STATUS	TITLE")

			for i, s := range scored {
				t := tasks[s.idx]
				fmt.Fprintf(w, "%d	%s	%s	%s	%s\n",
					i+1,
					t.ID.String()[:8],
					t.Priority,
					t.Status,
					truncateString(t.Title, 35))
			}
			w.Flush()
			return nil
		},
	}
}

// taskProgressCmd updates task progress.
func taskProgressCmd() *cli.Command {
	return &cli.Command{
		Name:      "progress",
		Usage:     "Update task progress (0-100)",
		ArgsUsage: "[task-id] [progress]",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: ramorie task progress <task-id> <progress>")
			}

			taskID := c.Args().Get(0)
			progressStr := c.Args().Get(1)

			progress, err := strconv.Atoi(progressStr)
			if err != nil || progress < 0 || progress > 100 {
				return fmt.Errorf("progress must be a number between 0 and 100")
			}

			client := api.NewClient()

			// First get the task to resolve short ID to full UUID
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Printf("Error finding task: %v\n", err)
				return err
			}

			updateData := map[string]interface{}{"progress": progress}
			task, err = client.UpdateTask(task.ID.String(), updateData)
			if err != nil {
				fmt.Printf("Error updating progress: %v\n", err)
				return err
			}

			// Visual progress bar
			filled := progress / 5
			empty := 20 - filled
			bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

			fmt.Printf("📊 Task '%s' progress updated\n", truncateString(task.Title, 30))
			fmt.Printf("   [%s] %d%%\n", bar, progress)
			return nil
		},
	}
}
