package commands

import (
	"fmt"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewFocusCommand creates all subcommands for the 'focus' command group.
func NewFocusCommand() *cli.Command {
	return &cli.Command{
		Name:  "focus",
		Usage: "Manage user focus (active context pack)",
		Action: func(c *cli.Context) error {
			// Default action is to show current focus
			return showFocus(c)
		},
		Subcommands: []*cli.Command{
			focusSetCmd(),
			focusClearCmd(),
		},
	}
}

// showFocus displays the current focus state
func showFocus(c *cli.Context) error {
	client := api.NewClient()
	focus, err := client.GetFocus()
	if err != nil {
		fmt.Printf("Error getting focus: %v\n", err)
		return err
	}

	if focus.ActiveContextPackID == nil || focus.ActivePack == nil {
		fmt.Println("No active focus set.")
		fmt.Println("💡 Use 'ramorie focus set <pack-id>' to set your focus.")
		return nil
	}

	pack := focus.ActivePack
	fmt.Printf("🎯 Current Focus:\n")
	fmt.Printf("   Pack: %s\n", pack.Name)
	fmt.Printf("   ID: %s\n", pack.ID[:8])
	fmt.Printf("   Type: %s\n", pack.Type)
	if pack.Description != nil {
		fmt.Printf("   Description: %s\n", *pack.Description)
	}
	fmt.Printf("\n📊 Statistics:\n")
	fmt.Printf("   Contexts: %d\n", pack.ContextsCount)
	fmt.Printf("   Memories: %d\n", pack.MemoriesCount)
	fmt.Printf("   Tasks: %d\n", pack.TasksCount)

	if len(pack.Contexts) > 0 {
		fmt.Printf("\n📦 Active Contexts:\n")
		for _, ctx := range pack.Contexts {
			fmt.Printf("   • %s (ID: %s)\n", ctx.Name, ctx.ID[:8])
		}
	}

	return nil
}

// focusSetCmd sets the active context pack
func focusSetCmd() *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "Set focus to a context pack",
		ArgsUsage: "[pack-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("context pack ID is required")
			}
			packID := c.Args().First()

			client := api.NewClient()
			focus, err := client.SetFocus(packID)
			if err != nil {
				fmt.Printf("Error setting focus: %v\n", err)
				return err
			}

			if focus.ActivePack != nil {
				fmt.Printf("✅ Focus set to '%s'\n", focus.ActivePack.Name)
				fmt.Printf("   ID: %s\n", focus.ActivePack.ID[:8])
				fmt.Printf("   %d contexts, %d memories, %d tasks are now active.\n",
					focus.ActivePack.ContextsCount,
					focus.ActivePack.MemoriesCount,
					focus.ActivePack.TasksCount)
			} else {
				fmt.Println("✅ Focus updated successfully.")
			}
			return nil
		},
	}
}

// focusClearCmd clears the active context pack
func focusClearCmd() *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "Clear current focus",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			if err := client.ClearFocus(); err != nil {
				fmt.Printf("Error clearing focus: %v\n", err)
				return err
			}

			fmt.Println("✅ Focus cleared successfully.")
			fmt.Println("💡 Use 'ramorie focus set <pack-id>' to set a new focus.")
			return nil
		},
	}
}
