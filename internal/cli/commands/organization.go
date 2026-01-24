package commands

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// NewOrganizationCommand creates all subcommands for the 'organization' command group.
func NewOrganizationCommand() *cli.Command {
	return &cli.Command{
		Name:    "organization",
		Aliases: []string{"org"},
		Usage:   "Manage organizations",
		Subcommands: []*cli.Command{
			orgListCmd(),
			orgCreateCmd(),
			orgShowCmd(),
			orgUnlockCmd(),
			orgLockCmd(),
			orgEncryptionStatusCmd(),
			orgEncryptSetupCmd(),
			orgRotateKeyCmd(),
			// orgSwitchCmd(),    // TODO: Backend needs active org support
			// orgInviteCmd(),    // TODO: Backend needs member management
			// orgMembersCmd(),   // TODO: Backend needs member listing
			// orgLeaveCmd(),     // TODO: Backend needs leave functionality
		},
	}
}

// orgListCmd lists all organizations for the user.
func orgListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List your organizations",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			orgs, err := client.ListOrganizations()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(orgs) == 0 {
				fmt.Println("No organizations found. Use 'ramorie org create' to create one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tROLE")
			fmt.Fprintln(w, "--\t----\t-----------\t----")

			for _, org := range orgs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					org.ID[:8],
					truncateString(org.Name, 30),
					truncateString(org.Description, 40),
					"owner") // TODO: Backend should return role when member system is added
			}
			w.Flush()
			return nil
		},
	}
}

// orgCreateCmd creates a new organization.
func orgCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new organization",
		ArgsUsage: "[name]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Organization description",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization name is required")
			}
			name := c.Args().First()
			description := c.String("description")

			client := api.NewClient()
			org, err := client.CreateOrganization(name, description)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Organization '%s' created successfully!\n", org.Name)
			fmt.Printf("   ID: %s\n", org.ID[:8])
			if org.Description != "" {
				fmt.Printf("   Description: %s\n", org.Description)
			}
			return nil
		},
	}
}

// orgShowCmd shows details for a specific organization.
func orgShowCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Aliases:   []string{"info"},
		Usage:     "Show details for an organization",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required")
			}
			orgID := c.Args().First()

			client := api.NewClient()

			// If short ID provided, resolve to full ID
			if len(orgID) < 36 {
				orgs, err := client.ListOrganizations()
				if err != nil {
					fmt.Println(apierrors.ParseAPIError(err))
					return err
				}
				for _, org := range orgs {
					if strings.HasPrefix(org.ID, orgID) {
						orgID = org.ID
						break
					}
				}
			}

			org, err := client.GetOrganization(orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("Organization Details: %s\n", org.Name)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("ID:          %s\n", org.ID)
			fmt.Printf("Name:        %s\n", org.Name)
			fmt.Printf("Description: %s\n", org.Description)
			fmt.Printf("Owner ID:    %s\n", org.OwnerID[:8])
			fmt.Printf("Created At:  %s\n", org.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated At:  %s\n", org.UpdatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

// resolveOrgID resolves a short org ID to its full UUID
func resolveOrgID(client *api.Client, orgID string) (string, error) {
	if len(orgID) >= 36 {
		return orgID, nil
	}
	orgs, err := client.ListOrganizations()
	if err != nil {
		return "", err
	}
	for _, org := range orgs {
		if strings.HasPrefix(org.ID, orgID) {
			return org.ID, nil
		}
	}
	return "", fmt.Errorf("organization not found with ID prefix: %s", orgID)
}

// orgUnlockCmd unlocks the organization vault with passphrase
func orgUnlockCmd() *cli.Command {
	return &cli.Command{
		Name:      "unlock",
		Usage:     "Unlock organization vault with org passphrase",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required\n\nUsage: ramorie org unlock <org-id>")
			}
			orgID := c.Args().First()

			client := api.NewClient()

			// Resolve short ID
			fullOrgID, err := resolveOrgID(client, orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Check if already unlocked
			if crypto.IsOrgVaultUnlocked(fullOrgID) {
				fmt.Printf("\n✅ Organization vault is already unlocked (%s)\n", fullOrgID[:8])
				return nil
			}

			// First try auto-unlock via keyring (key stored from previous unlock)
			if crypto.HasStoredOrgKey(fullOrgID) {
				orgKey, err := crypto.RetrieveOrgKey(fullOrgID)
				if err == nil && orgKey != nil {
					// Restore org key in vault state
					crypto.RestoreOrgKeyFromKeyring(fullOrgID, orgKey)
					fmt.Printf("\n🔓 Organization vault unlocked (from keyring) - %s\n", fullOrgID[:8])
					return nil
				}
			}

			// Fetch org encryption config from API
			fmt.Print("🔄 Checking org encryption status...")
			encConfig, err := client.GetOrgEncryptionConfig(fullOrgID)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to check org encryption: %v\n", apierrors.ParseAPIError(err))
				return nil
			}

			if !encConfig.IsEnabled {
				fmt.Println(" ℹ️")
				fmt.Println("\nℹ️  Encryption is not enabled for this organization.")
				fmt.Println("   An admin can enable it via the web dashboard or 'ramorie org encrypt-setup'.")
				return nil
			}
			fmt.Println(" ✅")

			fmt.Println()
			fmt.Printf("🔐 Unlock Organization Vault (%s)\n", fullOrgID[:8])
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()

			// Secure passphrase input
			fmt.Print("Organization Passphrase: ")
			passphraseBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("could not read passphrase: %w", err)
			}
			passphrase := string(passphraseBytes)

			if passphrase == "" {
				return fmt.Errorf("passphrase is required")
			}

			fmt.Print("🔄 Verifying passphrase...")

			// Derive passphrase hash for server verification
			salt, err := crypto.Base64ToBytes(encConfig.Salt)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nInvalid salt in config: %v\n", err)
				return nil
			}

			iterations := encConfig.KDFIterations
			if iterations == 0 {
				iterations = crypto.PBKDF2Iterations
			}

			passphraseHash, err := crypto.CreateOrgPassphraseHash(passphrase, salt, iterations)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to derive hash: %v\n", err)
				return nil
			}

			// Verify passphrase hash with server
			verified, err := client.VerifyOrgPassphrase(fullOrgID, passphraseHash)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nVerification failed: %v\n", apierrors.ParseAPIError(err))
				return nil
			}
			if !verified {
				fmt.Println(" ❌")
				fmt.Println("\nIncorrect passphrase. Please try again.")
				return nil
			}
			fmt.Println(" ✅")

			fmt.Print("🔄 Deriving org key...")

			// Build org vault config and unlock
			orgConfig := &crypto.OrgVaultConfig{
				OrgID:         fullOrgID,
				Salt:          encConfig.Salt,
				KDFIterations: encConfig.KDFIterations,
				KDFAlgorithm:  encConfig.KDFAlgorithm,
				IsEnabled:     encConfig.IsEnabled,
			}

			err = crypto.UnlockOrgVault(fullOrgID, passphrase, orgConfig)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to unlock: %v\n", err)
				return nil
			}

			fmt.Println(" ✅")
			fmt.Println()
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Printf("🔓 Organization vault unlocked! (%s)\n", fullOrgID[:8])
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("Organization-encrypted data will now be decrypted automatically.")
			fmt.Printf("Run 'ramorie org lock %s' to lock the vault when done.\n", fullOrgID[:8])
			fmt.Println()
			return nil
		},
	}
}

// orgLockCmd locks the organization vault
func orgLockCmd() *cli.Command {
	return &cli.Command{
		Name:      "lock",
		Usage:     "Lock organization vault (clear org key from memory)",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required\n\nUsage: ramorie org lock <org-id>")
			}
			orgID := c.Args().First()

			client := api.NewClient()
			fullOrgID, err := resolveOrgID(client, orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if !crypto.IsOrgVaultUnlocked(fullOrgID) {
				fmt.Printf("\nℹ️  Organization vault is already locked (%s)\n", fullOrgID[:8])
				return nil
			}

			crypto.LockOrgVault(fullOrgID)

			fmt.Println()
			fmt.Printf("🔒 Organization vault locked (%s)\n", fullOrgID[:8])
			fmt.Println("   Organization-encrypted data is no longer accessible.")
			fmt.Println()
			return nil
		},
	}
}

// orgEncryptionStatusCmd shows the encryption status for an organization
func orgEncryptionStatusCmd() *cli.Command {
	return &cli.Command{
		Name:    "encryption-status",
		Aliases: []string{"enc-status"},
		Usage:   "Show organization encryption status",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required\n\nUsage: ramorie org encryption-status <org-id>")
			}
			orgID := c.Args().First()

			client := api.NewClient()
			fullOrgID, err := resolveOrgID(client, orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Fetch org encryption config (has algorithm/iterations)
			encConfig, err := client.GetOrgEncryptionConfig(fullOrgID)
			if err != nil {
				fmt.Printf("\nFailed to get encryption status: %v\n", apierrors.ParseAPIError(err))
				return nil
			}

			fmt.Println()
			fmt.Printf("🔐 Organization Encryption Status (%s)\n", fullOrgID[:8])
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			if !encConfig.IsEnabled {
				fmt.Println("Status: Not enabled")
				fmt.Println()
				fmt.Println("Run 'ramorie org encrypt-setup <org-id>' to set up encryption.")
				return nil
			}

			fmt.Printf("Status:     Enabled\n")
			fmt.Printf("Version:    %d\n", encConfig.EncryptionVersion)
			fmt.Printf("Algorithm:  %s\n", encConfig.KDFAlgorithm)
			fmt.Printf("Iterations: %d\n", encConfig.KDFIterations)

			// Show local vault status
			if crypto.IsOrgVaultUnlocked(fullOrgID) {
				fmt.Printf("Local:      🔓 Unlocked\n")
			} else {
				fmt.Printf("Local:      🔒 Locked\n")
			}

			// Show keyring status
			if crypto.HasStoredOrgKey(fullOrgID) {
				fmt.Printf("Keyring:    ✅ Key stored (%s)\n", crypto.GetStorageMode())
			} else {
				fmt.Printf("Keyring:    ❌ No key stored\n")
			}

			fmt.Println()
			return nil
		},
	}
}

// orgEncryptSetupCmd sets up organization encryption (owner/admin only)
func orgEncryptSetupCmd() *cli.Command {
	return &cli.Command{
		Name:      "encrypt-setup",
		Usage:     "Set up organization encryption (owner/admin only)",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required\n\nUsage: ramorie org encrypt-setup <org-id>")
			}
			orgID := c.Args().First()

			client := api.NewClient()
			fullOrgID, err := resolveOrgID(client, orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Check if encryption is already enabled
			encConfig, err := client.GetOrgEncryptionConfig(fullOrgID)
			if err == nil && encConfig.IsEnabled {
				fmt.Println("\n⚠️  Organization encryption is already enabled.")
				fmt.Printf("   Use 'ramorie org unlock %s' to unlock the vault.\n", fullOrgID[:8])
				return nil
			}

			fmt.Println()
			fmt.Printf("🔐 Set Up Organization Encryption (%s)\n", fullOrgID[:8])
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("⚠️  IMPORTANT: This passphrase protects all organization-encrypted data.")
			fmt.Println("   If lost, encrypted data will be PERMANENTLY inaccessible.")
			fmt.Println("   Share it securely with team members who need access.")
			fmt.Println()
			fmt.Println("Requirements: 12+ chars, uppercase, lowercase, number, symbol")
			fmt.Println()

			// Read passphrase
			fmt.Print("Organization Passphrase: ")
			pass1Bytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("could not read passphrase: %w", err)
			}
			passphrase := string(pass1Bytes)

			if len(passphrase) < 12 {
				fmt.Println("\n❌ Passphrase must be at least 12 characters.")
				return nil
			}

			// Confirm passphrase
			fmt.Print("Confirm Passphrase:      ")
			pass2Bytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("could not read passphrase: %w", err)
			}

			if string(pass2Bytes) != passphrase {
				fmt.Println("\n❌ Passphrases do not match. Please try again.")
				return nil
			}

			fmt.Print("\n🔄 Setting up organization encryption...")

			// Generate salt for PBKDF2
			salt, err := crypto.GenerateSalt()
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to generate salt: %v\n", err)
				return nil
			}

			iterations := crypto.PBKDF2Iterations
			saltBase64 := crypto.BytesToBase64(salt)

			// Create passphrase hash for server verification
			passphraseHash, err := crypto.CreateOrgPassphraseHash(passphrase, salt, iterations)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to create hash: %v\n", err)
				return nil
			}

			// Call backend to set up encryption
			err = client.SetupOrgEncryption(fullOrgID, saltBase64, passphraseHash, iterations)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nSetup failed: %v\n", apierrors.ParseAPIError(err))
				return nil
			}

			fmt.Println(" ✅")

			// Auto-unlock after setup
			encConfig, err = client.GetOrgEncryptionConfig(fullOrgID)
			if err == nil && encConfig.IsEnabled {
				orgConfig := &crypto.OrgVaultConfig{
					OrgID:         fullOrgID,
					Salt:          encConfig.Salt,
					KDFIterations: encConfig.KDFIterations,
					KDFAlgorithm:  encConfig.KDFAlgorithm,
					IsEnabled:     true,
				}
				_ = crypto.UnlockOrgVault(fullOrgID, passphrase, orgConfig)
			}

			fmt.Println()
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println("✅ Organization encryption enabled!")
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("Share the passphrase securely with team members.")
			fmt.Println("They can unlock with: ramorie org unlock " + fullOrgID[:8])
			fmt.Println()
			return nil
		},
	}
}

// orgRotateKeyCmd rotates the organization encryption key (admin only)
func orgRotateKeyCmd() *cli.Command {
	return &cli.Command{
		Name:      "rotate-key",
		Usage:     "Rotate organization encryption key (owner/admin only)",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required\n\nUsage: ramorie org rotate-key <org-id>")
			}
			orgID := c.Args().First()

			client := api.NewClient()
			fullOrgID, err := resolveOrgID(client, orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Check encryption is enabled
			encConfig, err := client.GetOrgEncryptionConfig(fullOrgID)
			if err != nil || !encConfig.IsEnabled {
				fmt.Println("\n❌ Organization encryption is not enabled.")
				return nil
			}

			fmt.Println()
			fmt.Printf("🔑 Rotate Organization Key (%s)\n", fullOrgID[:8])
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("⚠️  WARNING: Key rotation will:")
			fmt.Println("   - Invalidate all member auto-unlock keys")
			fmt.Println("   - Require all members to re-enter the new passphrase")
			fmt.Println("   - Existing data remains encrypted with old key until re-encrypted")
			fmt.Println()
			fmt.Print("Continue? (yes/no): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Operation cancelled.")
				return nil
			}

			fmt.Println()
			fmt.Println("Requirements: 12+ chars, uppercase, lowercase, number, symbol")
			fmt.Println()

			// Read new passphrase
			fmt.Print("New Passphrase: ")
			pass1Bytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("could not read passphrase: %w", err)
			}
			newPassphrase := string(pass1Bytes)

			if len(newPassphrase) < 12 {
				fmt.Println("\n❌ Passphrase must be at least 12 characters.")
				return nil
			}

			// Confirm
			fmt.Print("Confirm Passphrase: ")
			pass2Bytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("could not read passphrase: %w", err)
			}

			if string(pass2Bytes) != newPassphrase {
				fmt.Println("\n❌ Passphrases do not match.")
				return nil
			}

			fmt.Print("\n🔄 Rotating encryption key...")

			// Generate new salt
			newSalt, err := crypto.GenerateSalt()
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to generate salt: %v\n", err)
				return nil
			}

			iterations := crypto.PBKDF2Iterations
			newSaltBase64 := crypto.BytesToBase64(newSalt)

			// Create new passphrase hash
			newHash, err := crypto.CreateOrgPassphraseHash(newPassphrase, newSalt, iterations)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nFailed to create hash: %v\n", err)
				return nil
			}

			// Call backend to rotate key
			err = client.RotateOrgEncryption(fullOrgID, newSaltBase64, newHash, iterations)
			if err != nil {
				fmt.Println(" ❌")
				fmt.Printf("\nRotation failed: %v\n", apierrors.ParseAPIError(err))
				return nil
			}

			fmt.Println(" ✅")

			// Lock the old org vault and unlock with new key
			crypto.LockOrgVault(fullOrgID)
			newConfig, err := client.GetOrgEncryptionConfig(fullOrgID)
			if err == nil && newConfig.IsEnabled {
				orgConfig := &crypto.OrgVaultConfig{
					OrgID:         fullOrgID,
					Salt:          newConfig.Salt,
					KDFIterations: newConfig.KDFIterations,
					KDFAlgorithm:  newConfig.KDFAlgorithm,
					IsEnabled:     true,
				}
				_ = crypto.UnlockOrgVault(fullOrgID, newPassphrase, orgConfig)
			}

			fmt.Println()
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println("✅ Encryption key rotated successfully!")
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("All members must re-enter the new passphrase.")
			fmt.Println("Previously encrypted data needs re-encryption with the new key.")
			fmt.Println()
			return nil
		},
	}
}

// orgSwitchCmd switches the active organization.
// TODO: Implement when backend supports active organization
/*
func orgSwitchCmd() *cli.Command {
	return &cli.Command{
		Name:      "switch",
		Aliases:   []string{"use"},
		Usage:     "Switch to a different organization",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required")
			}
			orgID := c.Args().First()

			client := api.NewClient()

			// Resolve short ID to full ID if needed
			if len(orgID) < 36 {
				orgs, err := client.ListOrganizations()
				if err != nil {
					fmt.Println(apierrors.ParseAPIError(err))
					return err
				}
				for _, org := range orgs {
					if strings.HasPrefix(org.ID, orgID) {
						orgID = org.ID
						break
					}
				}
			}

			err := client.SwitchOrganization(orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Switched to organization %s\n", orgID[:8])
			return nil
		},
	}
}
*/

// orgInviteCmd invites a member to the organization.
// TODO: Implement when backend supports member management
/*
func orgInviteCmd() *cli.Command {
	return &cli.Command{
		Name:      "invite",
		Usage:     "Invite a member to the organization",
		ArgsUsage: "[email]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "role",
				Aliases: []string{"r"},
				Usage:   "Role for the member (member, admin)",
				Value:   "member",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("email address is required")
			}
			email := c.Args().First()
			role := c.String("role")

			client := api.NewClient()
			err := client.InviteOrganizationMember(email, role)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Invitation sent to %s with role: %s\n", email, role)
			return nil
		},
	}
}
*/

// orgMembersCmd lists organization members.
// TODO: Implement when backend supports member listing
/*
func orgMembersCmd() *cli.Command {
	return &cli.Command{
		Name:    "members",
		Aliases: []string{"ls-members"},
		Usage:   "List organization members",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			members, err := client.ListOrganizationMembers()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(members) == 0 {
				fmt.Println("No members found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "EMAIL\tROLE\tJOINED")
			fmt.Fprintln(w, "-----\t----\t------")

			for _, member := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					member.Email,
					member.Role,
					member.JoinedAt.Format("2006-01-02"))
			}
			w.Flush()
			return nil
		},
	}
}
*/

// orgLeaveCmd leaves the organization.
// TODO: Implement when backend supports leave functionality
/*
func orgLeaveCmd() *cli.Command {
	return &cli.Command{
		Name:  "leave",
		Usage: "Leave the organization",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force leave without confirmation",
			},
		},
		Action: func(c *cli.Context) error {
			force := c.Bool("force")

			if !force {
				fmt.Print("Are you sure you want to leave the organization? (yes/no): ")
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Operation cancelled.")
					return nil
				}
			}

			client := api.NewClient()
			err := client.LeaveOrganization()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Println("✅ You have left the organization.")
			return nil
		},
	}
}
*/
