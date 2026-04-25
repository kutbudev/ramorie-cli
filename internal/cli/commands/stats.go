package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// taskStats matches the backend's ProjectStats JSON shape.
type taskStats struct {
	Total      int64 `json:"total"`
	Todo       int64 `json:"todo"`
	InProgress int64 `json:"in_progress"`
	Completed  int64 `json:"completed"`
}

// NewStatsCommand wraps the stats endpoint that reports.go's `stats` subcommand
// already exposes. Promoted to top-level for v6 — daily verb, doesn't deserve
// to be buried under `reports`.
func NewStatsCommand() *cli.Command {
	return &cli.Command{
		Name:  "stats",
		Usage: "Task statistics (todo / in-progress / completed / total)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID"},
			&cli.BoolFlag{Name: "json", Usage: "Output raw JSON (always on when piped)"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			project := c.String("project")
			isTTY := term.IsTerminal(int(os.Stdout.Fd()))
			wantJSON := c.Bool("json") || !isTTY

			endpoint := "/reports/stats"
			if project != "" {
				endpoint += "?project=" + url.QueryEscape(project)
			}

			b, err := client.Request("GET", endpoint, nil)
			if err != nil {
				return err
			}

			if wantJSON {
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

			var s taskStats
			if err := json.Unmarshal(b, &s); err != nil {
				os.Stdout.Write(b)
				os.Stdout.Write([]byte("\n"))
				return nil
			}

			subtitle := ""
			if project != "" {
				subtitle = "project: " + project
			}
			fmt.Println(display.Header("📊 task stats", subtitle))
			fmt.Println()

			cols := []display.Column{
				{Title: "METRIC", Min: 14, Weight: 0},
				{Title: "VALUE", Min: 8, Weight: 1},
			}
			rows := [][]string{
				{display.Dim.Render("total"), fmt.Sprintf("%d", s.Total)},
				{display.Dim.Render("todo"), fmt.Sprintf("%d", s.Todo)},
				{display.Dim.Render("in_progress"), fmt.Sprintf("%d", s.InProgress)},
				{display.Dim.Render("completed"), fmt.Sprintf("%d", s.Completed)},
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
			return nil
		},
	}
}
