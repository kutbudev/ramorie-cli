// Package tui implements the `ramorie ui` interactive terminal navigator —
// a 3-pane (sidebar / list / detail) browser for tasks and memories.
package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/config"
)

// RunOptions carries the CLI-flag overrides for `ramorie ui`. Empty fields fall
// back to config then sensible defaults.
type RunOptions struct {
	Accent string // "auto"|"brand"|ANSI index|hex; "" => config/auto
	Icons  string // "nerd"|"unicode"|"auto"; "" => env/config/off
}

// Run starts the TUI. Blocks until the user quits.
//
// Accent and Nerd-Font are resolved and applied HERE, before tea.NewProgram —
// never at runtime: any terminal probe (e.g. termenv OSC color query) while
// bubbletea holds the tty in raw mode would race its input reader. ANSI-indexed
// accents need no probe at all; they follow the terminal's own palette.
func Run(opts RunOptions) error {
	cfg, _ := config.LoadConfig()

	accentSpec := firstNonEmpty(opts.Accent, configAccent(cfg))
	pal := resolveAccent(accentSpec)
	display.SetAccent(pal.Accent, pal.Bright)
	setNerdFont(resolveNerdFont(opts.Icons, cfg))

	client := api.NewClient()
	m := newRootModel(client)
	m.accentSpec = accentSpec
	m.accentGlamour = pal.Glamour

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func configAccent(cfg *config.Config) string {
	if cfg != nil {
		return cfg.Accent
	}
	return ""
}

// resolveNerdFont applies the precedence flag > env > config > off. A loaded
// font can't be detected, so the default is plain unicode (never tofu).
func resolveNerdFont(flag string, cfg *config.Config) bool {
	if v, ok := parseIconMode(flag); ok {
		return v
	}
	if v, ok := parseIconMode(os.Getenv("RAMORIE_ICONS")); ok {
		return v
	}
	if truthy(os.Getenv("NERD_FONT")) || truthy(os.Getenv("NERDFONT")) {
		return true
	}
	if cfg != nil && cfg.NerdFont != nil {
		return *cfg.NerdFont
	}
	return false
}

// parseIconMode maps an icon-mode string to (on, recognized).
func parseIconMode(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "nerd", "nerdfont", "on", "1", "true", "yes":
		return true, true
	case "unicode", "plain", "off", "0", "false", "no", "ascii":
		return false, true
	}
	return false, false // "auto"/""/unknown => not recognized, fall through
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
