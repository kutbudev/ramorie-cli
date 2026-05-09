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

// =============================================================================
// PR4 (mayis 2026) — Pack assemble + bulk ops + clone CLI commands
// =============================================================================

// contextPackUseCmd assembles a pack and prints the bundle to stdout.
// "Gemini Gem"-style: pipe into your agent or copy/paste manually.
//
//	ramorie pack use <id-or-name> --format xml --budget 4000
func contextPackUseCmd() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Aliases:   []string{"render"},
		Usage:     "Assemble a context pack and print the bundle (XML/JSON/MD) to stdout",
		ArgsUsage: "[pack-id-or-name]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "xml", Usage: "Output format: xml, json, markdown"},
			&cli.IntFlag{Name: "budget", Aliases: []string{"b"}, Value: 4000, Usage: "Token budget for the bundle"},
			&cli.StringSliceFlag{Name: "section", Usage: "Sections to include: memories, tasks, contexts (default all)"},
			&cli.BoolFlag{Name: "include-archived", Usage: "Include archived items"},
			&cli.BoolFlag{Name: "no-cache", Usage: "Skip the 6h render cache"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("pack id or name required")
			}
			ident := c.Args().First()
			client := api.NewClient()

			// Resolve name → id when arg is not a UUID.
			if !looksLikeUUID(ident) {
				resolved, err := client.ResolvePackByName(ident)
				if err != nil {
					return fmt.Errorf("resolve pack: %w", err)
				}
				ident = resolved.ID
			}

			useCache := !c.Bool("no-cache")
			opts := api.AssembleOptions{
				Format:          c.String("format"),
				MaxTokens:       c.Int("budget"),
				Sections:        c.StringSlice("section"),
				IncludeArchived: c.Bool("include-archived"),
				UseCache:        &useCache,
			}
			resp, err := client.AssembleContextPack(ident, opts)
			if err != nil {
				return fmt.Errorf("assemble: %w", err)
			}

			fmt.Print(resp.Bundle)
			fmt.Println()
			fmt.Fprintf(c.App.ErrWriter, "─── %d/%d items, %d/%d tokens, cache %v, %dms ───\n",
				resp.Meta.ItemsReturned, resp.Meta.ItemsTotal,
				resp.Meta.TokensEst, resp.Meta.TokenBudget,
				resp.Meta.CacheHit, resp.Meta.LatencyMs)
			return nil
		},
	}
}

// contextPackAddCmd bulk-links memories or tasks to a pack.
//
//	ramorie pack add <pack> --memory id1 id2  --task id3 id4
func contextPackAddCmd() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add memories and/or tasks to a context pack (bulk)",
		ArgsUsage: "[pack-id]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "memory", Aliases: []string{"m"}, Usage: "Memory IDs to link (repeat for multiple)"},
			&cli.StringSliceFlag{Name: "task", Aliases: []string{"t"}, Usage: "Task IDs to link (repeat for multiple)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("pack id required")
			}
			packID := c.Args().First()
			memIDs := c.StringSlice("memory")
			taskIDs := c.StringSlice("task")
			if len(memIDs) == 0 && len(taskIDs) == 0 {
				return fmt.Errorf("at least one --memory or --task required")
			}
			client := api.NewClient()
			if len(memIDs) > 0 {
				if _, err := client.BulkAddMemoriesToPack(packID, memIDs); err != nil {
					return fmt.Errorf("add memories: %w", err)
				}
				fmt.Printf("✅ Linked %d memories to pack %s\n", len(memIDs), packID[:8])
			}
			if len(taskIDs) > 0 {
				if _, err := client.BulkAddTasksToPack(packID, taskIDs); err != nil {
					return fmt.Errorf("add tasks: %w", err)
				}
				fmt.Printf("✅ Linked %d tasks to pack %s\n", len(taskIDs), packID[:8])
			}
			return nil
		},
	}
}

// contextPackRemoveCmd bulk-unlinks memories or tasks.
//
//	ramorie pack remove <pack> --memory id1 id2
func contextPackRemoveCmd() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Aliases:   []string{"unlink"},
		Usage:     "Remove memories and/or tasks from a context pack (bulk)",
		ArgsUsage: "[pack-id]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "memory", Aliases: []string{"m"}, Usage: "Memory IDs to unlink"},
			&cli.StringSliceFlag{Name: "task", Aliases: []string{"t"}, Usage: "Task IDs to unlink"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("pack id required")
			}
			packID := c.Args().First()
			memIDs := c.StringSlice("memory")
			taskIDs := c.StringSlice("task")
			if len(memIDs) == 0 && len(taskIDs) == 0 {
				return fmt.Errorf("at least one --memory or --task required")
			}
			client := api.NewClient()
			if len(memIDs) > 0 {
				if err := client.BulkRemoveMemoriesFromPack(packID, memIDs); err != nil {
					return fmt.Errorf("remove memories: %w", err)
				}
				fmt.Printf("🗑️  Unlinked %d memories from pack %s\n", len(memIDs), packID[:8])
			}
			if len(taskIDs) > 0 {
				if err := client.BulkRemoveTasksFromPack(packID, taskIDs); err != nil {
					return fmt.Errorf("remove tasks: %w", err)
				}
				fmt.Printf("🗑️  Unlinked %d tasks from pack %s\n", len(taskIDs), packID[:8])
			}
			return nil
		},
	}
}

// contextPackCloneCmd clones a pack into a fresh draft.
//
//	ramorie pack clone <id> --name "yeni isim"
func contextPackCloneCmd() *cli.Command {
	return &cli.Command{
		Name:      "clone",
		Usage:     "Clone an existing context pack",
		ArgsUsage: "[pack-id]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Name for the cloned pack (default: '<source> (copy)')"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("pack id required")
			}
			packID := c.Args().First()
			client := api.NewClient()
			clone, err := client.CloneContextPack(packID, c.String("name"))
			if err != nil {
				return fmt.Errorf("clone: %w", err)
			}
			fmt.Printf("✅ Cloned to '%s' (id %s)\n", clone.Name, clone.ID[:8])
			return nil
		},
	}
}

// looksLikeUUID — naive shape check (8-4-4-4-12). Avoids importing
// google/uuid here when we just need a heuristic to gate name resolution.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
