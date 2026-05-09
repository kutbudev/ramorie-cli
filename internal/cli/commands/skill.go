package commands

import (
	"encoding/json"
	"fmt"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewSkillCommand creates the 'skill' command group. PR6 (v6.8.0)
// ships with a single action — `use` — that mirrors `pack use`: pipe
// the rendered Claude Code-format markdown into stdout so agents,
// Claude Code's `Read`-style tools, or `pbcopy` can grab it without
// further parsing.
//
//	ramorie skill use <id-or-name>          # markdown body to stdout
//	ramorie skill use <id-or-name> --json   # full response JSON
//
// PR7 will extend this group with sync (`skill push`, `skill pull`)
// and PR8 with AI generation (`skill generate`). Subcommand list
// stays minimal until those land so help output isn't cluttered with
// unimplemented verbs.
func NewSkillCommand() *cli.Command {
	return &cli.Command{
		Name:    "skill",
		Aliases: []string{"skills"},
		Usage:   "Render and manage procedural skills",
		Subcommands: []*cli.Command{
			skillUseCmd(),
		},
	}
}

// skillUseCmd renders a skill via GET /memories/{id}/skill-render and
// prints the body to stdout (pipe-friendly). With --json, prints the
// full envelope (skill + body + source + _meta) instead.
//
//	ramorie skill use deploy-prod
//	ramorie skill use 7f3c1b3e-... --json
func skillUseCmd() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Aliases:   []string{"render", "load"},
		Usage:     "Render a skill (Claude Code-format markdown) to stdout",
		ArgsUsage: "[skill-id-or-name]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Output full response JSON instead of bare markdown body"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("skill id or name required")
			}
			ident := c.Args().First()

			client := api.NewClient()
			resp, err := client.LoadSkill(ident)
			if err != nil {
				return fmt.Errorf("load skill: %w", err)
			}

			// Encrypted body guard — server returns the cipher untouched
			// (zero-knowledge). Don't print it as if it were markdown;
			// fail loud with a non-zero exit so scripts piping into a
			// file or `pbcopy` don't silently capture ciphertext. --json
			// still works (full envelope inspection is useful even when
			// encrypted).
			if encVal, ok := resp.Meta["encrypted"]; ok && !c.Bool("json") {
				if enc, _ := encVal.(bool); enc {
					fmt.Fprintln(c.App.ErrWriter, "⚠ skill body is encrypted — vault unlock required (`ramorie unlock` then retry)")
					return cli.Exit("encrypted skill body cannot be rendered", 2)
				}
			}

			if c.Bool("json") {
				out, err := json.MarshalIndent(resp, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal skill response: %w", err)
				}
				fmt.Println(string(out))
				return nil
			}

			// Bare body to stdout — pipeable into pbcopy, an agent, or
			// `>` redirect for `.claude/skills/<name>/SKILL.md`. Stderr
			// gets a one-line byline so the human sees what just rendered
			// without polluting the markdown.
			fmt.Print(resp.Body)
			fmt.Println()
			fmt.Fprintf(c.App.ErrWriter, "─── skill %q v%s, %d steps ───\n",
				resp.Skill.Name, resp.Skill.Version, resp.Skill.StepsCount)
			return nil
		},
	}
}
