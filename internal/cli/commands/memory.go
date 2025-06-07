package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/config"
	"github.com/terzigolu/josepshbrain-go/internal/api"
)

// NewMemoryCmd creates the memory command with subcommands, fully API-driven.
func NewMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Memory management commands",
		Long:  "Store and recall knowledge with remember and recall operations.",
	}

	cmd.AddCommand(newRememberCmd())
	cmd.AddCommand(newRecallCmd())

	return cmd
}

// remember command (alias for memory create)
func newRememberCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "remember [content]",
		Short: "Store a memory/knowledge item",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			content := args[0]

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
			memory, err := client.CreateMemory(cfg.ActiveProjectID, content, tags)
			if err != nil {
				fmt.Printf("Error creating memory: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🧠 Memory stored successfully!\n")
			fmt.Printf("ID: %s\n", memory.ID.String())
			fmt.Printf("Content: %s\n", truncateString(memory.Content, 100))
			if len(memory.Tags) > 0 {
				var tags []string
				for k := range memory.Tags {
					tags = append(tags, k)
				}
				fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
			}
			if memory.Project != nil {
				fmt.Printf("Project: %s (%s)\n", memory.Project.Name, memory.ProjectID.String()[:8])
			}
		},
	}

	cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Memory tags (comma-separated)")

	return cmd
}

// recall command (alias for memory search)
func newRecallCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "recall [search-term]",
		Short: "Search and recall memories",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var search string
			if len(args) > 0 {
				search = args[0]
			}

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
			memories, err := client.ListMemories(projectID, search)
			if err != nil {
				fmt.Printf("Error recalling memories: %v\n", err)
				os.Exit(1)
			}

			if len(memories) == 0 {
				if search != "" {
					fmt.Printf("No memories found for search term: %s\n", search)
				} else {
					fmt.Println("No memories found.")
				}
				return
			}

			fmt.Printf("🧠 Found %d memories:\n\n", len(memories))
			
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"ID", "Content", "Tags", "Created At"})

			for _, memory := range memories {
				var tags []string
				for k := range memory.Tags {
					tags = append(tags, k)
				}
				table.Append([]string{
					memory.ID.String()[:8],
					truncateText(memory.Content, 50),
					strings.Join(tags, ", "),
					memory.CreatedAt.Format("2006-01-02 15:04"),
				})
			}
			table.Render()
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Filter by project ID")

	return cmd
}

// Root level remember command (shortcut)
func NewRememberCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "remember [content]",
		Short: "Store a memory/knowledge item (shortcut for memory remember)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			content := args[0]

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
			memory, err := client.CreateMemory(cfg.ActiveProjectID, content, tags)
			if err != nil {
				fmt.Printf("Error creating memory: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🧠 Memory stored successfully!\n")
			fmt.Printf("ID: %s\n", memory.ID.String())
			fmt.Printf("Content: %s\n", truncateString(memory.Content, 100))
			if len(memory.Tags) > 0 {
				var tags []string
				for k := range memory.Tags {
					tags = append(tags, k)
				}
				fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
			}
		},
	}

	cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Memory tags (comma-separated)")

	return cmd
}

func truncateText(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}