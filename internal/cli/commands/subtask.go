package commands

import (
	"fmt"
	"slices"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/urfave/cli/v2"
)

// NewSubtaskCommand creates the subtask command group.
func NewSubtaskCommand() *cli.Command {
	return &cli.Command{
		Name:    "subtask",
		Aliases: []string{"sub", "subtasks"},
		Usage:   "Manage subtasks",
		Subcommands: []*cli.Command{
			subtaskListCmd(),
			subtaskAddCmd(),
			subtaskCompleteCmd(),
			subtaskDeleteCmd(),
		},
	}
}

// subtaskListCmd lists subtasks for a task.
func subtaskListCmd() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Aliases:   []string{"ls"},
		Usage:     "List subtasks for a task",
		ArgsUsage: "[task-id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "newest-first", Usage: "Show newest item at the top (default: oldest at top)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()
			newestFirst := c.Bool("newest-first")

			client := api.NewClient()
			subtasks, err := client.ListSubtasks(taskID)
			if err != nil {
				fmt.Printf("Error listing subtasks: %v\n", err)
				return err
			}

			if len(subtasks) == 0 {
				fmt.Println(display.Dim.Render("  no subtasks"))
				return nil
			}

			// Default: chronological asc (oldest top, newest bottom).
			if !newestFirst {
				slices.Reverse(subtasks)
			}

			cols := []display.Column{
				{Title: "DONE", Min: 4, Weight: 0},
				{Title: "ID", Min: 8, Weight: 0},
				{Title: "DESCRIPTION", Min: 24, Weight: 4},
			}
			rows := make([][]string, 0, len(subtasks))
			for _, s := range subtasks {
				done := display.Dim.Render("○")
				if s.Completed == 1 {
					done = display.Good.Render("✓")
				}
				rows = append(rows, []string{
					done,
					display.Dim.Render(s.ID.String()[:8]),
					display.SingleLine(s.Description),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
			return nil
		},
	}
}

// subtaskAddCmd adds a subtask to a task.
func subtaskAddCmd() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a subtask to a task",
		ArgsUsage: "[task-id] [description]",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: ramorie subtask add <task-id> <description>")
			}

			taskID := c.Args().Get(0)
			description := c.Args().Get(1)

			// If description has multiple words, join them
			if c.NArg() > 2 {
				args := c.Args().Slice()
				description = ""
				for i := 1; i < len(args); i++ {
					if i > 1 {
						description += " "
					}
					description += args[i]
				}
			}

			client := api.NewClient()
			subtask, err := client.CreateSubtask(taskID, description)
			if err != nil {
				fmt.Printf("Error creating subtask: %v\n", err)
				return err
			}

			fmt.Printf("✅ Subtask added: %s\n", subtask.Description)
			fmt.Printf("   ID: %s\n", subtask.ID.String()[:8])
			return nil
		},
	}
}

// subtaskCompleteCmd marks a subtask as completed.
func subtaskCompleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "complete",
		Aliases:   []string{"done"},
		Usage:     "Mark a subtask as completed",
		ArgsUsage: "[task-id] [subtask-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: ramorie subtask complete <task-id> <subtask-id>")
			}

			taskID := c.Args().Get(0)
			subtaskID := c.Args().Get(1)

			client := api.NewClient()

			// Update subtask to completed
			updateData := map[string]interface{}{"completed": 1}
			_, err := client.Request("PUT", fmt.Sprintf("/tasks/%s/subtasks/%s", taskID, subtaskID), updateData)
			if err != nil {
				fmt.Printf("Error completing subtask: %v\n", err)
				return err
			}

			fmt.Printf("✅ Subtask %s marked as completed.\n", subtaskID[:8])
			return nil
		},
	}
}

// subtaskDeleteCmd deletes a subtask.
func subtaskDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm"},
		Usage:     "Delete a subtask",
		ArgsUsage: "[task-id] [subtask-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: ramorie subtask delete <task-id> <subtask-id>")
			}

			taskID := c.Args().Get(0)
			subtaskID := c.Args().Get(1)

			client := api.NewClient()

			_, err := client.Request("DELETE", fmt.Sprintf("/tasks/%s/subtasks/%s", taskID, subtaskID), nil)
			if err != nil {
				fmt.Printf("Error deleting subtask: %v\n", err)
				return err
			}

			fmt.Printf("✅ Subtask %s deleted.\n", subtaskID[:8])
			return nil
		},
	}
}
