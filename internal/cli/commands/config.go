package commands

import (
	"fmt"

	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/urfave/cli/v2"
)

// NewConfigCommand creates the 'config' command.
func NewConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "View or edit the CLI configuration",
		Subcommands: []*cli.Command{
			configShowCmd(),
			configSetApiKeyCmd(),
		},
	}
}

// configShowCmd displays the current configuration.
func configShowCmd() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Show current configuration",
		Action: func(c *cli.Context) error {
			cliCfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("could not load CLI config: %w", err)
			}

			fmt.Println("--- CLI Configuration ---")
			if cliCfg.APIKey != "" {
				fmt.Printf("API Key:     %s****\n", cliCfg.APIKey[:4])
			} else {
				fmt.Println("API Key:     Not set")
			}
			fmt.Println("-----------------------")

			return nil
		},
	}
}

// configSetApiKeyCmd sets the API key manually.
func configSetApiKeyCmd() *cli.Command {
	return &cli.Command{
		Name:      "set-apikey",
		Usage:     "Set your API key manually",
		ArgsUsage: "[api-key]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("API key is required")
			}
			apiKey := c.Args().First()

			cfg, err := config.LoadConfig()
			if err != nil {
				cfg = &config.Config{}
			}

			cfg.APIKey = apiKey
			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("could not save config: %w", err)
			}

			fmt.Println("✅ API Key saved successfully.")
			return nil
		},
	}
}
