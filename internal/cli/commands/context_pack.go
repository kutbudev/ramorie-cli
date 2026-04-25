package commands

import (
	"fmt"
	"slices"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/urfave/cli/v2"
)

// contextPackListCmd lists all context packs.
// Used as a subcommand under `ramorie context packs`.
func contextPackListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all context packs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "Filter by type (project, integration, decision, custom)"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status (draft, published)"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 20, Usage: "Limit results"},
			&cli.BoolFlag{Name: "newest-first", Usage: "Show newest item at the top (default: oldest at top)"},
		},
		Action: func(c *cli.Context) error {
			newestFirst := c.Bool("newest-first")
			client := api.NewClient()
			response, err := client.ListContextPacks(
				c.String("type"),
				c.String("status"),
				"",
				c.Int("limit"),
				0,
			)
			if err != nil {
				fmt.Printf("Error listing context packs: %v\n", err)
				return err
			}

			if len(response.ContextPacks) == 0 {
				fmt.Println(display.Dim.Render("  no context packs — use `ramorie context packs create` to add one"))
				return nil
			}

			// Default: chronological asc (oldest top, newest bottom).
			if !newestFirst {
				slices.Reverse(response.ContextPacks)
			}

			cols := []display.Column{
				{Title: "ACTIVE", Min: 4, Weight: 0},
				{Title: "ID", Min: 8, Weight: 0},
				{Title: "NAME", Min: 16, Weight: 1},
				{Title: "TYPE", Min: 10, Weight: 0},
				{Title: "STATUS", Min: 8, Weight: 0},
				{Title: "CTX", Min: 4, Weight: 0},
				{Title: "DESCRIPTION", Min: 20, Weight: 3},
			}
			rows := make([][]string, 0, len(response.ContextPacks))
			for _, pack := range response.ContextPacks {
				active := ""
				if pack.Status == "published" {
					active = display.Good.Render("✓")
				}
				desc := ""
				if pack.Description != nil {
					desc = *pack.Description
				}
				rows = append(rows, []string{
					active,
					display.Dim.Render(pack.ID[:8]),
					pack.Name,
					display.Dim.Render(pack.Type),
					display.Dim.Render(pack.Status),
					display.Dim.Render(fmt.Sprintf("%d", pack.ContextsCount)),
					display.SingleLine(desc),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
			return nil
		},
	}
}

// contextPackCreateCmd creates a new context pack.
func contextPackCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new context pack",
		ArgsUsage: "[name]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Value: "custom", Usage: "Pack type (project, integration, decision, custom)"},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Pack description"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Value: "draft", Usage: "Pack status (draft, published)"},
			&cli.StringSliceFlag{Name: "tags", Usage: "Tags for the pack"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("context pack name is required")
			}
			name := c.Args().First()

			client := api.NewClient()
			pack, err := client.CreateContextPack(
				name,
				c.String("type"),
				c.String("description"),
				c.String("status"),
				c.StringSlice("tags"),
			)
			if err != nil {
				fmt.Printf("Error creating context pack: %v\n", err)
				return err
			}

			fmt.Printf("✅ Context pack '%s' created successfully!\n", pack.Name)
			fmt.Printf("   ID: %s\n", pack.ID[:8])
			fmt.Printf("   Type: %s\n", pack.Type)
			fmt.Printf("   Status: %s\n", pack.Status)
			return nil
		},
	}
}

// contextPackDeleteCmd deletes a context pack.
func contextPackDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm"},
		Usage:     "Delete a context pack",
		ArgsUsage: "[pack-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("context pack ID is required")
			}
			packID := c.Args().First()

			client := api.NewClient()
			if err := client.DeleteContextPack(packID); err != nil {
				fmt.Printf("Error deleting context pack: %v\n", err)
				return err
			}

			fmt.Printf("🗑️ Context pack %s deleted successfully.\n", packID[:8])
			return nil
		},
	}
}
