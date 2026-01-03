package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

func NewReportsCommand() *cli.Command {
	return &cli.Command{
		Name:    "reports",
		Aliases: []string{"report"},
		Usage:   "Reports",
		Subcommands: []*cli.Command{
			reportsStatsCmd(),
			reportsHistoryCmd(),
			reportsBurndownCmd(),
			reportsSummaryCmd(),
		},
	}
}

func reportsStatsCmd() *cli.Command {
	return &cli.Command{
		Name:  "stats",
		Usage: "Get task stats",
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

func reportsHistoryCmd() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "Get activity history",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "days", Aliases: []string{"d"}, Usage: "How many days", Value: 7},
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "Max items", Value: 15},
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			days := c.Int("days")
			limit := c.Int("limit")
			project := c.String("project")

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

func reportsBurndownCmd() *cli.Command {
	return &cli.Command{
		Name:  "burndown",
		Usage: "Get burndown report",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "days", Aliases: []string{"d"}, Usage: "How many days", Value: 30},
			&cli.StringFlag{Name: "interval", Aliases: []string{"i"}, Usage: "daily or weekly", Value: "daily"},
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID"},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			days := c.Int("days")
			interval := c.String("interval")
			project := c.String("project")

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
		},
	}
}

func reportsSummaryCmd() *cli.Command {
	return &cli.Command{
		Name:      "summary",
		Usage:     "Generate summary",
		ArgsUsage: "[n]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project name or ID"},
			&cli.IntFlag{Name: "n", Usage: "How many tasks", Value: 10},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			project := c.String("project")

			n := c.Int("n")
			if c.NArg() > 0 {
				if v, err := parseIntArg(c.Args().First()); err == nil {
					n = v
				}
			}
			if n <= 0 {
				n = 10
			}

			req := map[string]interface{}{
				"project": project,
				"n":       n,
			}

			b, err := client.Request("POST", "/reports/summary", req)
			if err != nil {
				return err
			}

			var out struct {
				Summary string `json:"summary"`
			}
			if err := json.Unmarshal(b, &out); err != nil {
				os.Stdout.Write(b)
				os.Stdout.Write([]byte("\n"))
				return nil
			}

			fmt.Println(out.Summary)
			return nil
		},
	}
}

func parseIntArg(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}
