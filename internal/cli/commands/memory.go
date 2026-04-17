package commands

import (
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/constants"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/urfave/cli/v2"
)

// NewMemoryCommand creates all subcommands for the 'memory' command group.
func NewMemoryCommand() *cli.Command {
	return &cli.Command{
		Name:    "memory",
		Aliases: []string{"m", "memories"},
		Usage:   "Manage memories (knowledge base)",
		Subcommands: []*cli.Command{
			rememberCmd(),
			memoriesCmd(),
			getCmd(),
			recallCmd(),
			forgetCmd(),
		},
	}
}

// NewRememberCommand creates a standalone remember command
func NewRememberCommand() *cli.Command {
	return rememberCmd()
}

// rememberCmd creates a new memory item.
func rememberCmd() *cli.Command {
	return &cli.Command{
		Name:                   "remember",
		Usage:                  "Create a new memory",
		ArgsUsage:              "[content]",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "Project ID (required)",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:    "tags",
				Aliases: []string{"t"},
				Usage:   "Tags for the memory (can be used multiple times or comma-separated)",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory content is required")
			}
			content := c.Args().First()
			projectID := c.String("project")
			tags := c.StringSlice("tags")

			// Check content length limit before sending
			if !constants.IsWithinMemoryLimit(content) {
				chars, tokens, usage := constants.GetContentStats(content)
				fmt.Printf("❌ Content exceeds maximum limit!\n")
				fmt.Printf("   Your content: %d chars (~%d tokens)\n", chars, tokens)
				fmt.Printf("   Maximum: %d chars (~%d tokens)\n", constants.MaxMemoryChars, constants.MaxMemoryChars/constants.CharsPerToken)
				fmt.Printf("   Usage: %.1f%%\n", usage)
				return fmt.Errorf("content too large")
			}

			// Show warning if approaching limit (80%+)
			chars, tokens, usage := constants.GetContentStats(content)
			if usage >= constants.WarningThresholdPercent {
				fmt.Printf("⚠️  Warning: Content is %.1f%% of maximum limit (%d chars)\n", usage, chars)
			}

			client := api.NewClient()

			// Check if project belongs to an org (org projects skip encryption)
			isOrgProject := false
			if projectID != "" {
				projects, _ := client.ListProjects()
				for _, p := range projects {
					if p.ID.String() == projectID && p.OrganizationID != nil {
						isOrgProject = true
						break
					}
				}
			}

			// Check if vault is unlocked for encryption
			var memory *models.Memory
			var err error

			if crypto.IsVaultUnlocked() && !isOrgProject {
				// Personal project only — encrypt with personal key
				contentHash := crypto.ComputeContentHash(content)

				encryptedContent, nonce, isEncrypted, encErr := crypto.EncryptContent(content)
				if encErr != nil {
					return fmt.Errorf("encryption failed: %w", encErr)
				}

				if isEncrypted {
					memory, err = client.CreateEncryptedMemory(projectID, encryptedContent, nonce, contentHash, tags...)
					if err == nil {
						fmt.Printf("🔐 Memory encrypted and stored! (ID: %s)\n", memory.ID.String()[:8])
					}
				} else {
					memory, err = client.CreateMemory(projectID, content, tags...)
					if err == nil {
						fmt.Printf("🧠 Memory stored successfully! (ID: %s)\n", memory.ID.String()[:8])
					}
				}
			} else {
				// Org project or vault locked — send plaintext
				memory, err = client.CreateMemory(projectID, content, tags...)
				if err == nil {
					fmt.Printf("🧠 Memory stored successfully! (ID: %s)\n", memory.ID.String()[:8])
				}
			}

			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("   Size: %d chars (~%d tokens)\n", chars, tokens)
			if len(tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(tags, ", "))
			}

			// Show if memory was auto-linked to active task
			if memory.LinkedTaskID != nil {
				fmt.Printf("🔗 Auto-linked to active task: %s\n", memory.LinkedTaskID.String()[:8])
			}
			return nil
		},
	}
}

// memoriesCmd lists all memory items.
func memoriesCmd() *cli.Command {
	return &cli.Command{
		Name:                   "memories",
		Usage:                  "List all memories",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "List memories from all projects (including organization projects)",
			},
			&cli.BoolFlag{
				Name:  "org-only",
				Usage: "Only show memories from organization projects",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Limit number of results",
				Value:   0,
			},
			&cli.StringFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Filter by tag",
			},
		},
		Action: func(c *cli.Context) error {
			projectID := c.String("project")
			showAll := c.Bool("all")
			orgOnly := c.Bool("org-only")
			limit := c.Int("limit")
			tagFilter := c.String("tag")


			// If --all flag, don't filter by project
			if showAll {
				projectID = ""
			}

			client := api.NewClient()
			memories, err := client.ListMemories(projectID, "") // No search query
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Filter by tag if requested
			if tagFilter != "" {
				var filtered []models.Memory
				for _, m := range memories {
					tags := getTagsAsStrings(m.Tags)
					for _, tag := range tags {
						if strings.EqualFold(tag, tagFilter) {
							filtered = append(filtered, m)
							break
						}
					}
				}
				memories = filtered
			}

			// Filter by org-only if requested
			if orgOnly {
				var filtered []models.Memory
				for _, m := range memories {
					if m.Project != nil && m.Project.Organization != nil {
						filtered = append(filtered, m)
					}
				}
				memories = filtered
			}

			if len(memories) == 0 {
				fmt.Println(display.Dim.Render("  no memories — use `ramorie remember` to add one"))
				return nil
			}

			// Apply limit if specified
			if limit > 0 && len(memories) > limit {
				memories = memories[:limit]
			}

			countPart := fmt.Sprintf("🧠 %d memor", len(memories))
			if len(memories) == 1 {
				countPart += "y"
			} else {
				countPart += "ies"
			}
			fmt.Println(display.Header(countPart, ""))
			fmt.Println()

			// Budget content width: type badge (12) + short id (8) + age (12) +
			// tag tail (~20) + spaces ≈ 55 reserved.
			contentWidth := display.TerminalWidth() - 60
			if contentWidth < 30 {
				contentWidth = 30
			}

			for _, m := range memories {
				// CRITICAL: Decrypt memory content before displaying
				content := display.SingleLine(decryptMemoryForCLI(&m))
				content = display.Truncate(content, contentWidth)

				tagLine := ""
				if tags := getTagsAsStrings(m.Tags); len(tags) > 0 {
					tagLine = display.Sep() + display.Tags(tags, 3)
				}

				fmt.Printf(" %s %s  %s%s%s\n",
					display.TypeBadge(m.Type),
					display.Dim.Render(m.ID.String()[:8]),
					content,
					display.Sep()+display.Dim.Render(display.Relative(m.UpdatedAt)),
					tagLine,
				)
			}
			return nil
		},
	}
}

// recallCmd searches memory items.
func recallCmd() *cli.Command {
	return &cli.Command{
		Name:                   "recall",
		Usage:                  "Search within your memories",
		ArgsUsage:              "[search-query]",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Filter by project ID",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Limit number of results",
				Value:   20,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("a search query is required")
			}
			query := c.Args().First()
			projectID := c.String("project")
			limit := c.Int("limit")

			client := api.NewClient()
			memories, err := client.ListMemories(projectID, query)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Apply limit
			if limit > 0 && len(memories) > limit {
				memories = memories[:limit]
			}

			if len(memories) == 0 {
				fmt.Println(display.Dim.Render(fmt.Sprintf("  no matches for %q", query)))
				return nil
			}

			countPart := fmt.Sprintf("🔍 %d match", len(memories))
			if len(memories) != 1 {
				countPart += "es"
			}
			fmt.Println(display.Header(countPart, fmt.Sprintf("query: %q", query)))
			fmt.Println()

			contentWidth := display.TerminalWidth() - 60
			if contentWidth < 30 {
				contentWidth = 30
			}
			for _, m := range memories {
				content := display.SingleLine(decryptMemoryForCLI(&m))
				content = display.Truncate(content, contentWidth)
				tagLine := ""
				if tags := getTagsAsStrings(m.Tags); len(tags) > 0 {
					tagLine = display.Sep() + display.Tags(tags, 3)
				}
				fmt.Printf(" %s %s  %s%s%s\n",
					display.TypeBadge(m.Type),
					display.Dim.Render(m.ID.String()[:8]),
					content,
					display.Sep()+display.Dim.Render(display.Relative(m.UpdatedAt)),
					tagLine,
				)
			}
			return nil
		},
	}
}

// getCmd retrieves a memory item by ID.
func getCmd() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Retrieve a memory by ID",
		ArgsUsage: "[memory-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory ID is required")
			}
			memoryID := c.Args().First()

			client := api.NewClient()
			memory, err := client.GetMemory(memoryID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// CRITICAL: Decrypt memory content before displaying
			displayContent := decryptMemoryForCLI(memory)

			// Title — use first non-empty line as a banner
			title, body := splitFirstLine(displayContent)
			if title == "" {
				title = "(empty)"
			}
			fmt.Println(display.Title.Render(title))

			// Metadata row
			meta := []string{display.TypeBadge(memory.Type)}
			if memory.Project != nil && memory.Project.Name != "" {
				meta = append(meta, display.Dim.Render(memory.Project.Name))
			}
			meta = append(meta, display.Dim.Render("updated "+display.Relative(memory.UpdatedAt)))
			meta = append(meta, display.Dim.Render(memory.ID.String()[:8]))
			fmt.Println(strings.Join(meta, display.Sep()))

			// Tags
			if tags := getTagsAsStrings(memory.Tags); len(tags) > 0 {
				fmt.Println("  " + display.Tags(tags, 15))
			}

			// Body
			if strings.TrimSpace(body) != "" {
				fmt.Println()
				fmt.Println(body)
			}
			return nil
		},
	}
}

// splitFirstLine returns (headingLine, remainingBody). Strips a leading "# "
// so a markdown-style heading shows cleanly in the banner.
func splitFirstLine(s string) (head, rest string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	lines := strings.SplitN(s, "\n", 2)
	head = strings.TrimLeft(lines[0], "# ")
	if len(lines) > 1 {
		rest = strings.TrimSpace(lines[1])
	}
	return head, rest
}

// forgetCmd deletes a memory item.
func forgetCmd() *cli.Command {
	return &cli.Command{
		Name:      "forget",
		Usage:     "Delete a memory",
		ArgsUsage: "[memory-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory ID is required")
			}
			memoryID := c.Args().First()

			client := api.NewClient()
			err := client.DeleteMemory(memoryID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("🗑️ Memory %s forgotten successfully.\n", memoryID[:8])
			return nil
		},
	}
}

// getTagsAsStrings converts interface{} tags to []string
func getTagsAsStrings(tags interface{}) []string {
	if tags == nil {
		return nil
	}

	// Try []interface{} first (common JSON unmarshaling result)
	if arr, ok := tags.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	// Try []string directly
	if arr, ok := tags.([]string); ok {
		return arr
	}

	return nil
}

// decryptMemoryForCLI decrypts memory content if encrypted and vault is unlocked.
// Returns the plaintext content or a fallback message.
func decryptMemoryForCLI(m *models.Memory) string {
	if !m.IsEncrypted {
		return m.Content
	}

	// Check if we have encrypted content to decrypt
	if m.EncryptedContent == "" {
		// Content might be "[Encrypted]" placeholder from backend
		return m.Content
	}

	if !crypto.IsVaultUnlocked() {
		return "[Vault Locked - run 'ramorie vault unlock']"
	}

	plaintext, err := crypto.DecryptContent(m.EncryptedContent, m.ContentNonce, true)
	if err != nil {
		return "[Decryption Failed]"
	}

	return plaintext
}
