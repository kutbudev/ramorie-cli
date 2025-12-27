package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

const webURL = "https://ramorie.com"

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func NewSetupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Configure the CLI with user authentication",
		Subcommands: []*cli.Command{
			{
				Name:    "login",
				Aliases: []string{"l"},
				Usage:   "Login with your JosephsBrain account",
				Action: func(c *cli.Context) error {
					return handleUserLogin()
				},
			},
			{
				Name:  "api-key",
				Usage: "Manually set API key",
				Action: func(c *cli.Context) error {
					return handleManualAPIKey()
				},
			},
			{
				Name:  "status",
				Usage: "Check current authentication status",
				Action: func(c *cli.Context) error {
					return handleAuthStatus()
				},
			},
			{
				Name:  "logout",
				Usage: "Remove saved credentials",
				Action: func(c *cli.Context) error {
					return handleLogout()
				},
			},
		},
		Action: func(c *cli.Context) error {
			// Default action - interactive setup
			return handleInteractiveSetup()
		},
	}
}

func handleUserLogin() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("🔐 Ramorie Login")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read email: %w", err)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("email is required")
	}

	// Secure password input (hidden)
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after hidden input
	if err != nil {
		// Fallback to regular input if terminal not available
		password, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("could not read password: %w", err)
		}
		passwordBytes = []byte(strings.TrimSpace(password))
	}
	password := strings.TrimSpace(string(passwordBytes))

	if password == "" {
		return fmt.Errorf("password is required")
	}

	fmt.Println()
	fmt.Print("🔄 Logging in...")

	// Create API client and login user
	client := api.NewClient()
	apiKey, err := client.LoginUser(email, password)
	if err != nil {
		fmt.Println(" ❌")
		fmt.Println()
		fmt.Println("Login failed. Please check your credentials.")
		fmt.Println()
		fmt.Print("Don't have an account? Open browser to register? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			fmt.Println()
			fmt.Println("🌐 Opening browser...")
			browserErr := openBrowser(webURL + "/login")
			if browserErr != nil {
				fmt.Printf("Please visit: %s/login\n", webURL)
			}
		}
		return fmt.Errorf("login failed: %w", err)
	}

	// Save API key to config
	cfg := &config.Config{APIKey: apiKey}
	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println(" ✅")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Login successful!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("You can now use ramorie commands:")
	fmt.Println("  ramorie projects      - List your projects")
	fmt.Println("  ramorie list          - List your tasks")
	fmt.Println("  ramorie task \"...\"    - Create a new task")
	fmt.Println()
	return nil
}

func handleManualAPIKey() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("🔑 Manual API Key Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("You can find your API key in your account settings at:")
	fmt.Printf("  %s/settings\n", webURL)
	fmt.Println()

	fmt.Print("API Key: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = &config.Config{}
	}

	cfg.APIKey = apiKey

	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ API Key saved successfully!")
	return nil
}

func handleAuthStatus() error {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		fmt.Println()
		fmt.Println("❌ Not authenticated")
		fmt.Println()
		fmt.Println("To login, run:")
		fmt.Println("  ramorie setup login")
		fmt.Println()
		fmt.Println("Don't have an account? Register at:")
		fmt.Printf("  %s\n", webURL)
		fmt.Println()
		return nil
	}

	// Mask API key for display
	maskedKey := cfg.APIKey
	if len(maskedKey) > 12 {
		maskedKey = maskedKey[:8] + "..." + maskedKey[len(maskedKey)-4:]
	}

	fmt.Println()
	fmt.Println("✅ Authenticated")
	fmt.Println("━━━━━━━━━━━━━━━━")
	fmt.Printf("API Key: %s\n", maskedKey)
	fmt.Println()
	return nil
}

func handleLogout() error {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		fmt.Println("You are not logged in.")
		return nil
	}

	cfg.APIKey = ""
	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not clear credentials: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Logged out successfully")
	fmt.Println()
	return nil
}

func handleInteractiveSetup() error {
	reader := bufio.NewReader(os.Stdin)

	// Check if already authenticated
	cfg, err := config.LoadConfig()
	if err == nil && cfg.APIKey != "" {
		maskedKey := cfg.APIKey
		if len(maskedKey) > 12 {
			maskedKey = maskedKey[:8] + "..." + maskedKey[len(maskedKey)-4:]
		}
		fmt.Println()
		fmt.Println("✅ You are already authenticated")
		fmt.Printf("   API Key: %s\n", maskedKey)
		fmt.Println()
		fmt.Print("Do you want to login with a different account? (y/N): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			return nil
		}
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║          🧠 Ramorie CLI Setup             ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Welcome! To use the CLI, you need a Ramorie account.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [1] Login with existing account")
	fmt.Println("  [2] Enter API key manually")
	fmt.Println("  [3] Register a new account (opens browser)")
	fmt.Println("  [4] Exit")
	fmt.Println()
	fmt.Print("Choose an option (1-4): ")

	choice, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read choice: %w", err)
	}
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return handleUserLogin()
	case "2":
		return handleManualAPIKey()
	case "3":
		fmt.Println()
		fmt.Println("🌐 Opening browser for registration...")
		err := openBrowser(webURL + "/login")
		if err != nil {
			fmt.Println()
			fmt.Println("Could not open browser. Please visit:")
			fmt.Printf("  %s/login\n", webURL)
		} else {
			fmt.Println("✅ Browser opened!")
		}
		fmt.Println()
		fmt.Println("After registration, run 'ramorie setup login' to authenticate.")
		return nil
	case "4":
		fmt.Println("Setup cancelled.")
		return nil
	default:
		fmt.Println("Invalid option. Please run 'ramorie setup' again.")
		return nil
	}
}
