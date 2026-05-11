package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/hooks"
	"github.com/kutbudev/ramorie-cli/internal/mcpinstall"
	"github.com/kutbudev/ramorie-cli/internal/protocol"
	"github.com/kutbudev/ramorie-cli/internal/rules"
	"github.com/urfave/cli/v2"
)

// doctorStatus is the result tier for a single check.
type doctorStatus int

const (
	doctorOK   doctorStatus = iota // ✓ everything wired correctly
	doctorWarn                     // ⚠ degraded but non-fatal (e.g. optional client missing)
	doctorFail                     // ✗ broken — exit code 1 territory
)

// doctorResult is one finding emitted by a check.
type doctorResult struct {
	Group   string       // logical group ("config", "vault", "mcp", "hooks", "rules")
	Status  doctorStatus // OK / Warn / Fail
	Message string       // human-readable line
	Remedy  string       // optional follow-up command
}

// symbol returns the printable status glyph for terminal output.
func (r doctorResult) symbol() string {
	switch r.Status {
	case doctorOK:
		return "✓"
	case doctorWarn:
		return "⚠"
	case doctorFail:
		return "✗"
	}
	return "·"
}

// NewDoctorCommand exposes `ramorie doctor` — an aggregate health check
// across auth, vault, MCP, hooks, and rules surfaces. Sub-arguments narrow
// the scope (e.g. `ramorie doctor hooks`); the default is `all`.
//
// Exit code is 0 when every check is OK or Warn; any Fail surfaces 1 so the
// command is CI-friendly.
func NewDoctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Diagnose Ramorie install — config, vault, MCP, hooks, rules",
		Description: "Runs a battery of health checks and reports ✓/⚠/✗ per surface.\n" +
			"   Filter with a sub-argument: config | vault | mcp | hooks | rules | all (default).",
		ArgsUsage: "[config|vault|mcp|hooks|rules|all]",
		Action: func(c *cli.Context) error {
			scope := strings.ToLower(strings.TrimSpace(c.Args().First()))
			if scope == "" {
				scope = "all"
			}
			results := runDoctorScope(scope)
			printDoctorResults(results)
			for _, r := range results {
				if r.Status == doctorFail {
					return cli.Exit("", 1)
				}
			}
			return nil
		},
	}
}

// runDoctorScope dispatches to per-group checks. Unknown scopes degrade to
// "all" with a friendly note so a typo doesn't silently skip everything.
func runDoctorScope(scope string) []doctorResult {
	switch scope {
	case "config":
		return checkConfig()
	case "vault":
		return checkVault()
	case "mcp":
		return checkMCP()
	case "hooks":
		return checkHooks()
	case "rules":
		return checkRules()
	case "all":
		return collectDoctorChecks()
	default:
		out := []doctorResult{{
			Group:   "doctor",
			Status:  doctorWarn,
			Message: fmt.Sprintf("unknown scope %q — running full check", scope),
		}}
		return append(out, collectDoctorChecks()...)
	}
}

// collectDoctorChecks runs every group in canonical order. The slice is
// concatenated so callers can iterate once and print/exit accordingly.
func collectDoctorChecks() []doctorResult {
	var out []doctorResult
	out = append(out, checkConfig()...)
	out = append(out, checkVault()...)
	out = append(out, checkMCP()...)
	out = append(out, checkHooks()...)
	out = append(out, checkRules()...)
	return out
}

// printDoctorResults emits a grouped, aligned summary. Remedies are printed
// on a second line when present so the operator gets a copy-pasteable fix.
func printDoctorResults(results []doctorResult) {
	currentGroup := ""
	for _, r := range results {
		if r.Group != currentGroup {
			fmt.Printf("\n[%s]\n", r.Group)
			currentGroup = r.Group
		}
		fmt.Printf("  %s %s\n", r.symbol(), r.Message)
		if r.Remedy != "" {
			fmt.Printf("       → %s\n", r.Remedy)
		}
	}
	fmt.Println()
}

// checkConfig verifies that the API key file is present and accepts a
// minimal authenticated request. We deliberately keep the request cheap —
// ListProjects with no orgID is sufficient.
//
// We split the failure cases so the operator sees the actual remedy:
//   - missing key       → "auth not configured" + run `ramorie setup`
//   - 401/403 response  → "auth invalid"        + run `ramorie setup login`
//   - any other error   → "backend unreachable" (Warn — config is fine)
//
// The old code collapsed all three into "Auth not configured" which was
// actively misleading when api.ramorie.com was unreachable but credentials
// were stored locally.
func checkConfig() []doctorResult {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		return []doctorResult{{
			Group:   "config",
			Status:  doctorFail,
			Message: "Auth not configured",
			Remedy:  "ramorie setup",
		}}
	}
	client := api.NewClient()
	client.APIKey = cfg.APIKey
	// Cheap probe — ListProjects without orgID hits a well-known endpoint.
	if _, err := client.ListProjects(); err != nil {
		if apierrors.IsAuthError(err) {
			return []doctorResult{{
				Group:   "config",
				Status:  doctorFail,
				Message: fmt.Sprintf("Auth invalid: %v", err),
				Remedy:  "ramorie setup login",
			}}
		}
		// Network / 5xx / unknown — credentials are fine, the backend isn't
		// reachable. Warn rather than Fail so offline doctor runs still pass.
		return []doctorResult{{
			Group:   "config",
			Status:  doctorWarn,
			Message: fmt.Sprintf("Backend unreachable: %v", err),
			Remedy:  "check network / api.ramorie.com status",
		}}
	}
	masked := cfg.APIKey
	if len(masked) > 12 {
		masked = masked[:8] + "…" + masked[len(masked)-4:]
	}
	return []doctorResult{{
		Group:   "config",
		Status:  doctorOK,
		Message: fmt.Sprintf("Auth configured (key %s)", masked),
	}}
}

// checkVault inspects the encryption config and reports whether the vault
// is unlocked. A locked vault with encryption enabled is a Warn, not a Fail
// — the user can still operate plaintext-only commands.
func checkVault() []doctorResult {
	cfg, err := config.LoadConfig()
	if err != nil || cfg.APIKey == "" {
		return []doctorResult{{
			Group:   "vault",
			Status:  doctorWarn,
			Message: "Skipped (not authenticated)",
		}}
	}
	if !cfg.EncryptionEnabled {
		return []doctorResult{{
			Group:   "vault",
			Status:  doctorOK,
			Message: "Encryption disabled — no vault required",
		}}
	}
	if crypto.IsVaultUnlocked() {
		return []doctorResult{{
			Group:   "vault",
			Status:  doctorOK,
			Message: "Vault unlocked",
		}}
	}
	return []doctorResult{{
		Group:   "vault",
		Status:  doctorWarn,
		Message: "Vault locked",
		Remedy:  "ramorie vault unlock",
	}}
}

// checkMCP confirms the stdio MCP server can be spawned. We invoke the
// binary with --version (cheapest possible probe); a 200ms timeout is
// generous for a process that should answer immediately.
func checkMCP() []doctorResult {
	results := []doctorResult{}
	binary := mcpinstall.BinaryPath()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "--version")
	if err := cmd.Run(); err != nil {
		results = append(results, doctorResult{
			Group:   "mcp",
			Status:  doctorFail,
			Message: fmt.Sprintf("ramorie binary not callable: %v", err),
			Remedy:  "reinstall via npm i -g ramorie",
		})
	} else {
		results = append(results, doctorResult{
			Group:   "mcp",
			Status:  doctorOK,
			Message: fmt.Sprintf("Binary reachable (%s)", binary),
		})
	}

	// Per-client install status — we don't fail on missing clients (Warn).
	for _, adapter := range mcpinstall.Registry() {
		det := adapter.Detect()
		if !det.Installed {
			continue
		}
		installed := false
		for _, s := range adapter.SupportedScopes() {
			if adapter.IsInstalled(s) {
				installed = true
				break
			}
		}
		if installed {
			results = append(results, doctorResult{
				Group:   "mcp",
				Status:  doctorOK,
				Message: fmt.Sprintf("%s: MCP entry present", adapter.Name()),
			})
		} else {
			results = append(results, doctorResult{
				Group:   "mcp",
				Status:  doctorWarn,
				Message: fmt.Sprintf("%s detected but Ramorie MCP not installed", adapter.Name()),
				Remedy:  "ramorie mcp install",
			})
		}
	}
	return results
}

// checkHooks reports whether protocol hooks are installed in every hook-
// capable client. A detected client without hooks is Warn (user might not
// want them); an undetected client is silently skipped.
func checkHooks() []doctorResult {
	installers := []hooks.Installer{
		hooks.NewClaudeCodeInstaller(),
		hooks.NewCodexInstaller(),
	}
	results := []doctorResult{}
	for _, inst := range installers {
		if !inst.Detect() {
			continue
		}
		entries, err := inst.Status()
		if err != nil {
			results = append(results, doctorResult{
				Group:   "hooks",
				Status:  doctorFail,
				Message: fmt.Sprintf("%s: status error: %v", inst.Name(), err),
			})
			continue
		}
		if len(entries) == 0 {
			results = append(results, doctorResult{
				Group:   "hooks",
				Status:  doctorWarn,
				Message: fmt.Sprintf("%s detected but no Ramorie hooks installed", inst.Name()),
				Remedy:  fmt.Sprintf("ramorie setup hooks install --client %s", inst.Name()),
			})
			continue
		}
		results = append(results, doctorResult{
			Group:   "hooks",
			Status:  doctorOK,
			Message: fmt.Sprintf("%s: %d hook(s) installed", inst.Name(), len(entries)),
		})
	}
	if len(results) == 0 {
		results = append(results, doctorResult{
			Group:   "hooks",
			Status:  doctorOK,
			Message: "No hook-capable clients detected — nothing to check",
		})
	}
	return results
}

// checkRules inspects rules-only clients. A version mismatch (installed
// version < protocol.Version) surfaces as a Warn with a re-install remedy.
func checkRules() []doctorResult {
	installers := []rules.Installer{
		rules.NewCursorInstaller(),
		rules.NewWindsurfInstaller(),
	}
	results := []doctorResult{}
	for _, inst := range installers {
		if !inst.Detect() {
			continue
		}
		installed, version, err := inst.Status()
		if err != nil {
			results = append(results, doctorResult{
				Group:   "rules",
				Status:  doctorFail,
				Message: fmt.Sprintf("%s: status error: %v", inst.Name(), err),
			})
			continue
		}
		if !installed {
			results = append(results, doctorResult{
				Group:   "rules",
				Status:  doctorWarn,
				Message: fmt.Sprintf("%s detected but rules-file not installed", inst.Name()),
				Remedy:  fmt.Sprintf("ramorie setup hooks install --client %s", inst.Name()),
			})
			continue
		}
		if version != "" && version != protocol.Version {
			results = append(results, doctorResult{
				Group:   "rules",
				Status:  doctorWarn,
				Message: fmt.Sprintf("%s rules outdated (v%s → v%s available)", inst.Name(), version, protocol.Version),
				Remedy:  "ramorie setup",
			})
			continue
		}
		results = append(results, doctorResult{
			Group:   "rules",
			Status:  doctorOK,
			Message: fmt.Sprintf("%s rules installed (v%s)", inst.Name(), version),
		})
	}
	if len(results) == 0 {
		results = append(results, doctorResult{
			Group:   "rules",
			Status:  doctorOK,
			Message: "No rules-only clients detected — nothing to check",
		})
	}
	return results
}

