package tui

import "github.com/charmbracelet/lipgloss"

// glyphs.go provides an optional Nerd Font icon layer. A loaded terminal font
// can't be queried at runtime, so Nerd Font mode is strictly OPT-IN (flag/env/
// config). When off, every icon falls back to the plain-unicode glyph the TUI
// has always used, so the default experience never shows tofu.
//
// All Nerd Font codepoints below are classic FontAwesome-4 (nf-fa, U+F000
// range) — the most stable subset, never removed across Nerd Font v2 to v3.
// They are built from hex codepoints (pure-ASCII source) rather than embedded
// PUA bytes.
//
// WIDTH RULE: PUA glyphs report width 1 to go-runewidth but render ~2 cells.
// nf() bakes a trailing space so the STRING measures 2 columns (glyph 1 +
// space 1), keeping the titledPane border splice and footer math honest.
// Never measure a bare PUA rune — use iconW().

// nerdFont is the process-wide icon mode, set once at TUI startup (mirrors the
// display.SetAccent pattern). Single-process TUI, so a package global is safe.
var nerdFont bool

func setNerdFont(on bool) { nerdFont = on }

// nf builds a column-padded Nerd Font glyph (glyph + trailing space = 2 cols).
func nf(cp rune) string { return string(cp) + " " }

// nfb builds a bare Nerd Font glyph (no padding) for embedding in markdown.
func nfb(cp rune) string { return string(cp) }

type glyph struct{ nerd, plain string }

var glyphs = map[string]glyph{
	// Sidebar categories.
	"cat_tasks":    {nf(0xf0ae), "◎"}, // tasks
	"cat_memories": {nf(0xf0eb), "✱"}, // lightbulb-o
	"cat_projects": {nf(0xf07b), "▤"}, // folder
	"cat_orgs":     {nf(0xf1ad), "⬢"}, // building
	"cat_activity": {nf(0xf1da), "◷"}, // history
	"cat_kanban":   {nf(0xf009), "▦"}, // th-large
	"cat_profile":  {nf(0xf007), "⊙"}, // user
	"cat_search":   {nf(0xf002), "⌕"}, // search

	// Memory / activity types (plain fallback is the bracketed label).
	"type_decision":   {nf(0xf0e3), "[decision]"},   // gavel
	"type_bug_fix":    {nf(0xf188), "[bug_fix]"},    // bug
	"type_preference": {nf(0xf013), "[preference]"}, // cog
	"type_pattern":    {nf(0xf0e8), "[pattern]"},    // sitemap
	"type_reference":  {nf(0xf02d), "[reference]"},  // book
	"type_skill":      {nf(0xf005), "[skill]"},      // star
	"type_general":    {nf(0xf111), "[general]"},    // circle

	// Task priority.
	"prio_high":    {nf(0xf062), "[H]"}, // arrow-up
	"prio_med":     {nf(0xf068), "[M]"}, // minus
	"prio_low":     {nf(0xf063), "[L]"}, // arrow-down
	"prio_unknown": {nf(0xf128), "[?]"}, // question

	// Task status (also used inside markdown — bare glyph, no trailing space).
	"st_todo":        {nfb(0xf10c), "○"}, // circle-o
	"st_in_progress": {nfb(0xf192), "◐"}, // dot-circle-o
	"st_completed":   {nfb(0xf058), "✓"}, // check-circle
	"st_blocked":     {nfb(0xf05e), "✗"}, // ban
	"st_review":      {nfb(0xf06e), "⊙"}, // eye
}

// icon returns the active glyph for name (nerd variant when enabled and
// present, else the plain fallback). Unknown names yield "".
func icon(name string) string {
	g, ok := glyphs[name]
	if !ok {
		return ""
	}
	if nerdFont && g.nerd != "" {
		return g.nerd
	}
	return g.plain
}

// iconW is the rendered column width of icon(name): a fixed 2 for nerd glyphs
// (glyph + baked trailing space), else the real width of the plain fallback.
func iconW(name string) int {
	g, ok := glyphs[name]
	if !ok {
		return 0
	}
	if nerdFont && g.nerd != "" {
		return 2
	}
	return lipgloss.Width(g.plain)
}
