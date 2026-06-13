package resolve

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// AutoResolveProject behaves like ResolveProject, but when arg is empty it
// auto-detects the project from the environment so users rarely have to pass
// -p at all. Detection order (first hit wins):
//
//  1. current working directory name matches a project name
//  2. git remote (origin) repo name matches a project name
//  3. the user has exactly one project
//  4. the last project remembered in config (~/.ramorie/config.json)
//
// On any successful resolution the project is persisted as last-used so the
// next invocation — even from an unrelated directory — can fall back to it.
func AutoResolveProject(arg string, l ProjectLister) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg != "" {
		id, err := ResolveProject(arg, l)
		if err == nil {
			config.RememberLastProject(id)
		}
		return id, err
	}

	projects, err := l.ListProjects()
	if err != nil {
		return "", fmt.Errorf("could not fetch projects: %w", err)
	}
	if len(projects) == 0 {
		return "", errors.New("no projects found — create one with `ramorie project create <name>`")
	}

	// 1. Current working directory name.
	if id := matchByCWD(projects); id != "" {
		config.RememberLastProject(id)
		return id, nil
	}

	// 2. Git remote repo name.
	if repo := detectGitRemoteRepo(); repo != "" {
		repoNorm := normalizeForMatch(repo)
		for _, p := range projects {
			if normalizeForMatch(p.Name) == repoNorm {
				config.RememberLastProject(p.ID.String())
				return p.ID.String(), nil
			}
		}
	}

	// 3. Single project — unambiguous.
	if len(projects) == 1 {
		id := projects[0].ID.String()
		config.RememberLastProject(id)
		return id, nil
	}

	// 4. Last-used project from config (verify it still exists).
	if last := config.LoadLastProject(); last != "" {
		for _, p := range projects {
			if p.ID.String() == last {
				return last, nil
			}
		}
	}

	names := make([]string, 0, len(projects))
	for _, p := range projects {
		names = append(names, p.Name)
	}
	return "", fmt.Errorf(
		"could not auto-detect a project — pass one with -p <name>.\nAvailable projects: %s",
		strings.Join(names, ", "),
	)
}

// matchByCWD returns the project UUID whose normalized name equals a path
// segment of the current working directory, or "" if none match.
func matchByCWD(projects []models.Project) string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return ""
	}
	for _, segment := range strings.Split(cwd, string(os.PathSeparator)) {
		if segment == "" {
			continue
		}
		segNorm := normalizeForMatch(segment)
		if segNorm == "" {
			continue
		}
		for _, p := range projects {
			// Exact normalized match only — no prefix, so a short project
			// name ("ramorie") never matches a longer dir ("ramoriefrontend").
			if normalizeForMatch(p.Name) == segNorm {
				return p.ID.String()
			}
		}
	}
	return ""
}

// normalizeForMatch lowercases and strips spaces/hyphens/underscores/dots so
// "Ramorie CLI", "ramorie-cli" and "ramorie_cli" all compare equal. Mirrors
// the helper in internal/mcp/similarity.go (kept local to avoid coupling the
// CLI resolve package to the MCP package).
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

// detectGitRemoteRepo returns the repo name from origin's remote URL (e.g.
// "ramorie-cli" from "git@github.com:kutbudev/ramorie-cli.git"), or "" if not
// in a git repo / no origin. Bounded by a 500ms timeout.
func detectGitRemoteRepo() string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	if url == "" {
		return ""
	}
	url = strings.TrimSuffix(url, ".git")
	if i := strings.LastIndexAny(url, "/:"); i >= 0 {
		url = url[i+1:]
	}
	return url
}
