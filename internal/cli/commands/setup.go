package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
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
				Usage:   "Login with your Ramorie account",
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
			{
				Name:  "unlock",
				Usage: "Unlock your encrypted vault with master password",
				Action: func(c *cli.Context) error {
					return handleVaultUnlock()
				},
			},
			{
				Name:  "lock",
				Usage: "Lock your encrypted vault (clear keys from memory)",
				Action: func(c *cli.Context) error {
					return handleVaultLock()
				},
			},
			{
				Name:  "vault-status",
				Usage: "Check encryption vault status",
				Action: func(c *cli.Context) error {
					return handleVaultStatus()
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

		// Use enhanced error parsing
		errorMsg := apierrors.ParseAPIError(err)
		fmt.Println(errorMsg)
		fmt.Println()

		// Offer alternatives
		fmt.Println("What would you like to do?")
		fmt.Println("  [1] Try again")
		fmt.Println("  [2] Enter API key instead (from Settings page)")
		fmt.Println("  [3] Register a new account (opens browser)")
		fmt.Println("  [4] Exit")
		fmt.Println()
		fmt.Print("Choose (1-4): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		switch answer {
		case "1":
			return handleUserLogin()
		case "2":
			return handleManualAPIKey()
		case "3":
			fmt.Println()
			fmt.Println("🌐 Opening browser...")
			browserErr := openBrowser(webURL + "/login")
			if browserErr != nil {
				fmt.Printf("Please visit: %s/login\n", webURL)
			}
			fmt.Println()
			fmt.Println("After registration, run 'ramorie setup' to authenticate.")
			return nil
		default:
			return nil
		}
	}

	// Save API key to config
	cfg := &config.Config{APIKey: apiKey}
	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println(" ✅")

	// Fetch encryption config with the new API key
	client.APIKey = apiKey
	encConfig, err := client.GetEncryptionConfig()
	if err == nil && encConfig != nil && encConfig.EncryptionEnabled {
		// Save encryption config
		cfg.EncryptionEnabled = encConfig.EncryptionEnabled
		cfg.EncryptedSymmetricKey = encConfig.EncryptedSymmetricKey
		cfg.KeyNonce = encConfig.KeyNonce
		cfg.Salt = encConfig.Salt
		cfg.KDFIterations = encConfig.KDFIterations
		cfg.KDFAlgorithm = encConfig.KDFAlgorithm
		_ = config.SaveConfig(cfg) // Best effort

		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("✅ Login successful!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("🔐 Encryption is enabled for this account.")
		fmt.Println("   Run 'ramorie vault unlock' to unlock your vault.")
		fmt.Println()
	} else {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("✅ Login successful!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

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
		fmt.Println("  ramorie setup")
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
		fmt.Println("After registration, run 'ramorie setup' to authenticate.")
		return nil
	case "4":
		fmt.Println("Setup cancelled.")
		return nil
	default:
		fmt.Println("Invalid option. Please run 'ramorie setup' again.")
		return nil
	}
}

// handleVaultUnlock prompts for master password and unlocks the vault
func handleVaultUnlock() error {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		fmt.Println()
		fmt.Println("❌ Not authenticated")
		fmt.Println("   Please run 'ramorie setup' first.")
		return nil
	}

	if crypto.IsVaultUnlocked() {
		fmt.Println()
		fmt.Println("✅ Vault is already unlocked")
		return nil
	}

	// Fetch encryption config from API (always fresh, not from local cache)
	fmt.Print("🔄 Checking encryption status...")
	client := api.NewClient()
	client.APIKey = cfg.APIKey
	encConfig, err := client.GetEncryptionConfig()
	if err != nil {
		fmt.Println(" ❌")
		fmt.Println()
		fmt.Printf("Failed to check encryption status: %v\n", err)
		return nil
	}

	if !encConfig.EncryptionEnabled {
		fmt.Println(" ℹ️")
		fmt.Println()
		fmt.Println("ℹ️  Encryption is not enabled for this account.")
		fmt.Println("   Enable encryption in the web dashboard at:")
		fmt.Printf("   %s/settings/security\n", webURL)
		return nil
	}
	fmt.Println(" ✅")

	// Update local config with encryption data for future use
	cfg.EncryptionEnabled = encConfig.EncryptionEnabled
	cfg.EncryptedSymmetricKey = encConfig.EncryptedSymmetricKey
	cfg.Salt = encConfig.Salt
	cfg.KDFIterations = encConfig.KDFIterations
	cfg.KDFAlgorithm = encConfig.KDFAlgorithm
	_ = config.SaveConfig(cfg) // Best effort cache update

	fmt.Println()
	fmt.Println("🔐 Vault Unlock")
	fmt.Println("━━━━━━━━━━━━━━━")
	fmt.Println()

	// Secure password input
	fmt.Print("Master Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("could not read password: %w", err)
	}
	masterPassword := string(passwordBytes)

	if masterPassword == "" {
		return fmt.Errorf("master password is required")
	}

	fmt.Print("🔄 Unlocking vault...")

	// Create vault config from API encryption data
	vaultConfig := &crypto.VaultConfig{
		EncryptionEnabled:     encConfig.EncryptionEnabled,
		EncryptedSymmetricKey: encConfig.EncryptedSymmetricKey,
		KeyNonce:              encConfig.KeyNonce,
		Salt:                  encConfig.Salt,
		KDFIterations:         encConfig.KDFIterations,
		KDFAlgorithm:          encConfig.KDFAlgorithm,
	}

	err = crypto.UnlockVault(masterPassword, vaultConfig)
	if err != nil {
		fmt.Println(" ❌")
		fmt.Println()
		fmt.Println("Invalid master password. Please try again.")
		return nil
	}

	fmt.Println(" ✅")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔓 Vault unlocked successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Your memories and tasks will now be encrypted/decrypted automatically.")
	fmt.Println("Run 'ramorie vault lock' to lock your vault when done.")
	fmt.Println()
	return nil
}

// handleVaultLock clears keys from memory
func handleVaultLock() error {
	if !crypto.IsVaultUnlocked() {
		fmt.Println()
		fmt.Println("ℹ️  Vault is already locked")
		return nil
	}

	crypto.LockVault()

	fmt.Println()
	fmt.Println("🔒 Vault locked successfully!")
	fmt.Println("   Encryption keys have been cleared from memory.")
	fmt.Println()
	return nil
}

// handleVaultStatus shows the current vault status
func handleVaultStatus() error {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		fmt.Println()
		fmt.Println("❌ Not authenticated")
		return nil
	}

	fmt.Println()
	fmt.Println("🔐 Vault Status")
	fmt.Println("━━━━━━━━━━━━━━━")
	fmt.Println()

	if !cfg.EncryptionEnabled {
		fmt.Println("Encryption: Disabled")
		fmt.Println()
		fmt.Println("Enable encryption in the web dashboard at:")
		fmt.Printf("  %s/settings/security\n", webURL)
		return nil
	}

	fmt.Println("Encryption: ✅ Enabled")

	if crypto.IsVaultUnlocked() {
		fmt.Println("Vault:      🔓 Unlocked")
		fmt.Println()
		fmt.Println("Your data is being encrypted/decrypted automatically.")
	} else {
		fmt.Println("Vault:      🔒 Locked")
		fmt.Println()
		fmt.Println("Run 'ramorie vault unlock' to unlock your vault.")
	}

	fmt.Println()
	return nil
}

// NewVaultCommand creates a top-level vault command as an alias for setup vault commands
// This allows users to run 'ramorie vault unlock' instead of 'ramorie vault unlock'
func NewVaultCommand() *cli.Command {
	return &cli.Command{
		Name:  "vault",
		Usage: "Manage your encrypted vault (alias for setup vault commands)",
		Subcommands: []*cli.Command{
			{
				Name:  "unlock",
				Usage: "Unlock your encrypted vault with master password",
				Action: func(c *cli.Context) error {
					return handleVaultUnlock()
				},
			},
			{
				Name:  "lock",
				Usage: "Lock your encrypted vault (clear keys from memory)",
				Action: func(c *cli.Context) error {
					return handleVaultLock()
				},
			},
			{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Check encryption vault status",
				Action: func(c *cli.Context) error {
					return handleVaultStatus()
				},
			},
		},
		Action: func(c *cli.Context) error {
			// Default action - show vault status
			return handleVaultStatus()
		},
	}
}
