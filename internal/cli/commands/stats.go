package commands

import (
	"encoding/json"
	"net/url"
	"os"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewStatsCommand wraps the stats endpoint that reports.go's `stats` subcommand
// already exposes. Promoted to top-level for v6 — daily verb, doesn't deserve
// to be buried under `reports`.
func NewStatsCommand() *cli.Command {
	return &cli.Command{
		Name:  "stats",
		Usage: "Task statistics (todo / in-progress / completed / total)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			project := c.String("project")
			endpoint := "/reports/stats"
			if project != "" {
				endpoint += "?project=" + url.QueryEscape(project)
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
