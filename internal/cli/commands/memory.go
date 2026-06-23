package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/cli/resolve"
	"github.com/kutbudev/ramorie-cli/internal/constants"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	"github.com/kutbudev/ramorie-cli/internal/encstate"
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
			memoryHygieneCmd(),
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

Project resolution accepts a name, short ID prefix, or full UUID. When -p is
omitted the project is auto-detected from the current directory name, the git
remote, your single project, or the last project you used — so inside a project
directory you can just run ` + "`ramorie remember \"my note\"`" + `.

-p may also be placed AFTER the content; it is rescued even though urfave/cli
would otherwise swallow it as positional text.

Content sources (in priority order):
  1. Positional argument(s) — joined with spaces
  2. Piped stdin when no positional is given and stdin is not a TTY

For multi-line content with leading "-" bullets, prefer piping via stdin
(or use the "--" separator) so urfave/cli doesn't treat lines as flags:

  cat memory.md | ramorie remember -p "Ramorie Backend"
  ramorie remember "my note"                 # project auto-detected
  ramorie remember "my note" -p ramorie-cli  # -p after content also works`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project (name | short id | UUID). Optional: auto-detected when omitted.",
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

			// 1. Resolve project (name, short id, UUID, or auto-detect).
			//    Rescue a -p/--project that urfave/cli swallowed into the
			//    positionals when the user typed it AFTER the content.
			projectArg := c.String("project")
			posArgs := c.Args().Slice()
			if projectArg == "" {
				if rescued, rest := extractProjectFlag(posArgs); rescued != "" {
					projectArg = rescued
					posArgs = rest
				}
			}
			projectID, err := resolve.AutoResolveProject(projectArg, client)
			if err != nil {
				return err
			}

			// 2. Get content from positionals or piped stdin.
			content := strings.TrimSpace(strings.Join(posArgs, " "))
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
			// Gate on the SERVER's CURRENT encryption status (encstate) in
			// addition to the unlocked vault. Previously this path skipped the
			// encryption-enabled flag entirely, so it kept encrypting with the
			// old personal key after the user disabled encryption server-side.
			if encstate.ShouldEncryptPersonal(encstate.FetcherFor(client)) && crypto.IsVaultUnlocked() && !isOrgProject {
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
				if apierrors.IsEncryptionRequiredError(err) {
					projectName := projectNameFor(projects, projectID)
					return fmt.Errorf("%s", apierrors.EncryptionRequiredMessage(
						projectName,
						crypto.IsVaultUnlocked(),
						encstate.ShouldEncryptPersonal(encstate.FetcherFor(client)),
					))
				}
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

			total := len(memories)
			truncated := false
			if limit > 0 && total > limit {
				memories = memories[:limit]
				truncated = true
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
			direction := "↓ newest"
			if newestFirst {
				direction = "↑ newest"
			}
			subtitle := direction
			if truncated {
				subtitle = fmt.Sprintf("%d of %d · %s", len(memories), total, direction)
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

type memoryHygieneReport struct {
	ProjectID  string               `json:"project_id,omitempty"`
	Scanned    int                  `json:"scanned"`
	IssueCount int                  `json:"issue_count"`
	Counts     map[string]int       `json:"counts"`
	Issues     []memoryHygieneIssue `json:"issues"`
	DryRun     bool                 `json:"dry_run"`
}

type memoryHygieneIssue struct {
	Kind      string `json:"kind"`
	Severity  string `json:"severity"`
	MemoryID  string `json:"memory_id"`
	Type      string `json:"type"`
	AgeDays   int    `json:"age_days"`
	Reason    string `json:"reason"`
	Preview   string `json:"preview,omitempty"`
	RelatedID string `json:"related_id,omitempty"`
}

func memoryHygieneCmd() *cli.Command {
	return &cli.Command{
		Name:  "hygiene",
		Usage: "Dry-run memory hygiene report (stale, duplicate, low-value, unstructured runbooks)",
		Description: `Scans memories and reports hygiene risks without changing anything.

No archive, delete, merge, or consolidation is performed. Use this before manual
cleanup or before deciding whether to run a consolidation job.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project (name | short id | UUID). Optional: auto-detected when omitted.",
			},
			&cli.IntFlag{
				Name:  "max",
				Usage: "Maximum memories to scan",
				Value: 500,
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Print machine-readable JSON report",
			},
		},
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			projectID, err := resolve.AutoResolveProject(c.String("project"), client)
			if err != nil {
				return err
			}

			maxItems := c.Int("max")
			if maxItems <= 0 {
				maxItems = 500
			}
			memories, err := fetchMemoryHygienePageSet(client, projectID, maxItems)
			if err != nil {
				return err
			}
			report := analyzeMemoryHygiene(projectID, memories, time.Now())

			if c.Bool("json") {
				b, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			}

			printMemoryHygieneReport(report)
			return nil
		},
	}
}

func fetchMemoryHygienePageSet(client *api.Client, projectID string, maxItems int) ([]models.Memory, error) {
	pageSize := 100
	if maxItems < pageSize {
		pageSize = maxItems
	}
	var out []models.Memory
	for page := 1; len(out) < maxItems; page++ {
		items, hasMore, err := client.ListMemoriesPage(projectID, "", page, pageSize)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if !hasMore || len(items) == 0 {
			break
		}
		remaining := maxItems - len(out)
		if remaining <= 0 {
			break
		}
		if remaining < pageSize {
			pageSize = remaining
		}
	}
	if len(out) > maxItems {
		out = out[:maxItems]
	}
	return out, nil
}

func analyzeMemoryHygiene(projectID string, memories []models.Memory, now time.Time) memoryHygieneReport {
	report := memoryHygieneReport{
		ProjectID: projectID,
		Scanned:   len(memories),
		Counts:    map[string]int{},
		DryRun:    true,
	}

	byNormalizedContent := map[string]models.Memory{}
	for _, m := range memories {
		content := strings.TrimSpace(decryptMemoryForCLI(&m))
		ageDays := int(now.Sub(m.CreatedAt).Hours()/24 + 0.5)
		if ageDays < 0 {
			ageDays = 0
		}

		if stale, reason := memoryLooksStale(m, now); stale {
			report.addIssue(memoryHygieneIssue{
				Kind:     "stale",
				Severity: staleSeverity(m),
				MemoryID: m.ID.String(),
				Type:     m.Type,
				AgeDays:  ageDays,
				Reason:   reason,
				Preview:  display.SingleLine(content),
			})
		}

		if m.Type == "skill" && (strings.TrimSpace(valueOrEmpty(m.Trigger)) == "" || len(m.Steps) == 0) {
			report.addIssue(memoryHygieneIssue{
				Kind:     "skill_unstructured",
				Severity: "high",
				MemoryID: m.ID.String(),
				Type:     m.Type,
				AgeDays:  ageDays,
				Reason:   "skill memory is missing trigger or steps; before-action hooks cannot reliably surface it",
				Preview:  display.SingleLine(content),
			})
		}

		if m.Type != "skill" && looksLikeRunbookProse(content) {
			report.addIssue(memoryHygieneIssue{
				Kind:     "runbook_prose",
				Severity: "high",
				MemoryID: m.ID.String(),
				Type:     m.Type,
				AgeDays:  ageDays,
				Reason:   "procedural runbook appears stored as prose instead of type=skill with trigger/steps/validation",
				Preview:  display.SingleLine(content),
			})
		}

		if looksLowValueMemory(m, content, now) {
			report.addIssue(memoryHygieneIssue{
				Kind:     "thin_low_value",
				Severity: "low",
				MemoryID: m.ID.String(),
				Type:     m.Type,
				AgeDays:  ageDays,
				Reason:   "short general memory with no tags and no reuse; review before consolidating or archiving",
				Preview:  display.SingleLine(content),
			})
		}

		if norm := normalizeMemoryForHygiene(content); norm != "" {
			if prev, ok := byNormalizedContent[norm]; ok && prev.ID != m.ID {
				report.addIssue(memoryHygieneIssue{
					Kind:      "duplicate_exact",
					Severity:  "medium",
					MemoryID:  m.ID.String(),
					Type:      m.Type,
					AgeDays:   ageDays,
					Reason:    "normalized content exactly matches another memory; review and merge manually if redundant",
					Preview:   display.SingleLine(content),
					RelatedID: prev.ID.String(),
				})
			} else {
				byNormalizedContent[norm] = m
			}
		}
	}

	sort.SliceStable(report.Issues, func(i, j int) bool {
		if severityRank(report.Issues[i].Severity) != severityRank(report.Issues[j].Severity) {
			return severityRank(report.Issues[i].Severity) > severityRank(report.Issues[j].Severity)
		}
		if report.Issues[i].Kind != report.Issues[j].Kind {
			return report.Issues[i].Kind < report.Issues[j].Kind
		}
		return report.Issues[i].AgeDays > report.Issues[j].AgeDays
	})
	report.IssueCount = len(report.Issues)
	return report
}

func (r *memoryHygieneReport) addIssue(issue memoryHygieneIssue) {
	r.Issues = append(r.Issues, issue)
	r.Counts[issue.Kind]++
}

func memoryLooksStale(m models.Memory, now time.Time) (bool, string) {
	touch := m.CreatedAt
	if m.LastAccessedAt != nil && m.LastAccessedAt.After(touch) {
		touch = *m.LastAccessedAt
	} else if m.UpdatedAt.After(touch) {
		touch = m.UpdatedAt
	}
	window := 30 * 24 * time.Hour
	class := "transient"
	if isHygieneEvergreenType(m.Type) {
		window = 90 * 24 * time.Hour
		class = "evergreen"
	}
	idle := now.Sub(touch)
	if idle <= window {
		return false, ""
	}
	return true, fmt.Sprintf("untouched for %dd (>%dd %s freshness window); re-verify before relying on it",
		int(idle.Hours()/24+0.5), int(window.Hours()/24), class)
}

func staleSeverity(m models.Memory) string {
	if isHygieneEvergreenType(m.Type) || m.AccessCount >= 5 {
		return "medium"
	}
	return "low"
}

func isHygieneEvergreenType(t string) bool {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "decision", "reference", "skill":
		return true
	default:
		return false
	}
}

func looksLowValueMemory(m models.Memory, content string, now time.Time) bool {
	if strings.ToLower(strings.TrimSpace(m.Type)) != "general" {
		return false
	}
	if m.AccessCount > 0 || len(getTagsAsStrings(m.Tags)) > 0 {
		return false
	}
	if now.Sub(m.CreatedAt) < 7*24*time.Hour {
		return false
	}
	return len(strings.Fields(content)) < 8
}

func looksLikeRunbookProse(content string) bool {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "before:") || strings.Contains(lower, "runbook") {
		return true
	}
	return strings.Contains(lower, "steps") && (strings.Contains(lower, "build") || strings.Contains(lower, "deploy") || strings.Contains(lower, "test"))
}

func normalizeMemoryForHygiene(content string) string {
	words := strings.Fields(strings.ToLower(content))
	if len(words) < 6 {
		return ""
	}
	for i, w := range words {
		words[i] = strings.Trim(w, ".,;:!?()[]{}\"'`")
	}
	return strings.Join(words, " ")
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func severityRank(s string) int {
	switch strings.ToLower(s) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func printMemoryHygieneReport(report memoryHygieneReport) {
	fmt.Println(display.Header("Memory hygiene", fmt.Sprintf("dry-run · scanned %d · issues %d", report.Scanned, report.IssueCount)))
	fmt.Println(display.Dim.Render("No changes were made. Review these candidates manually before archive, merge, or skill conversion."))
	fmt.Println()

	if report.IssueCount == 0 {
		fmt.Println(display.Dim.Render("  no hygiene issues found"))
		return
	}

	kinds := make([]string, 0, len(report.Counts))
	for k := range report.Counts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		parts = append(parts, fmt.Sprintf("%s=%d", k, report.Counts[k]))
	}
	fmt.Println(display.Dim.Render("Counts: " + strings.Join(parts, ", ")))
	fmt.Println()

	cols := []display.Column{
		{Title: "SEV", Min: 6, Weight: 0},
		{Title: "KIND", Min: 18, Weight: 0},
		{Title: "ID", Min: 8, Weight: 0},
		{Title: "TYPE", Min: 12, Weight: 0},
		{Title: "AGE", Min: 8, Weight: 0},
		{Title: "REASON", Min: 30, Weight: 3},
	}
	rows := make([][]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		id := issue.MemoryID
		if len(id) > 8 {
			id = id[:8]
		}
		reason := issue.Reason
		if issue.RelatedID != "" {
			rel := issue.RelatedID
			if len(rel) > 8 {
				rel = rel[:8]
			}
			reason += " (related " + rel + ")"
		}
		rows = append(rows, []string{
			issue.Severity,
			issue.Kind,
			display.Dim.Render(id),
			display.TypeBadge(issue.Type),
			display.Dim.Render(fmt.Sprintf("%dd", issue.AgeDays)),
			reason,
		})
	}
	fmt.Println(display.NewResponsiveTable(cols, rows))
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

// projectNameFor returns the display name of the project with the given ID
// from an already-fetched list, or "" if not present. Used to build a
// friendlier ENCRYPTION_REQUIRED error without an extra round-trip.
func projectNameFor(projects []models.Project, projectID string) string {
	for _, p := range projects {
		if p.ID.String() == projectID {
			return p.Name
		}
	}
	return ""
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
