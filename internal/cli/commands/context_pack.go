package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
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
		},
		Action: func(c *cli.Context) error {
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
				fmt.Println("No context packs found. Use 'ramorie context packs create' to add one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ACTIVE\tID\tNAME\tTYPE\tSTATUS\tCONTEXTS")
			fmt.Fprintln(w, "------\t--\t----\t----\t------\t--------")

			for _, pack := range response.ContextPacks {
				active := ""
				if pack.Status == "published" {
					active = "📦"
				}
				contextCount := 0 // TODO: Backend should return contexts count
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
					active,
					pack.ID[:8],
					truncateString(pack.Name, 30),
					pack.Type,
					pack.Status,
					contextCount)
			}
			w.Flush()
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
