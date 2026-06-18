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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "accent",
				Usage: "accent color: auto (follow terminal theme) | brand | <ansi 0-15> | <#hex>",
			},
			&cli.StringFlag{
				Name:  "icons",
				Usage: "icon set: nerd | unicode | auto (env RAMORIE_ICONS / config also honored)",
			},
		},
		Action: func(c *cli.Context) error {
			return tui.Run(tui.RunOptions{
				Accent: c.String("accent"),
				Icons:  c.String("icons"),
			})
		},
	}
}
