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
	"github.com/kutbudev/ramorie-cli/internal/hooks"
	"github.com/kutbudev/ramorie-cli/internal/mcpinstall"
	"github.com/kutbudev/ramorie-cli/internal/protocol"
	"github.com/kutbudev/ramorie-cli/internal/rules"
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
	// v7.1.0 alias — expose `setup-hooks` machinery as `setup hooks` too,
	// so both invocation styles work. The underlying subcommand actions
	// are re-used verbatim; we just re-parent the subtree.
	hooksAlias := NewSetupHooksCommand()
	hooksAlias.Name = "hooks"
	hooksAlias.Usage = "[alias for `ramorie setup-hooks`] Install/uninstall protocol hooks + rules"

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
			hooksAlias,
			{
				Name:  "vault",
				Usage: "Encrypted vault operations (unlock | lock | status)",
				Subcommands: []*cli.Command{
					{
						Name:   "unlock",
						Usage:  "Unlock vault with master password",
						Action: func(c *cli.Context) error { return handleVaultUnlock() },
					},
					{
						Name:   "lock",
						Usage:  "Lock vault (clear keys from memory)",
						Action: func(c *cli.Context) error { return handleVaultLock() },
					},
					{
						Name:    "status",
						Aliases: []string{"s"},
						Usage:   "Show vault status",
						Action:  func(c *cli.Context) error { return handleVaultStatus() },
					},
				},
			},
		},
		Action: func(c *cli.Context) error {
			// Default action — full one-command setup orchestrator. Falls
			// back to the legacy interactive picker when --legacy is set.
			if c.Bool("legacy") {
				return handleInteractiveSetup()
			}
			return runFullSetup(c)
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "legacy",
				Usage: "Use the legacy interactive picker (login/api-key/register)",
			},
			&cli.BoolFlag{
				Name:  "skip-mcp",
				Usage: "Skip MCP server install step",
			},
			&cli.BoolFlag{
				Name:  "skip-hooks",
				Usage: "Skip hook installation (Claude Code, Codex)",
			},
			&cli.BoolFlag{
				Name:  "skip-rules",
				Usage: "Skip rules-file installation (Cursor, Windsurf)",
			},
		},
	}
}

// runFullSetup performs the complete Ramorie onboarding in 6 steps:
//
//  1. Auth — login or skip if already authenticated.
//  2. MCP install — every detected client (Claude Code, Claude Desktop,
//     Cursor, Windsurf, VS Code, Zed).
//  3. Hook install — Claude Code + Codex (shell-command hooks).
//  4. Rules-file install — Cursor + Windsurf (managed markdown blocks).
//  5. Vault unlock — prompt if encryption is enabled but vault is locked.
//  6. Diagnostic — quick health check summary.
//
// Idempotent: re-running emits "already configured" markers for completed
// steps. Per-step failures degrade gracefully — a warning is printed and the
// next step still runs.
func runFullSetup(c *cli.Context) error {
	fmt.Println()
	fmt.Println("🚀 Ramorie Setup — One-command full install")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 1. Auth
	if isAuthConfigured() {
		fmt.Println("✓ Step 1/6 — Already authenticated")
	} else {
		fmt.Println("🔐 Step 1/6 — Authentication")
		if err := handleInteractiveSetup(); err != nil {
			return fmt.Errorf("auth step failed: %w", err)
		}
		// Re-check; if user bailed out of the interactive picker we should
		// still try to continue with the other steps that don't need auth
		// (hooks/rules), but warn that the MCP install may fail.
		if !isAuthConfigured() {
			warn("authentication not completed — continuing with best-effort install")
		}
	}

	// 2. MCP install — requires auth (the installer writes per-user MCP entries
	// that reference the saved API key). If auth still isn't configured at this
	// point (user bailed in Step 1), skip with a clear note instead of letting
	// the installer emit a less-obvious warning later.
	switch {
	case c.Bool("skip-mcp"):
		fmt.Println("\n✓ Step 2/6 — Skipped (--skip-mcp)")
	case !isAuthConfigured():
		fmt.Println()
		fmt.Println("📦 Step 2/6 — MCP server installation")
		warn("MCP install requires auth — skipping (use --skip-mcp to silence)")
	default:
		fmt.Println()
		fmt.Println("📦 Step 2/6 — MCP server installation")
		if err := installMCPForAllDetected(); err != nil {
			warn(fmt.Sprintf("mcp install: %v", err))
		}
	}

	// 3. Hook install (hook-capable clients)
	if !c.Bool("skip-hooks") {
		fmt.Println()
		fmt.Println("🪝 Step 3/6 — Hook installation (Claude Code, Codex)")
		if err := installHooksForAllDetected(false); err != nil {
			warn(fmt.Sprintf("hook install: %v", err))
		}
	} else {
		fmt.Println("\n✓ Step 3/6 — Skipped (--skip-hooks)")
	}

	// 4. Rules install (rules-only clients)
	if !c.Bool("skip-rules") {
		fmt.Println()
		fmt.Println("📜 Step 4/6 — Rules-file installation (Cursor, Windsurf)")
		if err := installRulesForAllDetected(false); err != nil {
			warn(fmt.Sprintf("rules install: %v", err))
		}
	} else {
		fmt.Println("\n✓ Step 4/6 — Skipped (--skip-rules)")
	}

	// 5. Vault unlock
	fmt.Println()
	fmt.Println("🔓 Step 5/6 — Vault status")
	if err := promptVaultUnlockIfNeeded(); err != nil {
		warn(fmt.Sprintf("vault step: %v", err))
	}

	// 6. Diagnostic — propagate failure as exit 1 so CI / shell-conditional
	// callers can detect a broken install. The earlier steps already print
	// per-surface warnings; we just need the final verdict here.
	fmt.Println()
	fmt.Println("🩺 Step 6/6 — Health check")
	doctorFailed := runQuickDoctor()

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if doctorFailed {
		fmt.Println("⚠ Setup completed with errors. Run `ramorie doctor` for details.")
		fmt.Println()
		return cli.Exit("setup completed with errors", 1)
	}

	fmt.Println("✨ Ramorie ready. Restart your MCP clients to pick up new config.")
	fmt.Println()
	fmt.Println("Useful next commands:")
	fmt.Println("  ramorie doctor          — re-run the health check")
	fmt.Println("  ramorie setup hooks     — manage protocol hooks")
	fmt.Println("  ramorie project list    — list your projects")
	fmt.Println()
	return nil
}

// isAuthConfigured returns true when the config file already holds a non-
// empty API key. Used by runFullSetup to short-circuit the login step.
func isAuthConfigured() bool {
	cfg, err := config.LoadConfig()
	if err != nil {
		return false
	}
	return cfg.APIKey != ""
}

// warn prints a non-fatal warning. We use a leading bullet rather than ⚠ so
// the output is grep-friendly and clearly distinct from ✓/✗ status markers.
func warn(msg string) {
	fmt.Printf("  ⚠ %s\n", msg)
}

// installMCPForAllDetected runs the standard MCP installer in TUI mode if
// stdout looks interactive; otherwise it falls back to a quiet, opinionated
// non-interactive install for every detected client (user-scope where
// supported, project-scope as a secondary).
//
// We delegate to mcpinstall.Run for the rich path; here in the orchestrator
// we just print a one-liner so the operator knows what happened.
func installMCPForAllDetected() error {
	// The interactive TUI is the safest UX — it surfaces a preview before
	// writing anything. runFullSetup callers expect a guided experience.
	return mcpinstall.Run(mcpinstall.BinaryPath(), []string{"mcp", "serve"})
}

// installHooksForAllDetected installs protocol hooks into every hook-capable
// client we can detect. dryRun = true logs target paths but writes nothing.
func installHooksForAllDetected(dryRun bool) error {
	installers := []hooks.Installer{
		hooks.NewClaudeCodeInstaller(),
		hooks.NewCodexInstaller(),
	}
	any := false
	for _, inst := range installers {
		if !inst.Detect() {
			fmt.Printf("  · %s not detected — skipping\n", inst.Name())
			continue
		}
		any = true
		if dryRun {
			fmt.Printf("  · %s — would write to %s\n", inst.Name(), inst.SettingsPath())
			continue
		}
		if err := inst.Install(hooks.DefaultEntries()); err != nil {
			warn(fmt.Sprintf("%s hooks: %v", inst.Name(), err))
			continue
		}
		fmt.Printf("  ✓ %s — installed %d hook(s) → %s\n",
			inst.Name(), len(hooks.DefaultEntries()), inst.SettingsPath())
	}
	if !any {
		fmt.Println("  · no hook-capable clients detected")
	}
	return nil
}

// installRulesForAllDetected installs the protocol rules-file into every
// rules-only client we can detect.
func installRulesForAllDetected(dryRun bool) error {
	installers := []rules.Installer{
		rules.NewCursorInstaller(),
		rules.NewWindsurfInstaller(),
	}
	any := false
	for _, inst := range installers {
		if !inst.Detect() {
			fmt.Printf("  · %s not detected — skipping\n", inst.Name())
			continue
		}
		any = true
		path := inst.RulesPath()
		if dryRun {
			fmt.Printf("  · %s — would write to %s\n", inst.Name(), path)
			continue
		}
		if err := inst.Install(protocol.EnglishSessionStartText); err != nil {
			warn(fmt.Sprintf("%s rules: %v", inst.Name(), err))
			continue
		}
		fmt.Printf("  ✓ %s — installed protocol v%s → %s\n",
			inst.Name(), protocol.Version, path)
	}
	if !any {
		fmt.Println("  · no rules-only clients detected")
	}
	return nil
}

// promptVaultUnlockIfNeeded inspects encryption status and offers to unlock
// the vault when locked. No-ops when encryption is disabled or the vault is
// already unlocked.
func promptVaultUnlockIfNeeded() error {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		fmt.Println("  · vault: skipped (not authenticated)")
		return nil
	}
	client := api.NewClient()
	client.APIKey = cfg.APIKey
	encCfg, err := client.GetEncryptionConfig()
	if err != nil || encCfg == nil || !encCfg.EncryptionEnabled {
		fmt.Println("  ✓ vault: encryption disabled — nothing to unlock")
		return nil
	}
	if crypto.IsVaultUnlocked() {
		fmt.Println("  ✓ vault: already unlocked")
		return nil
	}
	fmt.Println("  · vault: encryption enabled but locked")
	fmt.Println("    Run `ramorie vault unlock` to unlock with your master password.")
	return nil
}

// runQuickDoctor prints an inline summary of installed surfaces and reports
// whether any check returned doctorFail. Returning the failure bit (instead of
// just printing) lets runFullSetup translate "one or more surfaces broken"
// into a non-zero exit code — without that, `ramorie setup` always exits 0
// even when the final health check found problems.
func runQuickDoctor() bool {
	results := collectDoctorChecks()
	failed := false
	for _, r := range results {
		fmt.Printf("  %s %s\n", r.symbol(), r.Message)
		if r.Status == doctorFail {
			failed = true
		}
	}
	return failed
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
	fmt.Println("  ramorie project list        - List your projects")
	fmt.Println("  ramorie task list           - List your tasks")
	fmt.Println("  ramorie task create \"...\"   - Create a new task")
	fmt.Println("  ramorie mcp serve           - Start MCP server")
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

