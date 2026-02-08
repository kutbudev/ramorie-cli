package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/urfave/cli/v2"
)

// NewDecisionCommand creates all subcommands for the 'decision' command group.
func NewDecisionCommand() *cli.Command {
	return &cli.Command{
		Name:    "decision",
		Aliases: []string{"dec", "adr", "decisions"},
		Usage:   "Manage architectural decisions (ADRs)",
		Subcommands: []*cli.Command{
			decisionListCmd(),
			decisionCreateCmd(),
			decisionShowCmd(),
			decisionUpdateCmd(),
			decisionDeleteCmd(),
		},
	}
}

// decisionListCmd lists all decisions with optional filtering
func decisionListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List architectural decisions",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status (draft, proposed, approved, deprecated)"},
			&cli.StringFlag{Name: "area", Aliases: []string{"a"}, Usage: "Filter by area (Architecture, API, Database, Security, Infrastructure, Frontend, Backend, DevOps)"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 20, Usage: "Limit results"},
		},
		Action: func(c *cli.Context) error {
			status := c.String("status")
			area := c.String("area")
			limit := c.Int("limit")

			client := api.NewClient()
			decisions, err := client.ListDecisions("", status, area, limit) // Empty projectID to list all
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(decisions) == 0 {
				fmt.Println("No decisions found. Use 'ramorie decision create' to add one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ADR\tTITLE\tSTATUS\tAREA\tSOURCE")
			fmt.Fprintln(w, "---\t-----\t------\t----\t------")

			for _, d := range decisions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					d.ADRNumber,
					truncateString(d.Title, 40),
					d.Status,
					d.Area,
					d.Source)
			}
			w.Flush()
			return nil
		},
	}
}

// decisionCreateCmd creates a new decision
func decisionCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new architectural decision",
		ArgsUsage: "[title]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "area", Aliases: []string{"a"}, Value: "Architecture", Usage: "Decision area (Architecture, API, Database, Security, Infrastructure, Frontend, Backend, DevOps)"},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Decision description"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Value: "draft", Usage: "Decision status (draft, proposed, approved, deprecated)"},
			&cli.StringFlag{Name: "context", Aliases: []string{"c"}, Usage: "Decision context"},
			&cli.StringFlag{Name: "consequences", Usage: "Decision consequences"},
			&cli.StringFlag{Name: "source", Value: "user", Usage: "Decision source (user, agent, import)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("decision title is required")
			}
			title := c.Args().First()

			area := c.String("area")
			description := c.String("description")
			status := c.String("status")
			context := c.String("context")
			consequences := c.String("consequences")
			source := c.String("source")

			client := api.NewClient()
			decision, err := client.CreateDecision("", title, description, status, area, context, consequences, source) // Empty projectID
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Decision '%s' created successfully!\n", decision.Title)
			fmt.Printf("   ADR Number: %s\n", decision.ADRNumber)
			fmt.Printf("   ID: %s\n", decision.ID[:8])
			fmt.Printf("   Area: %s\n", decision.Area)
			fmt.Printf("   Status: %s\n", decision.Status)
			return nil
		},
	}
}

// decisionShowCmd shows details for a specific decision
func decisionShowCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Aliases:   []string{"view", "info"},
		Usage:     "Show details for a decision",
		ArgsUsage: "[id or ADR-XXX]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("decision ID or ADR number is required")
			}
			identifier := c.Args().First()

			client := api.NewClient()
			decision, err := client.GetDecision(identifier)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("Decision: %s\n", decision.Title)
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("ADR Number:   %s\n", decision.ADRNumber)
			fmt.Printf("ID:           %s\n", decision.ID)
			fmt.Printf("Title:        %s\n", decision.Title)
			fmt.Printf("Status:       %s\n", decision.Status)
			fmt.Printf("Area:         %s\n", decision.Area)
			fmt.Printf("Source:       %s\n", decision.Source)
			fmt.Printf("Created At:   %s\n", decision.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated At:   %s\n", decision.UpdatedAt.Format("2006-01-02 15:04:05"))

			if decision.Description != "" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Printf("Description:\n%s\n", decision.Description)
			}

			if decision.Context != nil && *decision.Context != "" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Printf("Context:\n%s\n", *decision.Context)
			}

			if decision.Consequences != nil && *decision.Consequences != "" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Printf("Consequences:\n%s\n", *decision.Consequences)
			}

			if decision.Content != nil && *decision.Content != "" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Printf("Content:\n%s\n", *decision.Content)
			}

			return nil
		},
	}
}

// decisionUpdateCmd updates an existing decision
func decisionUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a decision's properties",
		ArgsUsage: "[id]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "New title"},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "New description"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "New status (draft, proposed, approved, deprecated)"},
			&cli.StringFlag{Name: "area", Aliases: []string{"a"}, Usage: "New area (Architecture, API, Database, Security, Infrastructure, Frontend, Backend, DevOps)"},
			&cli.StringFlag{Name: "context", Aliases: []string{"c"}, Usage: "New context"},
			&cli.StringFlag{Name: "consequences", Usage: "New consequences"},
			&cli.StringFlag{Name: "content", Usage: "New content"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("decision ID is required")
			}
			decisionID := c.Args().First()

			updateData := map[string]interface{}{}

			if title := c.String("title"); title != "" {
				updateData["title"] = title
			}
			if description := c.String("description"); description != "" {
				updateData["description"] = description
			}
			if status := c.String("status"); status != "" {
				updateData["status"] = status
			}
			if area := c.String("area"); area != "" {
				updateData["area"] = area
			}
			if context := c.String("context"); context != "" {
				updateData["context"] = context
			}
			if consequences := c.String("consequences"); consequences != "" {
				updateData["consequences"] = consequences
			}
			if content := c.String("content"); content != "" {
				updateData["content"] = content
			}

			if len(updateData) == 0 {
				return fmt.Errorf("at least one flag is required to update")
			}

			client := api.NewClient()
			decision, err := client.UpdateDecision(decisionID, updateData)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Decision '%s' updated successfully.\n", decision.Title)
			fmt.Printf("   ADR: %s\n", decision.ADRNumber)
			fmt.Printf("   Status: %s\n", decision.Status)
			return nil
		},
	}
}

// decisionDeleteCmd deletes a decision
func decisionDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm"},
		Usage:     "Delete a decision",
		ArgsUsage: "[id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Force deletion without confirmation"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("decision ID is required")
			}
			decisionID := c.Args().First()

			// Optional: Add confirmation prompt if not forced
			if !c.Bool("force") {
				fmt.Printf("Are you sure you want to delete decision %s? (y/N): ", decisionID[:8])
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			client := api.NewClient()
			if err := client.DeleteDecision(decisionID); err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("🗑️  Decision %s deleted successfully.\n", decisionID[:8])
			return nil
		},
	}
}
