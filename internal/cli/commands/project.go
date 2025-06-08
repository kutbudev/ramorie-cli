package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/urfave/cli/v2"
)

// NewProjectCommand creates all subcommands for the 'project' command group.
func NewProjectCommand() *cli.Command {
	return &cli.Command{
		Name:    "project",
		Aliases: []string{"p"},
		Usage:   "Manage projects",
		Subcommands: []*cli.Command{
			projectListCmd(),
			projectCreateCmd(),
			projectShowCmd(),
			projectUseCmd(),
			projectDeleteCmd(),
		},
	}
}

// projectListCmd lists all projects.
func projectListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all projects",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			projects, err := client.ListProjects()
			if err != nil {
				fmt.Printf("Error listing projects: %v\n", err)
				return err
			}

			if len(projects) == 0 {
				fmt.Println("No projects found. Use 'jbraincli project create' to add one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ACTIVE\tID\tNAME\tDESCRIPTION")
			fmt.Fprintln(w, "------\t--\t----\t-----------")

			for _, p := range projects {
				active := ""
				if p.IsActive {
					active = "✅"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					active,
					p.ID.String()[:8],
					p.Name,
					truncateString(p.Description, 40))
			}
			w.Flush()
			return nil
		},
	}
}

// projectCreateCmd creates a new project.
func projectCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new project",
		ArgsUsage: "[name]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Project description",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("project name is required")
			}
			name := c.Args().First()
			description := c.String("description")

			client := api.NewClient()
			project, err := client.CreateProject(name, description)
			if err != nil {
				fmt.Printf("Error creating project: %v\n", err)
				return err
			}

			fmt.Printf("✅ Project '%s' created successfully!\n", project.Name)
			fmt.Printf("ID: %s\n", project.ID.String())
			return nil
		},
	}
}

// projectShowCmd shows details for a specific project.
func projectShowCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "Show details for a project",
		ArgsUsage: "[project-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("project ID is required")
			}
			projectID := c.Args().First()

			client := api.NewClient()
			project, err := client.GetProject(projectID)
			if err != nil {
				fmt.Printf("Error getting project: %v\n", err)
				return err
			}

			fmt.Printf("Project Details for '%s':\n", project.Name)
			fmt.Printf("----------------------------------\n")
			fmt.Printf("ID:          %s\n", project.ID.String())
			fmt.Printf("Name:        %s\n", project.Name)
			fmt.Printf("Description: %s\n", project.Description)
			fmt.Printf("Created At:  %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated At:  %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

// projectUseCmd sets a project as the active one.
func projectUseCmd() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Usage:     "Set the active project",
		ArgsUsage: "[project-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("project ID is required")
			}
			projectID := c.Args().First()

			client := api.NewClient()
			if err := client.SetProjectActive(projectID); err != nil {
				fmt.Printf("Error setting active project: %v\n", err)
				return err
			}

			fmt.Printf("✅ Active project set to '%s'\n", projectID)
			return nil
		},
	}
}

// projectDeleteCmd deletes a project.
func projectDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a project",
		ArgsUsage: "[project-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("project ID is required")
			}
			projectID := c.Args().First()

			client := api.NewClient()
			err := client.DeleteProject(projectID)
			if err != nil {
				fmt.Printf("Error deleting project: %v\n", err)
				return err
			}

			fmt.Printf("🗑️ Project %s deleted successfully.\n", projectID)
			return nil
		},
	}
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}