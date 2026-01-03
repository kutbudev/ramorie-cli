package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewContextCommand creates all subcommands for the 'context' command group.
func NewContextCommand() *cli.Command {
	return &cli.Command{
		Name:    "context",
		Aliases: []string{"ctx"},
		Usage:   "Manage contexts",
		Subcommands: []*cli.Command{
			contextCreateCmd(),
			contextListCmd(),
			contextUseCmd(),
			contextDeleteCmd(),
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
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			contexts, err := client.ListContexts()
			if err != nil {
				fmt.Printf("Error listing contexts: %v\n", err)
				return err
			}

			if len(contexts) == 0 {
				fmt.Println("No contexts found. Use 'ramorie context create' to add one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ACTIVE\tID\tNAME\tDESCRIPTION")
			fmt.Fprintln(w, "------\t--\t----\t-----------")

			for _, ctx := range contexts {
				active := ""
				if ctx.IsActive {
					active = "✅"
				}
				desc := ""
				if ctx.Description != nil {
					desc = *ctx.Description
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					active,
					ctx.ID.String()[:8],
					ctx.Name,
					truncateString(desc, 40))
			}
			w.Flush()
			return nil
		},
	}
}

// contextUseCmd sets a context as active.
func contextUseCmd() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Usage:     "Set the active context",
		ArgsUsage: "[context-name]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("context name is required")
			}
			name := c.Args().First()

			client := api.NewClient()
			context, err := client.UseContext(name)
			if err != nil {
				fmt.Printf("Error setting active context: %v\n", err)
				return err
			}

			fmt.Printf("✅ Active context set to '%s'\n", context.Name)
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
