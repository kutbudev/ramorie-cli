package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/cli/resolve"
	"github.com/kutbudev/ramorie-cli/internal/constants"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// NewMemoryCommand creates all subcommands for the 'memory' command group.
func NewMemoryCommand() *cli.Command {
	return &cli.Command{
		Name:    "memory",
		Aliases: []string{"m", "memories"},
		Usage:   "Manage memories (knowledge base)",
		Subcommands: []*cli.Command{
			memoriesCmd(),
			getCmd(),
			forgetCmd(),
			memoryLinkCmd(),
			memoryLinksCmd(),
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
		ArgsUsage:              "[content]   (or pipe via stdin)",
		UseShortOptionHandling: true,
		Description: `Create a new memory entry, scoped to a project.

Project resolution accepts a name, short ID prefix, or full UUID — same
semantics as ` + "`task list -p`" + ` and ` + "`kanban -p`" + `.

Content sources (in priority order):
  1. Positional argument(s) — joined with spaces
  2. Piped stdin when no positional is given and stdin is not a TTY

For multi-line content with leading "-" bullets, prefer piping via stdin
(or use the "--" separator) so urfave/cli doesn't treat lines as flags:

  cat memory.md | ramorie remember -p "Ramorie Backend"
  ramorie remember -p "Ramorie Backend" -- "- bullet one"`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "Project (name | short id | UUID)",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:    "tags",
				Aliases: []string{"t"},
				Usage:   "Tags (comma-separated or repeated -t)",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Print result as JSON (for agents / scripts)",
			},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()

			// 1. Resolve project (name, short id, or UUID).
			projectArg := c.String("project")
			projectID, err := resolve.ResolveProject(projectArg, client)
			if err != nil {
				return err
			}

			// 2. Get content from positionals or piped stdin.
			content := strings.TrimSpace(strings.Join(c.Args().Slice(), " "))
			if content == "" {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					b, readErr := io.ReadAll(os.Stdin)
					if readErr != nil {
						return fmt.Errorf("read stdin: %w", readErr)
					}
					content = strings.TrimRight(string(b), "\n")
				}
			}
			if content == "" {
				return fmt.Errorf("memory content is required (positional arg or piped stdin)")
			}

			// 3. Tags: split CSV — turn -t "a,b" into [a,b].
			rawTags := c.StringSlice("tags")
			tags := make([]string, 0, len(rawTags))
			for _, t := range rawTags {
				for _, sub := range strings.Split(t, ",") {
					s := strings.TrimSpace(sub)
					if s != "" {
						tags = append(tags, s)
					}
				}
			}

			// 4. Length guard.
			if !constants.IsWithinMemoryLimit(content) {
				chars, tokens, usage := constants.GetContentStats(content)
				fmt.Fprintf(os.Stderr, "❌ Content exceeds maximum limit!\n")
				fmt.Fprintf(os.Stderr, "   Your content: %d chars (~%d tokens)\n", chars, tokens)
				fmt.Fprintf(os.Stderr, "   Maximum: %d chars (~%d tokens)\n", constants.MaxMemoryChars, constants.MaxMemoryChars/constants.CharsPerToken)
				fmt.Fprintf(os.Stderr, "   Usage: %.1f%%\n", usage)
				return fmt.Errorf("content too large")
			}
			chars, tokens, usage := constants.GetContentStats(content)
			if usage >= constants.WarningThresholdPercent {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Content is %.1f%% of maximum limit (%d chars)\n", usage, chars)
			}

			// 5. Encryption decision: org projects skip personal-vault encryption.
			isOrgProject := false
			projects, _ := client.ListProjects()
			for _, p := range projects {
				if p.ID.String() == projectID && p.OrganizationID != nil {
					isOrgProject = true
					break
				}
			}

			var memory *models.Memory
			if crypto.IsVaultUnlocked() && !isOrgProject {
				contentHash := crypto.ComputeContentHash(content)
				encryptedContent, nonce, isEncrypted, encErr := crypto.EncryptContent(content)
				if encErr != nil {
					return fmt.Errorf("encryption failed: %w", encErr)
				}
				if isEncrypted {
					memory, err = client.CreateEncryptedMemory(projectID, encryptedContent, nonce, contentHash, tags...)
				} else {
					memory, err = client.CreateMemory(projectID, content, tags...)
				}
			} else {
				memory, err = client.CreateMemory(projectID, content, tags...)
			}
			if err != nil {
				return fmt.Errorf("%s", apierrors.ParseAPIError(err))
			}

			// 6. Output.
			if c.Bool("json") {
				out := map[string]interface{}{
					"id":         memory.ID.String(),
					"project_id": projectID,
					"type":       memory.Type,
					"encrypted":  memory.IsEncrypted,
					"chars":      chars,
					"tokens":     tokens,
					"tags":       tags,
				}
				if memory.LinkedTaskID != nil {
					out["linked_task_id"] = memory.LinkedTaskID.String()
				}
				b, mErr := json.MarshalIndent(out, "", "  ")
				if mErr != nil {
					return fmt.Errorf("marshal json: %w", mErr)
				}
				fmt.Println(string(b))
				return nil
			}

			// Human-readable output.
			if memory.IsEncrypted {
				fmt.Printf("🔐 Memory encrypted and stored! (ID: %s)\n", memory.ID.String()[:8])
			} else {
				fmt.Printf("🧠 Memory stored successfully! (ID: %s)\n", memory.ID.String()[:8])
			}
			fmt.Printf("   Size: %d chars (~%d tokens)\n", chars, tokens)
			if len(tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(tags, ", "))
			}
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
		Name:                   "list",
		Aliases:                []string{"ls"},
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
			&cli.BoolFlag{
				Name:  "newest-first",
				Usage: "Show newest item at the top (default: oldest at top)",
			},
		},
		Action: func(c *cli.Context) error {
			projectArg := c.String("project")
			showAll := c.Bool("all")
			orgOnly := c.Bool("org-only")
			limit := c.Int("limit")
			tagFilter := c.String("tag")
			newestFirst := c.Bool("newest-first")

			client := api.NewClient()

			projectID := ""
			if projectArg != "" && !showAll {
				resolved, err := resolve.ResolveProject(projectArg, client)
				if err != nil {
					return err
				}
				projectID = resolved
			}

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

			// Apply limit if specified (slice the newest N from DESC backend response)
			if limit > 0 && len(memories) > limit {
				memories = memories[:limit]
			}

			// Default: chronological asc — oldest at top, newest at bottom
			// (pipe to `tail` to see the most recent). `--newest-first` keeps
			// the legacy DESC order.
			if !newestFirst {
				slices.Reverse(memories)
			}

			countPart := fmt.Sprintf("🧠 %d memor", len(memories))
			if len(memories) == 1 {
				countPart += "y"
			} else {
				countPart += "ies"
			}
			subtitle := "oldest first"
			if newestFirst {
				subtitle = "newest first"
			}
			fmt.Println(display.Header(countPart, subtitle))
			fmt.Println()

			cols := []display.Column{
				{Title: "TYPE", Min: 12, Weight: 0}, // [general] etc — already padded
				{Title: "ID", Min: 8, Weight: 0},
				{Title: "PROJECT", Min: 10, Weight: 1},
				{Title: "PREVIEW", Min: 24, Weight: 4},
				{Title: "TAGS", Min: 14, Weight: 1},
				{Title: "AGE", Min: 8, Weight: 0},
			}
			rows := make([][]string, 0, len(memories))
			for _, m := range memories {
				proj := ""
				if m.Project != nil {
					proj = m.Project.Name
				}
				// CRITICAL: Decrypt memory content before displaying
				preview := display.SingleLine(decryptMemoryForCLI(&m))
				tags := ""
				if tagList := getTagsAsStrings(m.Tags); len(tagList) > 0 {
					tags = display.Tags(tagList, 3)
				}
				rows = append(rows, []string{
					display.TypeBadge(m.Type),
					display.Dim.Render(m.ID.String()[:8]),
					display.Dim.Render(proj),
					preview,
					tags,
					display.Dim.Render(display.Relative(m.UpdatedAt)),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
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

// memoryLinkCmd creates a manual memory↔task link from the memory side.
func memoryLinkCmd() *cli.Command {
	return &cli.Command{
		Name:      "link",
		Usage:     "Link this memory to a task",
		ArgsUsage: "<memory-id> <task-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: ramorie memory link <memory-id> <task-id>")
			}
			memoryID := c.Args().Get(0)
			taskID := c.Args().Get(1)
			client := api.NewClient()
			if _, err := client.CreateMemoryTaskLink(taskID, memoryID, ""); err != nil {
				return fmt.Errorf("failed to link: %w", err)
			}
			shortMem := memoryID
			if len(shortMem) > 8 {
				shortMem = shortMem[:8]
			}
			shortTask := taskID
			if len(shortTask) > 8 {
				shortTask = shortTask[:8]
			}
			fmt.Printf("✅ Linked memory %s ↔ task %s\n", shortMem, shortTask)
			return nil
		},
	}
}

// memoryLinksCmd lists tasks linked to a memory.
func memoryLinksCmd() *cli.Command {
	return &cli.Command{
		Name:      "links",
		Usage:     "List tasks linked to a memory",
		ArgsUsage: "<memory-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("memory ID is required")
			}
			memoryID := c.Args().First()
			client := api.NewClient()
			tasks, err := client.ListMemoryTasks(memoryID)
			if err != nil {
				return err
			}
			if len(tasks) == 0 {
				fmt.Println(display.Dim.Render("  no linked tasks"))
				return nil
			}
			cols := []display.Column{
				{Title: "S", Min: 3, Weight: 0},
				{Title: "P", Min: 3, Weight: 0},
				{Title: "ID", Min: 8, Weight: 0},
				{Title: "TITLE", Min: 24, Weight: 4},
				{Title: "UPDATED", Min: 10, Weight: 0},
			}
			rows := make([][]string, 0, len(tasks))
			for _, t := range tasks {
				title, _ := decryptTaskForCLI(&t)
				rows = append(rows, []string{
					display.StatusIcon(t.Status),
					display.PriorityBadge(t.Priority),
					display.Dim.Render(t.ID.String()[:8]),
					display.SingleLine(title),
					display.Dim.Render(display.Relative(t.UpdatedAt)),
				})
			}
			fmt.Println(display.NewResponsiveTable(cols, rows))
			return nil
		},
	}
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
