package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// NewDecisionCommand creates all subcommands for the 'decision' command group.
// NOTE: The /v1/decisions backend routes have been removed. These CLI commands are
// deprecated and will be removed in a future release. Use `ramorie memory` instead.
func NewDecisionCommand() *cli.Command {
	return &cli.Command{
		Name:    "decision",
		Aliases: []string{"dec", "adr", "decisions"},
		Usage:   "[DEPRECATED] Manage architectural decisions (ADRs) — use `ramorie memory` instead",
		Subcommands: []*cli.Command{
			decisionListCmd(),
			decisionCreateCmd(),
			decisionShowCmd(),
			decisionUpdateCmd(),
			decisionDeleteCmd(),
		},
	}
}

const decisionDeprecatedMsg = `[DEPRECATED] The decision CLI commands have been removed.
The /v1/decisions backend routes no longer exist.

Use the memory commands instead:
  ramorie memory list   --project <name>          # list memories (filter by type=decision)
  ramorie remember "decided to use X because Y"  # auto-detects type=decision

Or via MCP: remember() or memory(action="list", type="decision")`

func decisionListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "[DEPRECATED] List architectural decisions",
		Action: func(c *cli.Context) error {
			fmt.Println(decisionDeprecatedMsg)
			return nil
		},
	}
}

func decisionCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "[DEPRECATED] Create a new architectural decision",
		Action: func(c *cli.Context) error {
			fmt.Println(decisionDeprecatedMsg)
			return nil
		},
	}
}

func decisionShowCmd() *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"view", "info"},
		Usage:   "[DEPRECATED] Show details for a decision",
		Action: func(c *cli.Context) error {
			fmt.Println(decisionDeprecatedMsg)
			return nil
		},
	}
}

func decisionUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "[DEPRECATED] Update a decision's properties",
		Action: func(c *cli.Context) error {
			fmt.Println(decisionDeprecatedMsg)
			return nil
		},
	}
}

func decisionDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Usage:   "[DEPRECATED] Delete a decision",
		Action: func(c *cli.Context) error {
			fmt.Println(decisionDeprecatedMsg)
			return nil
		},
	}
}
