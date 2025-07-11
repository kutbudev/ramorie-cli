package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
	"github.com/urfave/cli/v2"
)

// NewMemoryCommand creates all subcommands for the 'memory' command group.
func NewMemoryCommand() *cli.Command {
	return &cli.Command{
		Name:    "memory",
		Aliases: []string{"m"},
		Usage:   "Manage memories (knowledge base)",
		Subcommands: []*cli.Command{
			rememberCmd(),
			memoriesCmd(),
			getCmd(),
			recallCmd(),
			forgetCmd(),
		},
	}
}

// NewRememberCommand creates a standalone remember command
func NewRememberCommand() *cli.Command {
	return rememberCmd()
}

// rememberCmd creates a new memory item.
func rememberCmd() *cli.Command {
	return &cli.Command{
		Name:      "remember",
		Usage:     "Create a new memory",
		ArgsUsage: "[content]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID. Defaults to the active project.",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory content is required")
			}
			content := c.Args().First()
			projectID := c.String("project")

			if projectID == "" {
				cfg, err := config.LoadConfig()
				if err != nil || cfg.ActiveProjectID == "" {
					return fmt.Errorf("no active project set. Use 'jbraincli project use <id>' or specify --project")
				}
				projectID = cfg.ActiveProjectID
			}

			client := api.NewClient()
			memory, err := client.CreateMemory(projectID, content)
			if err != nil {
				fmt.Printf("Error creating memory: %v\n", err)
				return err
			}
			fmt.Printf("🧠 Memory stored successfully! (ID: %s)\n", memory.ID.String()[:8])
			return nil
		},
	}
}

// memoriesCmd lists all memory items.
func memoriesCmd() *cli.Command {
	return &cli.Command{
		Name:  "memories",
		Usage: "List all memories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID. If not provided, lists for the active project.",
			},
		},
		Action: func(c *cli.Context) error {
			projectID := c.String("project")
			if projectID == "" {
				cfg, err := config.LoadConfig()
				if err == nil && cfg.ActiveProjectID != "" {
					projectID = cfg.ActiveProjectID
				}
			}

			client := api.NewClient()
			memories, err := client.ListMemories(projectID, "") // No search query
			if err != nil {
				fmt.Printf("Error listing memories: %v\n", err)
				return err
			}

			if len(memories) == 0 {
				fmt.Println("No memories found.")
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

// recallCmd searches memory items.
func recallCmd() *cli.Command {
	return &cli.Command{
		Name:      "recall",
		Usage:     "Search within your memories",
		ArgsUsage: "[search-query]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("a search query is required")
			}
			query := c.Args().First()

			client := api.NewClient()
			memories, err := client.ListMemories("", query) // Search across all projects
			if err != nil {
				fmt.Printf("Error recalling memories: %v\n", err)
				return err
			}

			if len(memories) == 0 {
				fmt.Printf("No memories found matching '%s'.\n", query)
				return nil
			}

			fmt.Printf("Found %d memories matching your query:\n", len(memories))
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

// getCmd retrieves a memory item by ID.
func getCmd() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Retrieve a memory by ID",
		ArgsUsage: "[memory-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory ID is required")
			}
			memoryID := c.Args().First()

			client := api.NewClient()
			memory, err := client.GetMemory(memoryID)
			if err != nil {
				fmt.Printf("Error getting memory: %v\n", err)
				return err
			}

			fmt.Printf("Memory %s:\n%s\n", memory.ID.String()[:8], memory.Content)
			return nil
		},
	}
}

// forgetCmd deletes a memory item.
func forgetCmd() *cli.Command {
	return &cli.Command{
		Name:      "forget",
		Usage:     "Delete a memory",
		ArgsUsage: "[memory-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory ID is required")
			}
			memoryID := c.Args().First()

			client := api.NewClient()
			err := client.DeleteMemory(memoryID)
			if err != nil {
				fmt.Printf("Error forgetting memory: %v\n", err)
				return err
			}

			fmt.Printf("🗑️ Memory %s forgotten successfully.\n", memoryID[:8])
			return nil
		},
	}
}
