// Command to securely set and manage the Gemini API key for the CLI.
// Uses a config file in the user's home directory for storage.

package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

const geminiConfigFile = ".ramorie_gemini_key"

func NewGeminiKeyCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-gemini-key",
		Usage: "Set, update, or remove your Gemini API key securely",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "remove",
				Usage: "Remove the stored Gemini API key",
			},
		},
		Action: func(c *cli.Context) error {
			configPath, err := getGeminiConfigPath()
			if err != nil {
				return err
			}

			if c.Bool("remove") {
				if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove Gemini API key: %w", err)
				}
				fmt.Println("Gemini API key removed.")
				return nil
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

func getGeminiConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, geminiConfigFile), nil
}
