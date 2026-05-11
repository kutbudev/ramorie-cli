package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewSkillCommand creates the 'skill' command group. PR6 (v6.8.0)
// ships with a single action — `use` — that mirrors `pack use`: pipe
// the rendered Claude Code-format markdown into stdout so agents,
// Claude Code's `Read`-style tools, or `pbcopy` can grab it without
// further parsing.
//
//	ramorie skill use <id-or-name>          # markdown body to stdout
//	ramorie skill use <id-or-name> --json   # full response JSON
//
// PR7 will extend this group with sync (`skill push`, `skill pull`)
// and PR8 with AI generation (`skill generate`). Subcommand list
// stays minimal until those land so help output isn't cluttered with
// unimplemented verbs.
func NewSkillCommand() *cli.Command {
	return &cli.Command{
		Name:    "skill",
		Aliases: []string{"skills"},
		Usage:   "Render and manage procedural skills",
		Subcommands: []*cli.Command{
			skillUseCmd(),
			skillUploadCmd(),
			skillSyncCmd(),
			skillPullCmd(),
			skillDiffCmd(),
			skillGenerateCmd(),
		},
	}
}

// skillUseCmd renders a skill via GET /memories/{id}/skill-render and
// prints the body to stdout (pipe-friendly). With --json, prints the
// full envelope (skill + body + source + _meta) instead.
//
//	ramorie skill use deploy-prod
//	ramorie skill use 7f3c1b3e-... --json
func skillUseCmd() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Aliases:   []string{"render", "load"},
		Usage:     "Render a skill (Claude Code-format markdown) to stdout",
		ArgsUsage: "[skill-id-or-name]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Output full response JSON instead of bare markdown body"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("skill id or name required")
			}
			ident := c.Args().First()

			client := api.NewClient()
			resp, err := client.LoadSkill(ident)
			if err != nil {
				return fmt.Errorf("load skill: %w", err)
			}

			// Encrypted body guard — server returns the cipher untouched
			// (zero-knowledge). Don't print it as if it were markdown;
			// fail loud with a non-zero exit so scripts piping into a
			// file or `pbcopy` don't silently capture ciphertext. --json
			// still works (full envelope inspection is useful even when
			// encrypted).
			if encVal, ok := resp.Meta["encrypted"]; ok && !c.Bool("json") {
				if enc, _ := encVal.(bool); enc {
					fmt.Fprintln(c.App.ErrWriter, "⚠ skill body is encrypted — vault unlock required (`ramorie unlock` then retry)")
					return cli.Exit("encrypted skill body cannot be rendered", 2)
				}
			}

			if c.Bool("json") {
				out, err := json.MarshalIndent(resp, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal skill response: %w", err)
				}
				fmt.Println(string(out))
				return nil
			}

			// Bare body to stdout — pipeable into pbcopy, an agent, or
			// `>` redirect for `.claude/skills/<name>/SKILL.md`. Stderr
			// gets a one-line byline so the human sees what just rendered
			// without polluting the markdown.
			fmt.Print(resp.Body)
			fmt.Println()
			fmt.Fprintf(c.App.ErrWriter, "─── skill %q v%s, %d steps ───\n",
				resp.Skill.Name, resp.Skill.Version, resp.Skill.StepsCount)
			return nil
		},
	}
}

// ============================================================================
// PR7 — Filesystem sync subcommands
//
// The CLI views the filesystem as a mirror of Ramorie. Conventions:
//   * Default sync dir is ~/.claude/skills/ (user-level Claude Code agent
//     skills); --dir overrides for project-level .claude/skills/.
//   * Each skill is a single SKILL.md file inside its own folder named
//     after the frontmatter `name` (matches the Claude Code layout).
//   * `upload` is single-file; `sync` walks the directory; `pull` writes
//     Ramorie → disk; `diff` shows hash mismatches without writing.
// ============================================================================

// defaultSkillsDir is the canonical user-level location for skill files,
// matching Claude Code's discovery path. Returned as ~/.claude/skills/.
func defaultSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

// computeLocalSkillHash mirrors the backend's computeSkillHash exactly:
// CRLF→LF normalisation + trailing-whitespace trim before SHA-256. Keep
// this in lock-step with handlers/skill_handler_sync.go::computeSkillHash
// or `skill diff` will report false positives.
func computeLocalSkillHash(markdown string) string {
	normalized := strings.TrimRight(strings.ReplaceAll(markdown, "\r\n", "\n"), " \t\n")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// extractFrontmatterName parses the leading YAML frontmatter and returns
// the `name` field. Best-effort; returns "" when the block is malformed
// or absent so the caller can fall back to the filename.
func extractFrontmatterName(md string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	md = strings.TrimSpace(md)
	if !strings.HasPrefix(md, "---") {
		return ""
	}
	rest := md[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	fm := rest[:end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

// findSkillMarkdownFiles walks `root` and returns absolute paths of
// candidate skill files. Heuristic: any `*.md` file living under root,
// excluding hidden dirs (`.git`, `.cache`, …). Preserves discovery order
// (sorted by path) so output is deterministic.
func findSkillMarkdownFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base != "." && base != ".." && strings.HasPrefix(base, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// skillUploadCmd pushes a single markdown file to Ramorie. Reads the
// file, sends as-is — the server is the canonical parser.
//
//	ramorie skill upload ~/.claude/skills/foo/SKILL.md
//	ramorie skill upload ./bar.md --overwrite
func skillUploadCmd() *cli.Command {
	return &cli.Command{
		Name:      "upload",
		Usage:     "Upload a single skill markdown file to Ramorie",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "overwrite", Usage: "Replace existing skill with the same name when content differs"},
			&cli.BoolFlag{Name: "json", Usage: "Output JSON response"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("path to markdown file required")
			}
			path := c.Args().First()
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			absPath, _ := filepath.Abs(path)

			client := api.NewClient()
			resp, err := client.UploadSkill(string(data), absPath, c.Bool("overwrite"))
			if err != nil {
				return err
			}
			if c.Bool("json") {
				out, _ := json.MarshalIndent(resp, "", "  ")
				fmt.Println(string(out))
				return nil
			}
			fmt.Printf("%s %s (%s)\n", resp.Action, resp.ID, path)
			return nil
		},
	}
}

// skillSyncCmd bulk-uploads every skill markdown under --dir. Useful for
// onboarding (`ramorie skill sync` once after `ramorie setup`) and for
// re-syncing after manual file edits.
//
//	ramorie skill sync
//	ramorie skill sync --dir .claude/skills/ --overwrite
func skillSyncCmd() *cli.Command {
	return &cli.Command{
		Name:  "sync",
		Usage: "Bulk-push all skill markdown files under a directory to Ramorie",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "dir", Usage: "Skills directory (default: ~/.claude/skills/)"},
			&cli.BoolFlag{Name: "overwrite", Usage: "Force overwrite on name collisions"},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir == "" {
				d, err := defaultSkillsDir()
				if err != nil {
					return err
				}
				dir = d
			}
			if _, err := os.Stat(dir); err != nil {
				return fmt.Errorf("skills dir %q: %w", dir, err)
			}

			files, err := findSkillMarkdownFiles(dir)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Fprintf(c.App.ErrWriter, "no .md files found under %s\n", dir)
				return nil
			}

			client := api.NewClient()
			var created, updated, noop, failed int
			for _, f := range files {
				data, err := os.ReadFile(f)
				if err != nil {
					fmt.Fprintf(c.App.ErrWriter, "skip %s: %v\n", f, err)
					failed++
					continue
				}
				resp, err := client.UploadSkill(string(data), f, c.Bool("overwrite"))
				if err != nil {
					fmt.Fprintf(c.App.ErrWriter, "fail %s: %v\n", f, err)
					failed++
					continue
				}
				switch resp.Action {
				case "created":
					created++
				case "updated":
					updated++
				case "noop":
					noop++
				}
				fmt.Printf("%-8s %s\n", resp.Action, f)
			}
			fmt.Fprintf(c.App.ErrWriter, "\nSynced %d file(s): %d new, %d updated, %d unchanged, %d failed\n",
				len(files), created, updated, noop, failed)
			if failed > 0 {
				return cli.Exit("", 1)
			}
			return nil
		},
	}
}

// skillPullCmd writes every Ramorie skill to disk under --dir. Files are
// laid out as <dir>/<name>/SKILL.md to match Claude Code's discovery
// layout. Skills without a frontmatter `name` are written under their
// UUID to keep filenames stable.
//
//	ramorie skill pull
//	ramorie skill pull --dir ./.claude/skills/
func skillPullCmd() *cli.Command {
	return &cli.Command{
		Name:  "pull",
		Usage: "Write Ramorie skills to the local filesystem (Ramorie → disk)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "dir", Usage: "Target directory (default: ~/.claude/skills/)"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Print planned writes without touching the filesystem"},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir == "" {
				d, err := defaultSkillsDir()
				if err != nil {
					return err
				}
				dir = d
			}
			if !c.Bool("dry-run") {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("create %s: %w", dir, err)
				}
			}

			client := api.NewClient()
			items, err := client.ListSkillsSyncState()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Fprintln(c.App.ErrWriter, "no skills found")
				return nil
			}

			var written, skipped int
			for _, it := range items {
				name := it.Name
				if name == "" {
					name = it.ID
				}
				safeName := sanitizeSkillFolderName(name)
				targetDir := filepath.Join(dir, safeName)
				targetFile := filepath.Join(targetDir, "SKILL.md")

				md, err := client.PullSkillMarkdown(it.ID)
				if err != nil {
					fmt.Fprintf(c.App.ErrWriter, "skip %s: %v\n", name, err)
					skipped++
					continue
				}
				if c.Bool("dry-run") {
					fmt.Printf("would write %s (%d bytes)\n", targetFile, len(md))
					continue
				}
				if err := os.MkdirAll(targetDir, 0o755); err != nil {
					fmt.Fprintf(c.App.ErrWriter, "mkdir %s: %v\n", targetDir, err)
					skipped++
					continue
				}
				if err := os.WriteFile(targetFile, []byte(md), 0o644); err != nil {
					fmt.Fprintf(c.App.ErrWriter, "write %s: %v\n", targetFile, err)
					skipped++
					continue
				}
				written++
				fmt.Printf("write    %s\n", targetFile)
			}
			fmt.Fprintf(c.App.ErrWriter, "\nPulled %d skill(s): %d written, %d skipped\n",
				len(items), written, skipped)
			return nil
		},
	}
}

// skillDiffCmd compares local files (under --dir) against the server's
// sync-state hashes. Pure read; no writes on either side. Output lines:
//
//	+ skill-name       (local only — not yet uploaded)
//	- skill-name       (remote only — not yet pulled)
//	~ skill-name       (hash mismatch — content differs)
//	= skill-name       (in sync) [only with --verbose]
func skillDiffCmd() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "Compare local skill files with Ramorie",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "dir", Usage: "Skills directory (default: ~/.claude/skills/)"},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Also show in-sync rows"},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir == "" {
				d, err := defaultSkillsDir()
				if err != nil {
					return err
				}
				dir = d
			}

			// Load local skills indexed by frontmatter name (falling back
			// to filename when frontmatter is absent).
			localByName := map[string]string{} // name -> hash
			localPathByName := map[string]string{}
			if _, err := os.Stat(dir); err == nil {
				files, err := findSkillMarkdownFiles(dir)
				if err != nil {
					return err
				}
				for _, f := range files {
					data, err := os.ReadFile(f)
					if err != nil {
						continue
					}
					name := extractFrontmatterName(string(data))
					if name == "" {
						name = strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
					}
					localByName[name] = computeLocalSkillHash(string(data))
					localPathByName[name] = f
				}
			}

			client := api.NewClient()
			remote, err := client.ListSkillsSyncState()
			if err != nil {
				return err
			}
			remoteByName := map[string]SkillSyncStateShadow{}
			for _, r := range remote {
				remoteByName[r.Name] = SkillSyncStateShadow{
					ID:       r.ID,
					Name:     r.Name,
					SyncHash: r.SyncHash,
				}
			}

			// Union of names, deterministic output.
			seen := map[string]struct{}{}
			var names []string
			for n := range localByName {
				if _, ok := seen[n]; !ok {
					names = append(names, n)
					seen[n] = struct{}{}
				}
			}
			for n := range remoteByName {
				if _, ok := seen[n]; !ok {
					names = append(names, n)
					seen[n] = struct{}{}
				}
			}
			// Stable order — sort alphabetically.
			for i := 0; i < len(names); i++ {
				for j := i + 1; j < len(names); j++ {
					if names[j] < names[i] {
						names[i], names[j] = names[j], names[i]
					}
				}
			}

			var added, removed, modified, same int
			for _, n := range names {
				local, hasLocal := localByName[n]
				rem, hasRemote := remoteByName[n]
				switch {
				case hasLocal && !hasRemote:
					added++
					fmt.Printf("+ %s\n", n)
				case !hasLocal && hasRemote:
					removed++
					fmt.Printf("- %s\n", n)
				case hasLocal && hasRemote:
					rh := ""
					if rem.SyncHash != nil {
						rh = *rem.SyncHash
					}
					if rh == local {
						same++
						if c.Bool("verbose") {
							fmt.Printf("= %s\n", n)
						}
					} else {
						modified++
						fmt.Printf("~ %s\n", n)
					}
				}
			}
			fmt.Fprintf(c.App.ErrWriter, "\n%d local-only, %d remote-only, %d modified, %d in sync\n",
				added, removed, modified, same)
			return nil
		},
	}
}

// SkillSyncStateShadow is a tiny local mirror of api.SkillSyncStateItem
// — keeps the diff command from leaking the full struct shape into
// closures.
type SkillSyncStateShadow struct {
	ID       string
	Name     string
	SyncHash *string
}

// sanitizeSkillFolderName makes a skill name safe for use as a directory
// component. Replaces path separators and other reserved characters with
// "-". Mirrors what Claude Code's file discovery does on disk.
func sanitizeSkillFolderName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "skill"
	}
	repl := func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
			return '-'
		}
		return r
	}
	return strings.Map(repl, s)
}

// skillGenerateCmd (PR8) wraps POST /v1/memories/generate-skill with the
// smart-context strategies the SkillWizard uses on the web. Default
// strategy is "relevant" — same HyDE+rerank pipeline as `ramorie find` —
// so a one-liner like `ramorie skill generate "deploy production"` does
// the right thing without picking memories by hand.
//
//	# Default — relevant (semantic + rerank)
//	ramorie skill generate "deploy production"
//
//	# Importance × usage × recency ranking
//	ramorie skill generate "team conventions" --strategy top
//
//	# Most recently updated memories
//	ramorie skill generate "what changed lately" --strategy recent
//
//	# Caller-supplied IDs (chips from the UI)
//	ramorie skill generate "deploy production" --strategy manual \
//	    --memory-ids id1,id2 --task-ids id3
func skillGenerateCmd() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Aliases:   []string{"gen"},
		Usage:     "Generate a skill markdown with smart context (memories + tasks)",
		ArgsUsage: "<goal>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "strategy", Value: "relevant", Usage: "Context strategy: relevant | top | recent | manual"},
			&cli.StringFlag{Name: "project", Usage: "Limit context to this project (UUID)"},
			&cli.StringFlag{Name: "model", Usage: "Override Gemini model (e.g. gemini-2.5-flash)"},
			&cli.IntFlag{Name: "max-memories", Value: 20, Usage: "Cap on memories included in context"},
			&cli.IntFlag{Name: "max-tasks", Value: 20, Usage: "Cap on tasks included in context"},
			&cli.StringFlag{Name: "memory-ids", Usage: "Comma-separated memory UUIDs (strategy=manual)"},
			&cli.StringFlag{Name: "task-ids", Usage: "Comma-separated task UUIDs (strategy=manual)"},
			&cli.BoolFlag{Name: "json", Usage: "Output full response JSON instead of the rendered markdown"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("goal argument required (quote multi-word goals)")
			}
			goal := strings.TrimSpace(strings.Join(c.Args().Slice(), " "))
			if goal == "" {
				return fmt.Errorf("goal must not be empty")
			}

			strategy := c.String("strategy")
			switch strategy {
			case "relevant", "top", "recent", "manual":
			default:
				return fmt.Errorf("invalid strategy %q (want: relevant|top|recent|manual)", strategy)
			}

			// splitIDs validates each comma-separated ID is a UUID. Failing
			// fast on the client side is much friendlier than letting the
			// backend reject the request with a generic 500. See PR8 Stuart S7.
			splitIDs := func(flag, raw string) ([]string, error) {
				if strings.TrimSpace(raw) == "" {
					return nil, nil
				}
				parts := strings.Split(raw, ",")
				out := make([]string, 0, len(parts))
				for _, p := range parts {
					s := strings.TrimSpace(p)
					if s == "" {
						continue
					}
					if _, err := uuid.Parse(s); err != nil {
						return nil, fmt.Errorf("invalid %s %q: not a UUID", flag, s)
					}
					out = append(out, s)
				}
				return out, nil
			}
			memIDs, err := splitIDs("--memory-ids", c.String("memory-ids"))
			if err != nil {
				return err
			}
			taskIDs, err := splitIDs("--task-ids", c.String("task-ids"))
			if err != nil {
				return err
			}

			// Friendly guard: manual without IDs is almost certainly user
			// error — without it the API would still 200 with an empty
			// context and the model would invent the skill from the goal
			// alone. Fail loud so users notice.
			if strategy == "manual" && len(memIDs) == 0 && len(taskIDs) == 0 {
				return fmt.Errorf("strategy=manual requires --memory-ids and/or --task-ids")
			}

			client := api.NewClient()
			resp, err := client.GenerateSkillSmart(api.GenerateSkillOptions{
				Goal:            goal,
				ProjectID:       c.String("project"),
				Model:           c.String("model"),
				Strategy:        strategy,
				ManualMemoryIDs: memIDs,
				ManualTaskIDs:   taskIDs,
				MaxMemories:     c.Int("max-memories"),
				MaxTasks:        c.Int("max-tasks"),
			})
			if err != nil {
				return fmt.Errorf("generate skill: %w", err)
			}

			if c.Bool("json") {
				out, err := json.MarshalIndent(resp, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal response: %w", err)
				}
				fmt.Println(string(out))
				return nil
			}

			// Reassemble the full markdown (frontmatter + body) so the
			// output is `> file.md`-ready. Mirrors the `skill use` output
			// contract — same pipeline assumptions.
			fmt.Println("---")
			fmt.Printf("name: %s\n", resp.Skill.Frontmatter.Name)
			fmt.Printf("description: %s\n", resp.Skill.Frontmatter.Description)
			fmt.Printf("when_to_use: %s\n", resp.Skill.Frontmatter.WhenToUse)
			if len(resp.Skill.Frontmatter.Tags) > 0 {
				fmt.Printf("tags: [%s]\n", strings.Join(resp.Skill.Frontmatter.Tags, ", "))
			}
			fmt.Println("---")
			fmt.Println()
			fmt.Println(resp.Skill.Body)
			// Byline mirrors `skill use` — stderr so it doesn't pollute
			// the markdown when redirected.
			fmt.Fprintf(c.App.ErrWriter, "─── generated via %s, %d items, %dms ───\n",
				resp.AIModel, resp.ContextItemsUsed, resp.LatencyMs)
			return nil
		},
	}
}
