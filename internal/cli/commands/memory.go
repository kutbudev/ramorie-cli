package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/constants"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/models"
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
		Name:                   "remember",
		Usage:                  "Create a new memory",
		ArgsUsage:              "[content]",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID. Defaults to the active project.",
			},
			&cli.StringSliceFlag{
				Name:    "tags",
				Aliases: []string{"t"},
				Usage:   "Tags for the memory (can be used multiple times or comma-separated)",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory content is required")
			}
			content := c.Args().First()
			projectID := c.String("project")
			tags := c.StringSlice("tags")

			// Check content length limit before sending
			if !constants.IsWithinMemoryLimit(content) {
				chars, tokens, usage := constants.GetContentStats(content)
				fmt.Printf("❌ Content exceeds maximum limit!\n")
				fmt.Printf("   Your content: %d chars (~%d tokens)\n", chars, tokens)
				fmt.Printf("   Maximum: %d chars (~%d tokens)\n", constants.MaxMemoryChars, constants.MaxMemoryChars/constants.CharsPerToken)
				fmt.Printf("   Usage: %.1f%%\n", usage)
				return fmt.Errorf("content too large")
			}

			// Show warning if approaching limit (80%+)
			chars, tokens, usage := constants.GetContentStats(content)
			if usage >= constants.WarningThresholdPercent {
				fmt.Printf("⚠️  Warning: Content is %.1f%% of maximum limit (%d chars)\n", usage, chars)
			}

			if projectID == "" {
				cfg, err := config.LoadConfig()
				if err != nil || cfg.ActiveProjectID == "" {
					return fmt.Errorf("no active project set. Use 'ramorie project use <id>' or specify --project")
				}
				projectID = cfg.ActiveProjectID
			}

			client := api.NewClient()
			memory, err := client.CreateMemory(projectID, content, tags...)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}
			fmt.Printf("🧠 Memory stored successfully! (ID: %s)\n", memory.ID.String()[:8])
			fmt.Printf("   Size: %d chars (~%d tokens)\n", chars, tokens)
			if len(tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(tags, ", "))
			}

			// Show if memory was auto-linked to active task
			if memory.LinkedTaskID != nil {
				fmt.Printf("🔗 Auto-linked to active task: %s\n", memory.LinkedTaskID.String()[:8])
			}
			return nil
		},
	}
}

// memoriesCmd lists all memory items.
func memoriesCmd() *cli.Command {
	return &cli.Command{
		Name:                   "memories",
		Usage:                  "List all memories",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID. If not provided, lists for the active project.",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "List memories from all projects (including organization projects)",
			},
			&cli.BoolFlag{
				Name:  "org-only",
				Usage: "Only show memories from organization projects",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Limit number of results",
				Value:   0,
			},
			&cli.StringFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Filter by tag",
			},
		},
		Action: func(c *cli.Context) error {
			projectID := c.String("project")
			showAll := c.Bool("all")
			orgOnly := c.Bool("org-only")
			limit := c.Int("limit")
			tagFilter := c.String("tag")

			if !showAll && projectID == "" {
				cfg, err := config.LoadConfig()
				if err == nil && cfg.ActiveProjectID != "" {
					projectID = cfg.ActiveProjectID
				}
			}

			// If --all flag, don't filter by project
			if showAll {
				projectID = ""
			}

			client := api.NewClient()
			memories, err := client.ListMemories(projectID, "") // No search query
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Filter by tag if requested
			if tagFilter != "" {
				var filtered []models.Memory
				for _, m := range memories {
					tags := getTagsAsStrings(m.Tags)
					for _, tag := range tags {
						if strings.EqualFold(tag, tagFilter) {
							filtered = append(filtered, m)
							break
						}
					}
				}
				memories = filtered
			}

			// Filter by org-only if requested
			if orgOnly {
				var filtered []models.Memory
				for _, m := range memories {
					if m.Project != nil && m.Project.Organization != nil {
						filtered = append(filtered, m)
					}
				}
				memories = filtered
			}

			if len(memories) == 0 {
				fmt.Println("No memories found.")
				return nil
			}

			// Apply limit if specified
			if limit > 0 && len(memories) > limit {
				memories = memories[:limit]
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTAGS\tCONTENT")
			fmt.Fprintln(w, "--\t----\t-------")
			for _, m := range memories {
				tagsStr := "-"
				tags := getTagsAsStrings(m.Tags)
				if len(tags) > 0 {
					tagsStr = strings.Join(tags, ",")
					if len(tagsStr) > 15 {
						tagsStr = tagsStr[:12] + "..."
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", m.ID.String()[:8], tagsStr, truncateString(m.Content, 55))
			}
			w.Flush()
			return nil
		},
	}
}

// recallCmd searches memory items.
func recallCmd() *cli.Command {
	return &cli.Command{
		Name:                   "recall",
		Usage:                  "Search within your memories",
		ArgsUsage:              "[search-query]",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID (default: search all projects)",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Limit number of results",
				Value:   20,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("a search query is required")
			}
			query := c.Args().First()
			projectID := c.String("project")
			limit := c.Int("limit")

			// If project not specified, use active project
			if projectID == "" && !c.IsSet("project") {
				cfg, err := config.LoadConfig()
				if err == nil && cfg.ActiveProjectID != "" {
					projectID = cfg.ActiveProjectID
				}
			}

			client := api.NewClient()
			memories, err := client.ListMemories(projectID, query)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Apply limit
			if limit > 0 && len(memories) > limit {
				memories = memories[:limit]
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
				fmt.Println(apierrors.ParseAPIError(err))
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
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("🗑️ Memory %s forgotten successfully.\n", memoryID[:8])
			return nil
		},
	}
}

// getTagsAsStrings converts interface{} tags to []string
func getTagsAsStrings(tags interface{}) []string {
	if tags == nil {
		return nil
	}

	// Try []interface{} first (common JSON unmarshaling result)
	if arr, ok := tags.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	// Try []string directly
	if arr, ok := tags.([]string); ok {
		return arr
	}

	return nil
}
