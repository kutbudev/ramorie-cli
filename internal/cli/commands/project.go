package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/cli/resolve"
	"github.com/urfave/cli/v2"
)

// NewProjectCommand creates all subcommands for the 'project' command group.
func NewProjectCommand() *cli.Command {
	return &cli.Command{
		Name:    "project",
		Aliases: []string{"p", "projects"},
		Usage:   "Manage projects",
		Subcommands: []*cli.Command{
			projectListCmd(),
			projectCreateCmd(),
			projectShowCmd(),
			projectDeleteCmd(),
			projectUpdateCmd(),
			projectSetEncryptionCmd(),
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
				fmt.Println("No projects found. Use 'ramorie project create' to add one.")
				return nil
			}

			cols := []display.Column{
				{Title: "ID", Min: 36, Weight: 0}, // full UUID — fixed
				{Title: "NAME", Min: 16, Weight: 1},
				{Title: "DESCRIPTION", Min: 24, Weight: 3}, // dropped on narrow terminals
			}
			rows := make([][]string, 0, len(projects))
			for _, p := range projects {
				rows = append(rows, []string{
					p.ID.String(),
					p.Name,
					display.SingleLine(p.Description),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
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
			arg := c.Args().First()
			client := api.NewClient()
			projectID, err := resolve.ResolveProject(arg, client)
			if err != nil {
				return err
			}
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
			if project.Configuration != nil && len(project.Configuration) > 0 {
				configJSON, err := json.MarshalIndent(project.Configuration, "", "  ")
				if err == nil {
					fmt.Printf("Configuration: \n%s\n", string(configJSON))
				}
			}
			fmt.Printf("Created At:  %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated At:  %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))
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
			arg := c.Args().First()
			client := api.NewClient()
			projectID, err := resolve.ResolveProject(arg, client)
			if err != nil {
				return err
			}
			err = client.DeleteProject(projectID)
			if err != nil {
				fmt.Printf("Error deleting project: %v\n", err)
				return err
			}

			fmt.Printf("🗑️ Project %s deleted successfully.\n", projectID)
			return nil
		},
	}
}

// projectUpdateCmd updates a project.
func projectUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a project's properties",
		ArgsUsage: "[project-id]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "New project name",
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "New project description",
			},
			&cli.StringFlag{
				Name:  "config-json-string",
				Usage: "Project configuration as a JSON string",
			},
			&cli.PathFlag{
				Name:  "config-json-file",
				Usage: "Path to a file containing project configuration as JSON",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("project ID is required")
			}
			arg := c.Args().First()

			updateData := make(map[string]interface{})

			if name := c.String("name"); name != "" {
				updateData["name"] = name
			}
			if description := c.String("description"); description != "" {
				updateData["description"] = description
			}

			configJSON := c.String("config-json-string")
			configFilePath := c.Path("config-json-file")

			if configJSON != "" && configFilePath != "" {
				return fmt.Errorf("please provide configuration using either --config-json-string or --config-json-file, not both")
			}

			if configFilePath != "" {
				fileBytes, err := os.ReadFile(configFilePath)
				if err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}
				configJSON = string(fileBytes)
			}

			if configJSON != "" {
				updateData["configuration"] = json.RawMessage(configJSON)
			}

			if len(updateData) == 0 {
				fmt.Println("No update fields provided.")
				return nil
			}

			client := api.NewClient()
			projectID, err := resolve.ResolveProject(arg, client)
			if err != nil {
				return err
			}
			project, err := client.UpdateProject(projectID, updateData)
			if err != nil {
				fmt.Printf("Error updating project: %v\n", err)
				return err
			}

			fmt.Printf("✅ Project '%s' (ID: %s) updated successfully.\n", project.Name, project.ID.String()[:8])
			return nil
		},
	}
}

// projectSetEncryptionCmd toggles a project's encryption_required flag.
//
// This is the operator-facing fix for the encryption_required deadlock:
// when a project requires encryption but the account has encryption
// disabled, writes fail with ENCRYPTION_REQUIRED and unlocking can't help.
// `ramorie project set-encryption <project> false` clears the requirement.
func projectSetEncryptionCmd() *cli.Command {
	return &cli.Command{
		Name:      "set-encryption",
		Usage:     "Set whether a project requires encrypted writes",
		ArgsUsage: "[project] [true|false]",
		Description: "Toggle the per-project encryption_required flag.\n" +
			"   true  → memories/tasks must be written encrypted (vault must be unlocked).\n" +
			"   false → plaintext writes are allowed (use this if your account encryption is disabled).\n\n" +
			"   Example: ramorie project set-encryption my-project false",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: ramorie project set-encryption <project> <true|false>")
			}
			arg := c.Args().Get(0)
			rawVal := c.Args().Get(1)

			required, perr := strconv.ParseBool(rawVal)
			if perr != nil {
				return fmt.Errorf("invalid value %q: expected true or false", rawVal)
			}

			client := api.NewClient()
			projectID, err := resolve.ResolveProject(arg, client)
			if err != nil {
				return err
			}

			project, err := client.UpdateProject(projectID, map[string]interface{}{
				"encryption_required": required,
			})
			if err != nil {
				fmt.Printf("Error updating project encryption: %v\n", err)
				return err
			}

			if required {
				fmt.Printf("🔒 Project '%s' now REQUIRES encrypted writes. Make sure your vault is unlocked (`ramorie setup unlock`).\n", project.Name)
			} else {
				fmt.Printf("🔓 Project '%s' no longer requires encryption — plaintext writes are now allowed.\n", project.Name)
			}
			return nil
		},
	}
}
