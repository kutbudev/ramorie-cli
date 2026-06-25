package commands

import (
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/hooks"
	"github.com/kutbudev/ramorie-cli/internal/protocol"
	"github.com/kutbudev/ramorie-cli/internal/rules"
	"github.com/urfave/cli/v2"
)

// NewSetupHooksCommand exposes `ramorie setup hooks` — install, uninstall,
// or status the persistent memory protocol across hook-capable clients
// (Claude Code, Codex) and rules-only clients (Cursor, Windsurf).
//
// We treat both surfaces uniformly so the operator never needs to know which
// editor uses hooks vs. rules — the `--client` flag accepts either kind.
func NewSetupHooksCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup-hooks",
		Usage: "Install/uninstall Ramorie protocol hooks + rules across editors",
		Description: "Manages the Persistent Memory Protocol installs across:\n" +
			"   - Hook-capable clients (Claude Code, Codex) → JSON hooks\n" +
			"   - Rules-only clients (Cursor, Windsurf) → managed markdown blocks\n\n" +
			"   Use `--client all` (default) to operate on every detected target.",
		Subcommands: []*cli.Command{
			{
				Name:  "install",
				Usage: "Install protocol hooks + rules into selected clients",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "client",
						Aliases: []string{"c"},
						Usage:   "target client: claude-code|codex|cursor|windsurf|all",
						Value:   "all",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be written without modifying files",
					},
				},
				Action: hooksInstallAction,
			},
			{
				Name:  "uninstall",
				Usage: "Remove protocol hooks + rules from selected clients",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "client",
						Aliases: []string{"c"},
						Usage:   "target client: claude-code|codex|cursor|windsurf|all",
						Value:   "all",
					},
				},
				Action: hooksUninstallAction,
			},
			{
				Name:   "status",
				Usage:  "Report which clients have protocol installed and at what version",
				Action: hooksStatusAction,
			},
		},
	}
}

// allHookInstallers returns every hook-capable client integration. Keep this
// list in sync with installHooksForAllDetected in setup.go.
func allHookInstallers() []hooks.Installer {
	return []hooks.Installer{
		hooks.NewClaudeCodeInstaller(),
		hooks.NewCodexInstaller(),
	}
}

// allRulesInstallers returns every rules-only client integration.
func allRulesInstallers() []rules.Installer {
	return []rules.Installer{
		rules.NewCursorInstaller(),
		rules.NewWindsurfInstaller(),
	}
}

// filterHookInstallers keeps installers whose name matches `client` (or
// returns the full list when client="all").
func filterHookInstallers(client string) []hooks.Installer {
	client = strings.ToLower(strings.TrimSpace(client))
	if client == "" || client == "all" {
		return allHookInstallers()
	}
	out := []hooks.Installer{}
	for _, inst := range allHookInstallers() {
		if inst.Name() == client {
			out = append(out, inst)
		}
	}
	return out
}

// filterRulesInstallers is the rules-side twin of filterHookInstallers.
func filterRulesInstallers(client string) []rules.Installer {
	client = strings.ToLower(strings.TrimSpace(client))
	if client == "" || client == "all" {
		return allRulesInstallers()
	}
	out := []rules.Installer{}
	for _, inst := range allRulesInstallers() {
		if inst.Name() == client {
			out = append(out, inst)
		}
	}
	return out
}

// hooksInstallAction wires the install path. When --client matches neither
// a hook nor a rules client we surface a helpful list of valid names rather
// than silently no-op.
func hooksInstallAction(c *cli.Context) error {
	client := c.String("client")
	dryRun := c.Bool("dry-run")

	hookInsts := filterHookInstallers(client)
	rulesInsts := filterRulesInstallers(client)

	if len(hookInsts) == 0 && len(rulesInsts) == 0 {
		return fmt.Errorf("unknown client %q — valid: claude-code, codex, cursor, windsurf, all", client)
	}

	fmt.Println()
	fmt.Println("🪝 Protocol install")
	fmt.Println("━━━━━━━━━━━━━━━━━━━")

	// Hook clients — JSON merge into settings.json / hooks.json.
	for _, inst := range hookInsts {
		if !inst.Detect() {
			fmt.Printf("  · %s not detected — skipping\n", inst.Name())
			continue
		}
		if dryRun {
			fmt.Printf("  · %s — would write %d hook(s) to %s\n",
				inst.Name(), len(hooks.DefaultEntries()), inst.SettingsPath())
			continue
		}
		if err := inst.Install(hooks.DefaultEntries()); err != nil {
			warn(fmt.Sprintf("%s: %v", inst.Name(), err))
			continue
		}
		fmt.Printf("  ✓ %s — %d hook(s) installed → %s\n",
			inst.Name(), len(hooks.DefaultEntries()), inst.SettingsPath())
	}

	// Rules clients — managed markdown block.
	for _, inst := range rulesInsts {
		if !inst.Detect() {
			fmt.Printf("  · %s not detected — skipping\n", inst.Name())
			continue
		}
		path := inst.RulesPath()
		if dryRun {
			fmt.Printf("  · %s — would write rules-file to %s\n", inst.Name(), path)
			continue
		}
		if err := inst.Install(protocol.EnglishSessionStartText); err != nil {
			warn(fmt.Sprintf("%s: %v", inst.Name(), err))
			continue
		}
		fmt.Printf("  ✓ %s — protocol v%s installed → %s\n",
			inst.Name(), protocol.Version, path)
	}

	fmt.Println()
	if dryRun {
		fmt.Println("Dry-run complete. Re-run without --dry-run to apply.")
	} else {
		fmt.Println("Done. Restart your editor(s) to pick up the new config.")
	}
	return nil
}

// hooksUninstallAction strips Ramorie-managed hooks/rules from the targeted
// clients. Foreign config in the same file is preserved by the underlying
// installer implementations.
func hooksUninstallAction(c *cli.Context) error {
	client := c.String("client")

	hookInsts := filterHookInstallers(client)
	rulesInsts := filterRulesInstallers(client)

	if len(hookInsts) == 0 && len(rulesInsts) == 0 {
		return fmt.Errorf("unknown client %q — valid: claude-code, codex, cursor, windsurf, all", client)
	}

	fmt.Println()
	fmt.Println("🧹 Protocol uninstall")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━")

	// Collect every Ramorie ID for sweep-style hook removal.
	ids := make([]string, 0, len(hooks.DefaultEntries()))
	for _, e := range hooks.DefaultEntries() {
		ids = append(ids, e.ID)
	}

	for _, inst := range hookInsts {
		if !inst.Detect() {
			continue
		}
		if err := inst.Uninstall(ids); err != nil {
			warn(fmt.Sprintf("%s: %v", inst.Name(), err))
			continue
		}
		fmt.Printf("  ✓ %s — hooks removed from %s\n", inst.Name(), inst.SettingsPath())
	}
	for _, inst := range rulesInsts {
		if !inst.Detect() {
			continue
		}
		if err := inst.Uninstall(); err != nil {
			warn(fmt.Sprintf("%s: %v", inst.Name(), err))
			continue
		}
		fmt.Printf("  ✓ %s — rules-file removed from %s\n", inst.Name(), inst.RulesPath())
	}

	fmt.Println()
	return nil
}

// hooksStatusAction prints a per-client snapshot. Output is grouped into a
// hooks section and a rules section so the operator can scan quickly.
func hooksStatusAction(c *cli.Context) error {
	fmt.Println()
	fmt.Println("🩺 Protocol status")
	fmt.Println("━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n[hooks]")
	for _, inst := range allHookInstallers() {
		if !inst.Detect() {
			fmt.Printf("  · %s: not detected\n", inst.Name())
			continue
		}
		entries, err := inst.Status()
		if err != nil {
			fmt.Printf("  ✗ %s: status error: %v\n", inst.Name(), err)
			continue
		}
		if len(entries) == 0 {
			fmt.Printf("  ⚠ %s: detected but no Ramorie hooks installed\n", inst.Name())
			continue
		}
		missing, stale := hooks.DiffEntries(hooks.DefaultEntries(), entries)
		marker := "✓"
		suffix := "installed"
		if len(missing) > 0 || len(stale) > 0 {
			marker = "⚠"
			suffix = fmt.Sprintf("outdated/incomplete (missing=%d stale=%d)", len(missing), len(stale))
		}
		fmt.Printf("  %s %s: %d hook(s) %s → %s\n",
			marker, inst.Name(), len(entries), suffix, inst.SettingsPath())
		for _, e := range entries {
			matcher := ""
			if e.Matcher != "" {
				matcher = " (matcher: " + e.Matcher + ")"
			}
			fmt.Printf("       · %s%s — id=%s\n", e.Event, matcher, e.ID)
		}
		if len(missing) > 0 || len(stale) > 0 {
			fmt.Printf("       → run `ramorie setup-hooks install --client %s` to refresh\n", inst.Name())
		}
	}

	fmt.Println("\n[rules]")
	for _, inst := range allRulesInstallers() {
		if !inst.Detect() {
			fmt.Printf("  · %s: not detected\n", inst.Name())
			continue
		}
		installed, version, err := inst.Status()
		if err != nil {
			fmt.Printf("  ✗ %s: status error: %v\n", inst.Name(), err)
			continue
		}
		if !installed {
			fmt.Printf("  ⚠ %s: detected but rules-file not installed\n", inst.Name())
			continue
		}
		marker := "✓"
		stale := version != "" && version != protocol.Version
		if stale {
			marker = "⚠"
		}
		fmt.Printf("  %s %s: rules v%s installed → %s\n",
			marker, inst.Name(), version, inst.RulesPath())
		if stale {
			fmt.Printf("       → outdated (current v%s); run `ramorie setup` to refresh\n", protocol.Version)
		}
	}

	fmt.Println()
	return nil
}
