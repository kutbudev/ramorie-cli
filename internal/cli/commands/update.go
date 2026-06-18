package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/selfupdate"
	"github.com/kutbudev/ramorie-cli/internal/version"
	"github.com/urfave/cli/v2"
)

// NewUpdateCommand returns the `ramorie update` command — upgrades the CLI to
// the latest release. It is install-aware: a Homebrew or npm install delegates
// to that manager; a direct binary is replaced in place from GitHub releases.
func NewUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:    "update",
		Aliases: []string{"upgrade"},
		Usage:   "Update ramorie to the latest version",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "check", Usage: "Only check whether a newer version exists; don't install"},
			&cli.BoolFlag{Name: "force", Usage: "Reinstall the latest version even if already current"},
		},
		Action: func(c *cli.Context) error {
			current := version.Version

			if c.Bool("check") {
				ctx, cancel := context.WithTimeout(c.Context, 10*time.Second)
				defer cancel()
				latest, err := selfupdate.LatestVersion(ctx)
				if err != nil {
					return fmt.Errorf("could not reach GitHub releases: %w", err)
				}
				if selfupdate.Newer(current, latest) {
					method, _ := selfupdate.DetectMethod()
					fmt.Printf("%s  current %s · latest %s  (installed via %s)\n",
						display.Warn.Render("update available"), current,
						display.Title.Render(latest), method)
					fmt.Println(display.Dim.Render("  run `ramorie update` to upgrade"))
				} else {
					fmt.Printf("%s ramorie is up to date (%s)\n", display.Good.Render("✓"), current)
				}
				return nil
			}

			// Downloads can take a little while on slow links.
			ctx, cancel := context.WithTimeout(c.Context, 5*time.Minute)
			defer cancel()
			return selfupdate.Update(ctx, current, c.Bool("force"), os.Stdout)
		},
	}
}
