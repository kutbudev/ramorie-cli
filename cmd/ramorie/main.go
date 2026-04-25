package main

import (
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/kutbudev/ramorie-cli/internal/cli/commands"
	"github.com/kutbudev/ramorie-cli/internal/cli/help"
	"github.com/urfave/cli/v2"
)

// Version is set during build with ldflags.
var Version = "6.0.0"

func main() {
	cli.AppHelpTemplate = help.AppHelpTemplate()
	cli.CommandHelpTemplate = help.CommandHelpTemplate()
	cli.SubcommandHelpTemplate = help.CommandHelpTemplate()

	cli.HelpPrinterCustom = func(out io.Writer, tmpl string, data interface{}, customFunc map[string]interface{}) {
		funcMap := template.FuncMap{
			"join":   strings.Join,
			"trim":   strings.TrimSpace,
			"indent": func(spaces int, v string) string { return strings.Repeat(" ", spaces) + v },
			"nindent": func(spaces int, v string) string {
				return "\n" + strings.Repeat(" ", spaces) + v
			},
			"subtract": func(a, b int) int { return a - b },
			"offset":   func(s string, n int) string { return strings.Repeat(" ", n) + s },
			"wrap":     func(input string, _ int) string { return input },
		}
		for k, v := range help.FuncMap() {
			funcMap[k] = v
		}
		for k, v := range customFunc {
			funcMap[k] = v
		}
		w := tabwriter.NewWriter(out, 1, 8, 2, ' ', 0)
		t := template.Must(template.New("help").Funcs(funcMap).Parse(tmpl))
		if err := t.Execute(w, data); err != nil {
			return
		}
		_ = w.Flush()
	}

	app := &cli.App{
		Name:    "ramorie",
		Usage:   "AI-powered task and memory management CLI",
		Version: Version,
		Commands: []*cli.Command{
			// 🔴 ESSENTIAL — daily.
			help.SetTier(commands.NewTaskCommand(), "essential"),
			help.SetTier(commands.NewMemoryCommand(), "essential"),
			help.SetTier(commands.NewProjectCommand(), "essential"),
			help.SetTier(commands.NewRememberCommand(), "essential"),
			help.SetTier(commands.NewFindCommand(), "essential"),
			help.SetTier(commands.NewUICommand(), "essential"),

			// 🟡 COMMON — frequent.
			help.SetTier(commands.NewKanbanCmd(), "common"),
			help.SetTier(commands.NewStatsCommand(), "common"),
			help.SetTier(commands.NewActivityCommand(), "common"),
			help.SetTier(commands.NewSubtaskCommand(), "common"),
			help.SetTier(commands.NewContextCommand(), "common"),

			// 🟢 ADMIN — setup.
			help.SetTier(commands.NewSetupCommand(), "admin"),
			help.SetTier(commands.NewUnlockCommand(), "admin"),
			help.SetTier(commands.NewLockCommand(), "admin"),
			help.SetTier(commands.NewConfigCommand(), "admin"),
			help.SetTier(commands.NewMcpCommand(), "admin"),
			help.SetTier(commands.NewHookCommand(), "admin"),
			help.SetTier(commands.NewOrganizationCommand(), "admin"),

			// Hidden — Claude Code hook helper.
			commands.NewFindRelatedCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
