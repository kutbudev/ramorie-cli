package commands

import (
	"encoding/json"
	"fmt"
	"os"

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
				Action: func(c *cli.Context) error {
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
