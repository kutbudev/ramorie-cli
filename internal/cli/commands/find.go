package commands

import (
	"fmt"
	"strconv"
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
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Value: 30, Usage: "Max results (1-50)"},
			&cli.IntFlag{Name: "budget", Aliases: []string{"b"}, Value: 12000, Usage: "Token budget for response"},
			&cli.StringFlag{Name: "hyde", Value: "default", Usage: "HyDE expansion: on | off | default"},
			&cli.StringFlag{Name: "rerank", Value: "default", Usage: "LLM rerank: on | off | default"},
			&cli.StringFlag{Name: "intent", Value: "auto", Usage: "Intent: auto | how_to | why | recent | owner | generic"},
			&cli.IntFlag{Name: "entity-hops", Value: 0, Usage: "Entity graph hops (0-3)"},
			&cli.BoolFlag{Name: "include-superseded", Usage: "Include memories marked superseded"},
			&cli.BoolFlag{Name: "fast", Usage: "Force HyDE + rerank off (literal queries)"},
		},
		Action: func(c *cli.Context) error {
			parsedArgs, err := parseFindArgs(c.Args().Slice())
			if err != nil {
				return err
			}
			if len(parsedArgs.TermParts) == 0 {
				return fmt.Errorf("search term is required. Usage: ramorie find \"yarn rule\"")
			}
			term := strings.Join(parsedArgs.TermParts, " ")
			client := api.NewClient()

			projectArg := c.String("project")
			if parsedArgs.Project != nil {
				projectArg = *parsedArgs.Project
			}
			var projectID string
			if projectArg != "" {
				resolved, err := resolve.ResolveProject(projectArg, client)
				if err != nil {
					return err
				}
				projectID = resolved
			}

			types := append([]string{}, c.StringSlice("types")...)
			types = append(types, parsedArgs.Types...)
			tags := append([]string{}, c.StringSlice("tags")...)
			tags = append(tags, parsedArgs.Tags...)
			limit := c.Int("limit")
			if parsedArgs.Limit != nil {
				limit = *parsedArgs.Limit
			}
			budget := c.Int("budget")
			if parsedArgs.Budget != nil {
				budget = *parsedArgs.Budget
			}
			hyde := c.String("hyde")
			if parsedArgs.HyDE != nil {
				hyde = *parsedArgs.HyDE
			}
			rerank := c.String("rerank")
			if parsedArgs.Rerank != nil {
				rerank = *parsedArgs.Rerank
			}
			intent := c.String("intent")
			if parsedArgs.Intent != nil {
				intent = *parsedArgs.Intent
			}
			entityHops := c.Int("entity-hops")
			if parsedArgs.EntityHops != nil {
				entityHops = *parsedArgs.EntityHops
			}
			includeSuperseded := c.Bool("include-superseded")
			if parsedArgs.IncludeSuperseded != nil {
				includeSuperseded = *parsedArgs.IncludeSuperseded
			}
			fastMode := c.Bool("fast")
			if parsedArgs.FastMode != nil {
				fastMode = *parsedArgs.FastMode
			}

			opts := api.FindMemoriesOptions{
				Term:              term,
				Project:           projectID,
				Types:             types,
				Tags:              tags,
				Limit:             limit,
				BudgetTokens:      budget,
				HyDE:              hyde,
				Rerank:            rerank,
				Intent:            intent,
				EntityHops:        entityHops,
				IncludeSuperseded: includeSuperseded,
				FastMode:          fastMode,
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

type findParsedArgs struct {
	TermParts         []string
	Project           *string
	Types             []string
	Tags              []string
	Limit             *int
	Budget            *int
	HyDE              *string
	Rerank            *string
	Intent            *string
	EntityHops        *int
	IncludeSuperseded *bool
	FastMode          *bool
}

func parseFindArgs(args []string) (findParsedArgs, error) {
	var out findParsedArgs
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			out.TermParts = append(out.TermParts, args[i+1:]...)
			break
		}
		if token == "-" || !strings.HasPrefix(token, "-") {
			out.TermParts = append(out.TermParts, token)
			continue
		}

		name, value, hasValue := splitFindFlag(token)
		switch name {
		case "project", "p":
			v, err := consumeFindFlagValue(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Project = &v
		case "types", "t":
			v, err := consumeFindFlagValue(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Types = append(out.Types, v)
		case "tags":
			v, err := consumeFindFlagValue(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Tags = append(out.Tags, v)
		case "limit", "n":
			v, err := consumeFindIntFlag(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Limit = &v
		case "budget", "b":
			v, err := consumeFindIntFlag(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Budget = &v
		case "hyde":
			v, err := consumeFindFlagValue(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.HyDE = &v
		case "rerank":
			v, err := consumeFindFlagValue(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Rerank = &v
		case "intent":
			v, err := consumeFindFlagValue(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.Intent = &v
		case "entity-hops":
			v, err := consumeFindIntFlag(name, value, hasValue, args, &i)
			if err != nil {
				return out, err
			}
			out.EntityHops = &v
		case "include-superseded":
			v, err := parseFindBoolFlag(name, value, hasValue)
			if err != nil {
				return out, err
			}
			out.IncludeSuperseded = &v
		case "fast":
			v, err := parseFindBoolFlag(name, value, hasValue)
			if err != nil {
				return out, err
			}
			out.FastMode = &v
		default:
			return out, fmt.Errorf("unknown find flag %q; put literal flag-like search text after --", token)
		}
	}
	return out, nil
}

func splitFindFlag(token string) (name, value string, hasValue bool) {
	name = strings.TrimLeft(token, "-")
	if before, after, ok := strings.Cut(name, "="); ok {
		return before, after, true
	}
	return name, "", false
}

func consumeFindFlagValue(name, value string, hasValue bool, args []string, index *int) (string, error) {
	if hasValue {
		if value == "" {
			return "", fmt.Errorf("flag --%s requires a value", name)
		}
		return value, nil
	}
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "-") {
		return "", fmt.Errorf("flag --%s requires a value", name)
	}
	*index = *index + 1
	return args[*index], nil
}

func consumeFindIntFlag(name, value string, hasValue bool, args []string, index *int) (int, error) {
	raw, err := consumeFindFlagValue(name, value, hasValue, args, index)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("flag --%s expects an integer, got %q", name, raw)
	}
	return parsed, nil
}

func parseFindBoolFlag(name, value string, hasValue bool) (bool, error) {
	if !hasValue {
		return true, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("flag --%s expects a boolean, got %q", name, value)
	}
	return parsed, nil
}
