package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
	"github.com/urfave/cli/v2"
)

func NewSetupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Configure the CLI with user authentication",
		Subcommands: []*cli.Command{
			{
				Name:  "register",
				Usage: "Register a new user account",
				Action: func(c *cli.Context) error {
					return handleUserRegistration()
				},
			},
			{
				Name:  "login",
				Usage: "Login with existing user credentials",
				Action: func(c *cli.Context) error {
					return handleUserLogin()
				},
			},
			{
				Name:  "api-key",
				Usage: "Manually set API key (for existing users)",
				Action: func(c *cli.Context) error {
					return handleManualAPIKey()
				},
			},
		},
		Action: func(c *cli.Context) error {
			// Default action - show help
			return cli.ShowCommandHelp(c, "setup")
		},
	}
}

func handleUserRegistration() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your first name: ")
	firstName, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read first name: %w", err)
	}
	firstName = strings.TrimSpace(firstName)

	fmt.Print("Enter your last name: ")
	lastName, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read last name: %w", err)
	}
	lastName = strings.TrimSpace(lastName)

	fmt.Print("Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read email: %w", err)
	}
	email = strings.TrimSpace(email)

	fmt.Print("Enter your password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read password: %w", err)
	}
	password = strings.TrimSpace(password)

	// Create API client and register user
	client := api.NewClient()
	apiKey, err := client.RegisterUser(firstName, lastName, email, password)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Save API key to config
	cfg := &config.Config{APIKey: apiKey}
	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println("✅ User registered successfully!")
	fmt.Printf("✅ API Key saved: %s\n", apiKey)
	return nil
}

func handleUserLogin() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read email: %w", err)
	}
	email = strings.TrimSpace(email)

	fmt.Print("Enter your password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read password: %w", err)
	}
	password = strings.TrimSpace(password)

	// Create API client and login user
	client := api.NewClient()
	apiKey, err := client.LoginUser(email, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Save API key to config
	cfg := &config.Config{APIKey: apiKey}
	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println("✅ Login successful!")
	fmt.Printf("✅ API Key saved: %s\n", apiKey)
	return nil
}

func handleManualAPIKey() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your API Key: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	cfg, err := config.LoadConfig()
	if err != nil {
		// If config doesn't exist, a new one will be created.
		cfg = &config.Config{}
	}

	cfg.APIKey = apiKey

	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println("✅ Configuration saved successfully!")
	return nil
}