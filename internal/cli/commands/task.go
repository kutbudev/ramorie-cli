package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/models"
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
			taskStopCmd(),
			taskCompleteCmd(),
			taskActiveCmd(),
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
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Filter by project ID or name"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status (TODO, IN_PROGRESS, COMPLETED)"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "Limit number of results", Value: 0},
		},
		Action: func(c *cli.Context) error {
			projectArg := c.String("project")
			status := c.String("status")
			limit := c.Int("limit")

			client := api.NewClient()

			// Resolve project name/short-id to full UUID
			var projectID string
			if projectArg != "" {
				projects, err := client.ListProjects()
				if err != nil {
					return fmt.Errorf("could not fetch projects: %w", err)
				}
				for _, p := range projects {
					if p.ID.String() == projectArg ||
						strings.HasPrefix(p.ID.String(), projectArg) ||
						strings.EqualFold(p.Name, projectArg) {
						projectID = p.ID.String()
						break
					}
				}
				if projectID == "" {
					return fmt.Errorf("project '%s' not found", projectArg)
				}
			}

			tasks, err := client.ListTasks(projectID, status)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found for the given criteria.")
				return nil
			}

			// Apply limit if specified
			if limit > 0 && len(tasks) > limit {
				tasks = tasks[:limit]
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPRIORITY")
			fmt.Fprintln(w, "--\t-----\t------\t--------")

			for _, t := range tasks {
				// CRITICAL: Decrypt task fields before displaying
				decryptedTitle, _ := decryptTaskForCLI(&t)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					t.ID.String()[:8],
					truncateString(decryptedTitle, 40),
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
		Name:                   "create",
		Usage:                  "Create a new task",
		ArgsUsage:              "[title]",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID or name (required)", Required: true},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Task description"},
			&cli.StringFlag{Name: "priority", Aliases: []string{"P"}, Usage: "Priority (H, M, L)", Value: "M"},
			&cli.StringSliceFlag{Name: "tags", Aliases: []string{"t"}, Usage: "Tags (comma-separated or multiple -t flags)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task title is required. Usage: ramorie task create [--priority H] [--tags tag1,tag2] \"Task title\"")
			}
			title := c.Args().First()
			projectArg := c.String("project")
			description := c.String("description")
			priority := c.String("priority")
			tags := c.StringSlice("tags")

			client := api.NewClient()

			// Resolve project name/short-id to full UUID
			projects, err := client.ListProjects()
			if err != nil {
				return fmt.Errorf("could not fetch projects: %w", err)
			}

			var projectID string
			for _, p := range projects {
				if p.ID.String() == projectArg ||
					strings.HasPrefix(p.ID.String(), projectArg) ||
					strings.EqualFold(p.Name, projectArg) {
					projectID = p.ID.String()
					break
				}
			}
			if projectID == "" {
				return fmt.Errorf("project '%s' not found", projectArg)
			}

			var task *models.Task

			if crypto.IsVaultUnlocked() {
				// Encrypt title and description before sending (zero-knowledge encryption)
				encTitle, titleNonce, titleEncrypted, encErr := crypto.EncryptContent(title)
				if encErr != nil {
					return fmt.Errorf("encryption failed: %w", encErr)
				}

				encDesc := ""
				descNonce := ""
				if description != "" {
					encDesc, descNonce, _, encErr = crypto.EncryptContent(description)
					if encErr != nil {
						return fmt.Errorf("encryption failed: %w", encErr)
					}
				}

				if titleEncrypted {
					task, err = client.CreateEncryptedTask(projectID, encTitle, titleNonce, encDesc, descNonce, priority, tags...)
					if err == nil {
						fmt.Printf("🔐 Task encrypted and created successfully!\n")
					}
				} else {
					task, err = client.CreateTask(projectID, title, description, priority, tags...)
					if err == nil {
						fmt.Printf("✅ Task '%s' created successfully!\n", task.Title)
					}
				}
			} else {
				// Vault is locked - send plaintext
				task, err = client.CreateTask(projectID, title, description, priority, tags...)
				if err == nil {
					fmt.Printf("✅ Task '%s' created successfully!\n", task.Title)
				}
			}

			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("ID: %s\n", task.ID.String()[:8])
			if len(tags) > 0 {
				fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
			}
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
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// CRITICAL: Decrypt task fields before displaying
			decryptedTitle, decryptedDesc := decryptTaskForCLI(task)

			fmt.Printf("Task Details: %s\n", decryptedTitle)
			fmt.Println(strings.Repeat("-", 40))
			fmt.Printf("ID:          %s\n", task.ID.String())
			fmt.Printf("Title:       %s\n", decryptedTitle)
			fmt.Printf("Description: %s\n", decryptedDesc)
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
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Task '%s' updated successfully.\n", task.Title)
			return nil
		},
	}
}

// taskStartCmd starts a task and sets it as the active task for memory linking.
func taskStartCmd() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Start a task (set as active + IN_PROGRESS, memories will auto-link)",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			err := client.StartTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			shortID := taskID
			if len(taskID) > 8 {
				shortID = taskID[:8]
			}
			fmt.Printf("🚀 Task %s is now ACTIVE and IN_PROGRESS.\n", shortID)
			fmt.Println("💡 New memories will automatically link to this task.")
			return nil
		},
	}
}

// taskCompleteCmd completes a task and clears active status.
func taskCompleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "complete",
		Aliases:   []string{"done"},
		Usage:     "Complete a task (COMPLETED + clears active status)",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			err := client.CompleteTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			shortID := taskID
			if len(taskID) > 8 {
				shortID = taskID[:8]
			}
			fmt.Printf("✅ Task %s marked as COMPLETED.\n", shortID)
			return nil
		},
	}
}

// taskStopCmd pauses work on a task (clears active status but keeps IN_PROGRESS).
func taskStopCmd() *cli.Command {
	return &cli.Command{
		Name:      "stop",
		Aliases:   []string{"pause"},
		Usage:     "Stop working on a task (clears active, keeps IN_PROGRESS)",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			err := client.StopTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			shortID := taskID
			if len(taskID) > 8 {
				shortID = taskID[:8]
			}
			fmt.Printf("⏸️  Task %s paused. No longer the active task.\n", shortID)
			fmt.Println("💡 New memories will NOT auto-link until you start a task again.")
			return nil
		},
	}
}

// taskActiveCmd shows the currently active task.
func taskActiveCmd() *cli.Command {
	return &cli.Command{
		Name:  "active",
		Usage: "Show the currently active task (for memory auto-linking)",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			task, err := client.GetActiveTask()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if task == nil {
				fmt.Println("📭 No active task set.")
				fmt.Println("💡 Use 'ramorie task start <task-id>' to set one.")
				return nil
			}

			// CRITICAL: Decrypt task fields before displaying
			decryptedTitle, _ := decryptTaskForCLI(task)

			fmt.Println("🎯 Active Task:")
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("ID:       %s\n", task.ID.String()[:8])
			fmt.Printf("Title:    %s\n", decryptedTitle)
			fmt.Printf("Status:   %s\n", task.Status)
			fmt.Printf("Priority: %s\n", task.Priority)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println("💡 New memories will automatically link to this task.")
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
				fmt.Println(apierrors.ParseAPIError(err))
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
				fmt.Println(apierrors.ParseAPIError(err))
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
				fmt.Println(apierrors.ParseAPIError(err))
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
				fmt.Println(apierrors.ParseAPIError(err))
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
			projectArg := c.String("project")

			client := api.NewClient()

			// Resolve project name/short-id to full UUID
			var projectID string
			if projectArg != "" {
				projects, err := client.ListProjects()
				if err != nil {
					return fmt.Errorf("could not fetch projects: %w", err)
				}
				for _, p := range projects {
					if p.ID.String() == projectArg ||
						strings.HasPrefix(p.ID.String(), projectArg) ||
						strings.EqualFold(p.Name, projectArg) {
						projectID = p.ID.String()
						break
					}
				}
				if projectID == "" {
					return fmt.Errorf("project '%s' not found", projectArg)
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
				// CRITICAL: Decrypt task fields before displaying
				decryptedTitle, _ := decryptTaskForCLI(&t)
				fmt.Fprintf(w, "%d	%s	%s	%s	%s\n",
					i+1,
					t.ID.String()[:8],
					t.Priority,
					t.Status,
					truncateString(decryptedTitle, 35))
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
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			updateData := map[string]interface{}{"progress": progress}
			task, err = client.UpdateTask(task.ID.String(), updateData)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Visual progress bar
			filled := progress / 5
			empty := 20 - filled
			bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

			// CRITICAL: Decrypt task fields before displaying
			decryptedTitle, _ := decryptTaskForCLI(task)
			fmt.Printf("📊 Task '%s' progress updated\n", truncateString(decryptedTitle, 30))
			fmt.Printf("   [%s] %d%%\n", bar, progress)
			return nil
		},
	}
}

// decryptTaskForCLI decrypts task title and description if encrypted and vault is unlocked.
// Returns decrypted title and description.
func decryptTaskForCLI(t *models.Task) (title, description string) {
	if !t.IsEncrypted {
		return t.Title, t.Description
	}

	// Check if vault is unlocked
	if !crypto.IsVaultUnlocked() {
		title = "[Vault Locked - run 'ramorie setup unlock']"
		description = "[Vault Locked - run 'ramorie setup unlock']"
		// If we have non-placeholder titles, use them
		if t.Title != "" && t.Title != "[Encrypted]" {
			title = t.Title
		}
		if t.Description != "" && t.Description != "[Encrypted]" {
			description = t.Description
		}
		return title, description
	}

	// Decrypt title
	if t.EncryptedTitle != "" {
		decrypted, err := crypto.DecryptContent(t.EncryptedTitle, t.TitleNonce, true)
		if err != nil {
			title = "[Decryption Failed]"
		} else {
			title = decrypted
		}
	} else {
		title = t.Title
	}

	// Decrypt description
	if t.EncryptedDescription != "" {
		decrypted, err := crypto.DecryptContent(t.EncryptedDescription, t.DescriptionNonce, true)
		if err != nil {
			description = "[Decryption Failed]"
		} else {
			description = decrypted
		}
	} else {
		description = t.Description
	}

	return title, description
}
