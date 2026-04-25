package commands

import (
	"github.com/kutbudev/ramorie-cli/internal/cli/tui"
	"github.com/urfave/cli/v2"
)

// NewUICommand returns the `ramorie ui` top-level command — the interactive
// 3-pane TUI navigator (sidebar / list / detail).
func NewUICommand() *cli.Command {
	return &cli.Command{
		Name:  "ui",
		Usage: "Interactive 3-pane TUI navigator (Yazi-style)",
		Action: func(c *cli.Context) error {
			return tui.Run()
		},
	}
}
