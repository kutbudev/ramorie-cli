package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// activityItem matches the backend's ReportHistoryItem JSON shape.
type activityItem struct {
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	ProjectID  *string   `json:"project_id,omitempty"`
	Summary    string    `json:"summary"`
	Timestamp  time.Time `json:"timestamp"`
}

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
			&cli.BoolFlag{Name: "json", Usage: "Output raw JSON (always on when piped)"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			project := c.String("project")
			isTTY := term.IsTerminal(int(os.Stdout.Fd()))
			wantJSON := c.Bool("json") || !isTTY

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

				// Burndown stays JSON — table layout is poor for time-series data.
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

			// TTY path: render as a responsive table.
			var items []activityItem
			if err := json.Unmarshal(b, &items); err != nil {
				// Fallback: dump raw output rather than swallowing it.
				os.Stdout.Write(b)
				os.Stdout.Write([]byte("\n"))
				return nil
			}

			if len(items) == 0 {
				fmt.Println(display.Dim.Render("  no activity in the last " + fmt.Sprintf("%d days", days)))
				return nil
			}

			countPart := fmt.Sprintf("📜 %d event", len(items))
			if len(items) != 1 {
				countPart += "s"
			}
			subtitle := fmt.Sprintf("last %dd", days)
			if project != "" {
				subtitle += " · project: " + project
			}
			fmt.Println(display.Header(countPart, subtitle))
			fmt.Println()

			cols := []display.Column{
				{Title: "TIME", Min: 16, Weight: 0},
				{Title: "TYPE", Min: 10, Weight: 0},
				{Title: "PROJECT", Min: 10, Weight: 1},
				{Title: "SUMMARY", Min: 30, Weight: 5},
				{Title: "ENTITY", Min: 8, Weight: 0},
			}
			rows := make([][]string, 0, len(items))
			for _, it := range items {
				proj := ""
				if it.ProjectID != nil && len(*it.ProjectID) >= 8 {
					proj = (*it.ProjectID)[:8]
				}
				entity := it.EntityID
				if len(entity) > 8 {
					entity = entity[:8]
				}
				rows = append(rows, []string{
					display.Dim.Render(display.Relative(it.Timestamp)),
					display.Dim.Render(it.EntityType),
					display.Dim.Render(proj),
					display.SingleLine(it.Summary),
					display.Dim.Render(entity),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
			return nil
		},
	}
}
