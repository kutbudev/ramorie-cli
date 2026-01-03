package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/mcp"
	"github.com/urfave/cli/v2"
)

func NewMcpCommand() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "MCP (Model Context Protocol) server management",
		Subcommands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start MCP server (stdio)",
				Action: func(c *cli.Context) error {
					client := api.NewClient()
					return mcp.ServeStdio(client)
				},
			},
			{
				Name:  "config",
				Usage: "Print MCP config examples for clients",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "client",
						Aliases: []string{"c"},
						Usage:   "target client (generic|codex)",
						Value:   "generic",
					},
				},
				Action: func(c *cli.Context) error {
					switch strings.ToLower(c.String("client")) {
					case "codex":
						printCodexConfig()
					default:
						printGenericConfig()
					}
					return nil
				},
			},
			{
				Name:  "tools",
				Usage: "List available MCP tools",
				Action: func(c *cli.Context) error {
					b, _ := json.MarshalIndent(mcp.ToolDefinitions(), "", "  ")
					os.Stdout.Write(b)
					os.Stdout.Write([]byte("\n"))
					return nil
				},
			},
		},
	}
}

func printGenericConfig() {
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"ramorie": map[string]interface{}{
				"command": "ramorie",
				"args":    []string{"mcp", "serve"},
			},
		},
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(b))
}

func printCodexConfig() {
	fmt.Println("# Add the following to ~/.codex/config.toml (merge with existing settings)")
	fmt.Println("[mcp_servers.ramorie]")
	fmt.Println("command = \"ramorie\"")
	fmt.Println("args = [\"mcp\", \"serve\"]")
	fmt.Println("enabled = true")
}
