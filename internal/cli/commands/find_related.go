package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewFindRelatedCommand exposes `ramorie find-related --file X` for use by the
// Claude Code hook (or humans who want a quick "what do I know about this
// file?" query). Writes a compact human-readable summary to stdout; exits 0
// even when no results so the hook pipeline doesn't block.
func NewFindRelatedCommand() *cli.Command {
	return &cli.Command{
		Name:  "find-related",
		Usage: "Print memories/decisions related to a file path (used by Claude Code hook)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Required: true, Usage: "Absolute file path"},
			&cli.IntFlag{Name: "budget", Value: 500, Usage: "Token budget for the response"},
			&cli.IntFlag{Name: "limit", Value: 2, Usage: "Max items"},
			&cli.StringFlag{Name: "project", Usage: "Optional project name/UUID override"},
		},
		Action: findRelatedAction,
	}
}

func findRelatedAction(c *cli.Context) error {
	filePath := c.String("file")
	if filePath == "" {
		return nil
	}

	client := api.NewClient()

	// Build a compact search term from filename + path hints.
	term := deriveSearchTerm(filePath)

	// Project hint: prefer explicit flag, fall back to cwd's basename so the
	// backend resolver can match by name.
	projectHint := c.String("project")
	if projectHint == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectHint = filepath.Base(cwd)
		}
	}

	resp, err := client.FindMemories(api.FindMemoriesOptions{
		Term:             term,
		ProjectHint:      projectHint,
		Limit:            c.Int("limit"),
		BudgetTokens:     c.Int("budget"),
		IncludeDecisions: true,
	})
	if err != nil || resp == nil || len(resp.Items) == 0 {
		return nil // silent on no-hit / errors
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Ramorie context for %s:\n", filepath.Base(filePath))
	for _, item := range resp.Items {
		tag := item.Type
		if item.Kind != "" {
			tag = fmt.Sprintf("%s/%s", item.Type, item.Kind)
		}
		fmt.Fprintf(&b, "- [%s] %s — %s\n", tag, item.Title, item.Preview)
	}
	fmt.Fprintf(&b, "(ranking: %s, %d token est)", resp.Meta.RankingMode, resp.Meta.TokensEst)

	fmt.Println(b.String())
	return nil
}

// deriveSearchTerm produces a useful query string from a file path. Uses the
// filename stem plus the two closest directory names, which usually captures
// the module/feature context.
func deriveSearchTerm(p string) string {
	base := filepath.Base(p)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	dir := filepath.Dir(p)
	parts := strings.Split(dir, string(filepath.Separator))

	// Take last 2 directory segments (ignoring empty / anchor).
	tail := []string{}
	for i := len(parts) - 1; i >= 0 && len(tail) < 2; i-- {
		if parts[i] == "" || parts[i] == "." {
			continue
		}
		tail = append([]string{parts[i]}, tail...)
	}

	joined := append([]string{stem}, tail...)
	return strings.Join(joined, " ")
}

// ---- hook cooldown (prevents edit→surface→edit feedback loops) --------------

const hookCooldownDir = ".cache/ramorie-hook"

func hookStateFile(filePath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	sum := sha1.Sum([]byte(filePath))
	name := hex.EncodeToString(sum[:])
	return filepath.Join(home, hookCooldownDir, name)
}

func wasRecentlyProcessed(filePath string) bool {
	f := hookStateFile(filePath)
	if f == "" {
		return false
	}
	info, err := os.Stat(f)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < 30*time.Second
}

func markProcessed(filePath string) {
	f := hookStateFile(filePath)
	if f == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(f), 0o755)
	_ = os.WriteFile(f, []byte("1"), 0o644)
}
