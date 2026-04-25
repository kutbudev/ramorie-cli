package commands

import (
	"fmt"
	"slices"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/urfave/cli/v2"
)

// NewContextCommand creates all subcommands for the 'context' command group.
func NewContextCommand() *cli.Command {
	return &cli.Command{
		Name:    "context",
		Aliases: []string{"ctx", "contexts"},
		Usage:   "Manage contexts",
		Subcommands: []*cli.Command{
			contextCreateCmd(),
			contextListCmd(),
			contextDeleteCmd(),
			{
				Name:    "packs",
				Aliases: []string{"pack"},
				Usage:   "Manage context packs (bundles of contexts)",
				Subcommands: []*cli.Command{
					contextPackListCmd(),
					contextPackCreateCmd(),
					contextPackDeleteCmd(),
				},
			},
		},
	}
}

// contextCreateCmd creates a new context.
func contextCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new context",
		ArgsUsage: "[name]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Context description"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("context name is required")
			}
			name := c.Args().First()
			description := c.String("description")

			client := api.NewClient()
			context, err := client.CreateContext(name, description)
			if err != nil {
				fmt.Printf("Error creating context: %v\n", err)
				return err
			}

			fmt.Printf("✅ Context '%s' created successfully!\n", context.Name)
			return nil
		},
	}
}

// contextListCmd lists all contexts.
func contextListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all available contexts",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "newest-first", Usage: "Show newest item at the top (default: oldest at top)"},
		},
		Action: func(c *cli.Context) error {
			newestFirst := c.Bool("newest-first")
			client := api.NewClient()
			contexts, err := client.ListContexts()
			if err != nil {
				fmt.Printf("Error listing contexts: %v\n", err)
				return err
			}

			if len(contexts) == 0 {
				fmt.Println(display.Dim.Render("  no contexts — use `ramorie context create` to add one"))
				return nil
			}

			// Default: chronological asc (oldest top, newest bottom).
			if !newestFirst {
				slices.Reverse(contexts)
			}

			cols := []display.Column{
				{Title: "ACTIVE", Min: 4, Weight: 0},
				{Title: "ID", Min: 8, Weight: 0},
				{Title: "NAME", Min: 16, Weight: 1},
				{Title: "DESCRIPTION", Min: 20, Weight: 4},
			}
			rows := make([][]string, 0, len(contexts))
			for _, ctx := range contexts {
				active := ""
				if ctx.IsActive {
					active = display.Good.Render("✓")
				}
				desc := ""
				if ctx.Description != nil {
					desc = *ctx.Description
				}
				rows = append(rows, []string{
					active,
					display.Dim.Render(ctx.ID.String()[:8]),
					ctx.Name,
					display.SingleLine(desc),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
			return nil
		},
	}
}

// contextDeleteCmd deletes a context.
func contextDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a context",
		ArgsUsage: "[context-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("context ID is required")
			}
			contextID := c.Args().First()

			client := api.NewClient()
			err := client.DeleteContext(contextID)
			if err != nil {
				fmt.Printf("Error deleting context: %v\n", err)
				return err
			}

			fmt.Printf("🗑️ Context %s deleted successfully.\n", contextID[:8])
			return nil
		},
	}
}
