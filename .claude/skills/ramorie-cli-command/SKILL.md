---
name: ramorie-cli-command
description: Use when creating new CLI commands or subcommands in Ramorie CLI - provides exact patterns for urfave/cli command structure, flags, API client usage, output formatting, and error handling
---

# Ramorie CLI Command Patterns

## Overview

CLI commands use **urfave/cli v2** with a consistent pattern: command group function returning `*cli.Command`, subcommand functions, API client initialization, and tabwriter output.

## Command Group Template

```go
// internal/cli/commands/feature.go
package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/urfave/cli/v2"
)

// NewFeatureCommand creates the 'feature' command group.
func NewFeatureCommand() *cli.Command {
	return &cli.Command{
		Name:    "feature",
		Aliases: []string{"f"},
		Usage:   "Manage features",
		Subcommands: []*cli.Command{
			featureListCmd(),
			featureCreateCmd(),
			featureShowCmd(),
			featureUpdateCmd(),
			featureDeleteCmd(),
		},
	}
}
```

## List Subcommand Template

```go
func featureListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List features",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Filter by project"},
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "Limit results", Value: 0},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()

			// Resolve project name/short-id to UUID
			projectArg := c.String("project")
			var projectID string
			if projectArg != "" {
				projectID = resolveProjectID(client, projectArg)
				if projectID == "" {
					return fmt.Errorf("project '%s' not found", projectArg)
				}
			}

			features, err := client.ListFeatures(projectID, c.String("status"))
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(features) == 0 {
				fmt.Println("No features found.")
				return nil
			}

			// Apply limit
			limit := c.Int("limit")
			if limit > 0 && len(features) > limit {
				features = features[:limit]
			}

			// Output with tabwriter
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS")
			fmt.Fprintln(w, "--\t-----\t------")

			for _, f := range features {
				title := f.Title
				// Handle encryption
				if f.IsEncrypted && crypto.IsVaultUnlocked() {
					if decrypted, err := crypto.DecryptContent(f.EncryptedTitle, f.TitleNonce, true); err == nil {
						title = decrypted
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					f.ID.String()[:8], // Short ID
					truncate(title, 50),
					f.Status,
				)
			}
			w.Flush()
			return nil
		},
	}
}
```

## Create Subcommand Template

```go
func featureCreateCmd() *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"new", "add"},
		Usage:   "Create a new feature",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID", Required: true},
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "Feature title"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()

			// Get title from flag or first arg
			title := c.String("title")
			if title == "" && c.NArg() > 0 {
				title = c.Args().First()
			}
			if title == "" {
				return fmt.Errorf("title is required")
			}

			// Resolve project
			projectID := resolveProjectID(client, c.String("project"))
			if projectID == "" {
				return fmt.Errorf("project not found")
			}

			feature, err := client.CreateFeature(projectID, title)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("Created feature: %s (%s)\n", feature.Title, feature.ID.String()[:8])
			return nil
		},
	}
}
```

## Show (Detail) Subcommand

```go
func featureShowCmd() *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"info", "get"},
		Usage:   "Show feature details",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("feature ID is required")
			}

			client := api.NewClient()
			featureID := c.Args().First()

			feature, err := client.GetFeature(featureID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Decrypt if needed
			title := feature.Title
			if feature.IsEncrypted && crypto.IsVaultUnlocked() {
				if d, err := crypto.DecryptContent(feature.EncryptedTitle, feature.TitleNonce, true); err == nil {
					title = d
				}
			}

			fmt.Printf("ID:      %s\n", feature.ID)
			fmt.Printf("Title:   %s\n", title)
			fmt.Printf("Status:  %s\n", feature.Status)
			fmt.Printf("Created: %s\n", feature.CreatedAt.Format("2006-01-02 15:04"))
			return nil
		},
	}
}
```

## Registering in main.go

```go
// cmd/ramorie/main.go
app := &cli.App{
	Name:    "ramorie",
	Usage:   "AI-powered task and memory management CLI",
	Version: Version,
	Commands: []*cli.Command{
		// ... existing commands
		commands.NewFeatureCommand(),
	},
}
```

## Helper Functions (utils.go)

```go
// Resolve project name/short-ID to full UUID
func resolveProjectID(client *api.Client, arg string) string {
	projects, err := client.ListProjects()
	if err != nil {
		return ""
	}
	for _, p := range projects {
		if p.ID.String() == arg ||
			strings.HasPrefix(p.ID.String(), arg) ||
			strings.EqualFold(p.Name, arg) {
			return p.ID.String()
		}
	}
	return ""
}

// Truncate string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
```

## Flag Types Reference

```go
&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "desc", Required: true}
&cli.IntFlag{Name: "limit", Value: 10, Usage: "desc"}
&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "desc"}
&cli.Float64Flag{Name: "progress", Usage: "desc"}
```

## Error Handling Pattern

```go
result, err := client.DoSomething(args)
if err != nil {
	fmt.Println(apierrors.ParseAPIError(err))
	return err
}
```

## Common Mistakes

- Not registering new command in `main.go`'s `Commands` slice
- Forgetting aliases (`[]string{"ls"}`) for common operations
- Not handling encrypted content (check `IsEncrypted` + vault state)
- Using `fmt.Println` for errors instead of `apierrors.ParseAPIError()`
- Not resolving project name to UUID before API calls
- Forgetting to flush `tabwriter` (`w.Flush()`)
- Not handling `c.NArg() == 0` for required positional arguments
