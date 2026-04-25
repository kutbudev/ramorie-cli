package commands

import (
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/cli/resolve"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/urfave/cli/v2"
)

// NewFindCommand returns the top-level `find` command — hybrid memory search
// that mirrors the MCP find tool. Calls POST /v1/memory/find under the hood.
func NewFindCommand() *cli.Command {
	return &cli.Command{
		Name:      "find",
		Usage:     "Hybrid memory search (HyDE + rerank + entity graph)",
		ArgsUsage: "<term>",
		Description: `Hybrid retrieval pipeline matching the MCP find tool.

Stages: HyDE query expansion → pgvector + FTS hybrid scan → entity graph
boost → propositional boost → intent routing → Gemini rerank.

If --project is omitted, the backend auto-scopes via the X-Project-Hint
header (cwd-derived project name).`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Filter by project (name, short id, or full UUID)"},
			&cli.StringSliceFlag{Name: "types", Aliases: []string{"t"}, Usage: "Filter by memory types (e.g. -t decision -t pattern)"},
			&cli.StringSliceFlag{Name: "tags", Usage: "Filter by tags"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Value: 5, Usage: "Max results (1-50)"},
			&cli.IntFlag{Name: "budget", Aliases: []string{"b"}, Value: 2000, Usage: "Token budget for response"},
			&cli.StringFlag{Name: "hyde", Value: "default", Usage: "HyDE expansion: on | off | default"},
			&cli.StringFlag{Name: "rerank", Value: "default", Usage: "LLM rerank: on | off | default"},
			&cli.StringFlag{Name: "intent", Value: "auto", Usage: "Intent: auto | how_to | why | recent | owner | generic"},
			&cli.IntFlag{Name: "entity-hops", Value: 0, Usage: "Entity graph hops (0-3)"},
			&cli.BoolFlag{Name: "include-superseded", Usage: "Include memories marked superseded"},
			&cli.BoolFlag{Name: "fast", Usage: "Force HyDE + rerank off (literal queries)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("search term is required. Usage: ramorie find \"yarn rule\"")
			}
			term := strings.Join(c.Args().Slice(), " ")
			client := api.NewClient()

			projectArg := c.String("project")
			var projectID string
			if projectArg != "" {
				resolved, err := resolve.ResolveProject(projectArg, client)
				if err != nil {
					return err
				}
				projectID = resolved
			}

			opts := api.FindMemoriesOptions{
				Term:              term,
				Project:           projectID,
				Types:             c.StringSlice("types"),
				Tags:              c.StringSlice("tags"),
				Limit:             c.Int("limit"),
				BudgetTokens:      c.Int("budget"),
				HyDE:              c.String("hyde"),
				Rerank:            c.String("rerank"),
				Intent:            c.String("intent"),
				EntityHops:        c.Int("entity-hops"),
				IncludeSuperseded: c.Bool("include-superseded"),
				FastMode:          c.Bool("fast"),
				IncludeDecisions:  true,
			}

			resp, err := client.FindMemories(opts)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}
			if resp == nil || len(resp.Items) == 0 {
				fmt.Println(display.Dim.Render("  no results — try a different term, or drop --fast"))
				return nil
			}

			titlePart := fmt.Sprintf("🔎 %d hit", len(resp.Items))
			if len(resp.Items) != 1 {
				titlePart += "s"
			}
			subtitle := fmt.Sprintf("intent: %s · ranking: %s · %dms",
				resp.Meta.Intent, resp.Meta.RankingMode, resp.Meta.LatencyMs)
			fmt.Println(display.Header(titlePart, subtitle))
			fmt.Println()

			titleWidth := display.TerminalWidth() - 50
			if titleWidth < 30 {
				titleWidth = 30
			}

			for _, it := range resp.Items {
				title := display.Truncate(display.SingleLine(it.Title), titleWidth)
				score := fmt.Sprintf("%.2f", it.Score)
				shortID := it.ID
				if len(shortID) > 8 {
					shortID = shortID[:8]
				}
				fmt.Printf(" %s %s  %s%s%s\n",
					display.TypeBadge(it.Type),
					display.Dim.Render(shortID),
					title,
					display.Sep()+display.Dim.Render(it.Project),
					display.Sep()+display.Dim.Render("score "+score),
				)
			}
			return nil
		},
	}
}
