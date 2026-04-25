package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewActivityCommand wraps the recent-activity endpoint (was `reports history`)
// with an optional --burndown flag that switches to the burndown report
// (was `reports burndown`).
func NewActivityCommand() *cli.Command {
	return &cli.Command{
		Name:  "activity",
		Usage: "Recent activity history (default) or --burndown report",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "burndown", Usage: "Show burndown report instead of activity feed"},
			&cli.IntFlag{Name: "days", Aliases: []string{"d"}, Usage: "How many days"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "Max items (activity only)", Value: 15},
			&cli.StringFlag{Name: "interval", Aliases: []string{"i"}, Usage: "daily or weekly (burndown only)", Value: "daily"},
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			project := c.String("project")

			if c.Bool("burndown") {
				days := c.Int("days")
				if days <= 0 {
					days = 30
				}
				interval := c.String("interval")

				params := url.Values{}
				if days > 0 {
					params.Set("days", fmt.Sprintf("%d", days))
				}
				if interval != "" {
					params.Set("interval", interval)
				}
				if project != "" {
					params.Set("project", project)
				}

				endpoint := "/reports/burndown"
				if encoded := params.Encode(); encoded != "" {
					endpoint += "?" + encoded
				}

				b, err := client.Request("GET", endpoint, nil)
				if err != nil {
					return err
				}

				var out interface{}
				if err := json.Unmarshal(b, &out); err != nil {
					os.Stdout.Write(b)
					os.Stdout.Write([]byte("\n"))
					return nil
				}

				pretty, _ := json.MarshalIndent(out, "", "  ")
				os.Stdout.Write(pretty)
				os.Stdout.Write([]byte("\n"))
				return nil
			}

			days := c.Int("days")
			if days <= 0 {
				days = 7
			}
			limit := c.Int("limit")

			params := url.Values{}
			if days > 0 {
				params.Set("days", fmt.Sprintf("%d", days))
			}
			if limit > 0 {
				params.Set("limit", fmt.Sprintf("%d", limit))
			}
			if project != "" {
				params.Set("project", project)
			}

			endpoint := "/reports/history"
			if encoded := params.Encode(); encoded != "" {
				endpoint += "?" + encoded
			}

			b, err := client.Request("GET", endpoint, nil)
			if err != nil {
				return err
			}

			var out interface{}
			if err := json.Unmarshal(b, &out); err != nil {
				os.Stdout.Write(b)
				os.Stdout.Write([]byte("\n"))
				return nil
			}

			pretty, _ := json.MarshalIndent(out, "", "  ")
			os.Stdout.Write(pretty)
			os.Stdout.Write([]byte("\n"))
			return nil
		},
	}
}
