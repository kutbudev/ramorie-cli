package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/urfave/cli/v2"
)

// NewSubtaskCommand creates the subtask command group.
func NewSubtaskCommand() *cli.Command {
	return &cli.Command{
		Name:    "subtask",
		Aliases: []string{"sub"},
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
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			subtasks, err := client.ListSubtasks(taskID)
			if err != nil {
				fmt.Printf("Error listing subtasks: %v\n", err)
				return err
			}

			if len(subtasks) == 0 {
				fmt.Println("No subtasks found for this task.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tDESCRIPTION\tSTATUS")
			fmt.Fprintln(w, "--\t-----------\t------")

			for _, s := range subtasks {
				status := "⬜ Pending"
				if s.Completed == 1 {
					status = "✅ Done"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					s.ID.String()[:8],
					truncateString(s.Description, 40),
					status)
			}
			w.Flush()
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
