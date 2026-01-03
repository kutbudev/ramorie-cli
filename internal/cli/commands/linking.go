package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

func NewLinkCommand() *cli.Command {
	return &cli.Command{
		Name:  "link",
		Usage: "Link memories and tasks",
		Subcommands: []*cli.Command{
			linkCreateCmd(),
		},
	}
}

func NewTaskMemoriesCommand() *cli.Command {
	return &cli.Command{
		Name:      "task-memories",
		Usage:     "List memories linked to a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()
			client := api.NewClient()
			memories, err := client.ListTaskMemories(taskID)
			if err != nil {
				return err
			}

			if len(memories) == 0 {
				fmt.Println("No linked memories found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tCONTENT")
			fmt.Fprintln(w, "--\t-------")
			for _, m := range memories {
				fmt.Fprintf(w, "%s\t%s\n", m.ID.String()[:8], truncateString(m.Content, 70))
			}
			w.Flush()
			return nil
		},
	}
}

func NewMemoryTasksCommand() *cli.Command {
	return &cli.Command{
		Name:      "memory-tasks",
		Usage:     "List tasks linked to a memory",
		ArgsUsage: "[memory-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory ID is required")
			}
			memoryID := c.Args().First()
			client := api.NewClient()
			tasks, err := client.ListMemoryTasks(memoryID)
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No linked tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPRIORITY")
			fmt.Fprintln(w, "--\t-----\t------\t--------")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID.String()[:8], truncateString(t.Title, 40), t.Status, t.Priority)
			}
			w.Flush()
			return nil
		},
	}
}

func linkCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a manual memory-task link",
		ArgsUsage: "[task-id] [memory-id]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "relation-type", Aliases: []string{"t"}, Usage: "Relation type"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("task ID and memory ID are required")
			}
			taskID := c.Args().Get(0)
			memoryID := c.Args().Get(1)
			relationType := c.String("relation-type")

			client := api.NewClient()
			_, err := client.CreateMemoryTaskLink(taskID, memoryID, relationType)
			if err != nil {
				return err
			}

			fmt.Println("✅ Link created successfully.")
			return nil
		},
	}
}
