// Package help builds MCP-style help templates for the urfave/cli app.
//
// Commands carry their tier in cli.Command.Category (string), one of
// "essential", "common", "admin". urfave/cli v2's *cli.Command does not
// expose a generic Metadata map, so we reuse the Category field — which is
// already designed to group commands in help output. The app help template
// groups commands by tier; the per-command template renders the tier as a
// badge after the name.
package help

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// TierBadge returns the colored tier badge for a tier string.
func TierBadge(tier string) string {
	switch tier {
	case "essential":
		return "🔴 ESSENTIAL"
	case "common":
		return "🟡 COMMON"
	case "admin":
		return "🟢 ADMIN"
	default:
		return ""
	}
}

// SetTier marks a command with a tier label that the app help template groups
// by. The tier is stored in cmd.Category so it both groups in the default
// help renderer and is accessible to our custom templates.
//
// Returns the command for chaining:
//
//	app.Commands = []*cli.Command{help.SetTier(cmd, "essential"), ...}
func SetTier(cmd *cli.Command, tier string) *cli.Command {
	cmd.Category = tier
	return cmd
}

// AppHelpTemplate returns the urfave/cli AppHelpTemplate string with three
// tier sections. Commands without a tier (empty Category) are omitted from
// the tier sections — wire them into a separate section if needed.
func AppHelpTemplate() string {
	return `NAME:
   {{.Name}} — {{.Usage}}

USAGE:
   {{.HelpName}} [global options] command [command options]

VERSION:
   {{.Version}}

COMMANDS — 🔴 ESSENTIAL (daily)
{{range .Commands}}{{if eq .Category "essential"}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}
{{end}}{{end}}
COMMANDS — 🟡 COMMON (frequent)
{{range .Commands}}{{if eq .Category "common"}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}
{{end}}{{end}}
COMMANDS — 🟢 ADMIN (setup)
{{range .Commands}}{{if eq .Category "admin"}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}
{{end}}{{end}}
GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}
`
}

// CommandHelpTemplate returns the per-command help template that prints the
// tier badge after the name and renders subcommands and options cleanly.
func CommandHelpTemplate() string {
	return `NAME:
   {{.HelpName}} — {{.Usage}}{{if .Category}} ({{tierBadge .Category}}){{end}}

{{if .Description}}DESCRIPTION:
   {{.Description}}

{{end}}USAGE:
   {{.HelpName}}{{if .VisibleFlags}} [options]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{end}}

{{if .Subcommands}}SUBCOMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{end}}{{end}}

{{end}}{{if .VisibleFlags}}OPTIONS:
   {{range $idx, $opt := .VisibleFlags}}{{if $idx}}
   {{end}}{{$opt}}{{end}}

{{end}}`
}

// FuncMap exposes template helpers for the urfave/cli help renderer.
// Wire via cli.HelpPrinterCustom in main.go.
func FuncMap() map[string]any {
	return map[string]any{
		"tierBadge": TierBadge,
	}
}

// MustValidate is a startup-time sanity check that templates are non-empty.
func MustValidate() error {
	if AppHelpTemplate() == "" || CommandHelpTemplate() == "" {
		return fmt.Errorf("help templates must not be empty")
	}
	return nil
}
