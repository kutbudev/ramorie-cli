package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
			configSetGeminiKeyCmd(),
			configUnsetGeminiKeyCmd(),
		},
	}
}

// configSetGeminiKeyCmd securely stores a Gemini API key (prompts for value).
func configSetGeminiKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-gemini-key",
		Usage: "Securely store a Gemini API key (prompts for value)",
		Action: func(c *cli.Context) error {
			configPath, err := getGeminiConfigPath()
			if err != nil {
				return err
			}

			fmt.Print("Enter your Gemini API key: ")
			reader := bufio.NewReader(os.Stdin)
			key, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			key = strings.TrimSpace(key)
			if key == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			if err := os.WriteFile(configPath, []byte(key), 0600); err != nil {
				return fmt.Errorf("failed to save Gemini API key: %w", err)
			}
			fmt.Println("Gemini API key saved securely.")
			return nil
		},
	}
}

// configUnsetGeminiKeyCmd removes the stored Gemini API key.
func configUnsetGeminiKeyCmd() *cli.Command {
	return &cli.Command{
		Name:    "unset-gemini-key",
		Aliases: []string{"remove-gemini-key"},
		Usage:   "Remove the stored Gemini API key",
		Action: func(c *cli.Context) error {
			configPath, err := getGeminiConfigPath()
			if err != nil {
				return err
			}
			if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove Gemini API key: %w", err)
			}
			fmt.Println("Gemini API key removed.")
			return nil
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
