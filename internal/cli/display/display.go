// Package display provides shared CLI output formatters — colored badges,
// relative timestamps, tag rendering, and a simple row renderer — so
// `ramorie task list`, `ramorie memory list`, `ramorie recall` and their
// detail views all look consistent.
//
// All styling goes through lipgloss, which automatically downgrades to
// plain text when stdout isn't a TTY (piping to jq/grep is clean).
package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// ---- palette ---------------------------------------------------------------

var (
	// Accent colors. Match the TUI installer so the CLI feels cohesive.
	ColorAccent = lipgloss.Color("#8a87ff")
	ColorDim    = lipgloss.Color("#8e8e8e")
	ColorGood   = lipgloss.Color("#7ed491")
	ColorWarn   = lipgloss.Color("#e6b450")
	ColorError  = lipgloss.Color("#ff6e6e")
	ColorInfo   = lipgloss.Color("#5fb3ff")

	Title = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	Dim   = lipgloss.NewStyle().Foreground(ColorDim)
	Label = lipgloss.NewStyle().Foreground(ColorDim).Bold(true)
	Good  = lipgloss.NewStyle().Foreground(ColorGood)
	Warn  = lipgloss.NewStyle().Foreground(ColorWarn)
	Err   = lipgloss.NewStyle().Foreground(ColorError)
	Info  = lipgloss.NewStyle().Foreground(ColorInfo)
)

// ---- status + priority -----------------------------------------------------

// StatusIcon returns a one-rune glyph colored by task status. Designed to
// line up into a fixed-width column alongside other rows.
func StatusIcon(status string) string {
	switch strings.ToUpper(status) {
	case "IN_PROGRESS":
		return Warn.Render("◐")
	case "COMPLETED", "DONE":
		return Good.Render("✓")
	case "REVIEW":
		return Info.Render("⊙")
	case "CANCELED", "CANCELLED":
		return Dim.Render("✗")
	default: // TODO and anything unknown
		return Dim.Render("○")
	}
}

// StatusLabel renders the status name with color (for detail views).
func StatusLabel(status string) string {
	up := strings.ToUpper(status)
	switch up {
	case "IN_PROGRESS":
		return Warn.Render(up)
	case "COMPLETED", "DONE":
		return Good.Render(up)
	case "REVIEW":
		return Info.Render(up)
	case "CANCELED", "CANCELLED":
		return Dim.Render(up)
	default:
		return Dim.Render(up)
	}
}

// PriorityBadge renders a fixed-width colored priority indicator, e.g. "[H]"
// red, "[M]" yellow, "[L]" dim. Unknown priorities fall back to dim.
func PriorityBadge(priority string) string {
	up := strings.ToUpper(strings.TrimSpace(priority))
	switch up {
	case "H", "HIGH", "P1", "P0":
		return Err.Render("[H]")
	case "M", "MEDIUM", "P2":
		return Warn.Render("[M]")
	case "L", "LOW", "P3", "P4":
		return Dim.Render("[L]")
	}
	return Dim.Render("[?]")
}

// ---- memory type badge -----------------------------------------------------

// TypeBadge returns a short colored tag for a memory's type. Designed to
// fit a fixed width so lists align cleanly.
func TypeBadge(t string) string {
	const w = 10 // widest built-in type label is "reference" (9)
	label := strings.ToLower(strings.TrimSpace(t))
	if label == "" {
		label = "general"
	}
	padded := fmt.Sprintf("[%s]", label)
	// Pad to w chars for column alignment BEFORE coloring (ANSI escapes
	// would throw off width math otherwise).
	if len(padded) < w+2 {
		padded += strings.Repeat(" ", w+2-len(padded))
	}
	switch label {
	case "decision":
		return Info.Render(padded)
	case "bug_fix":
		return Err.Render(padded)
	case "preference":
		return Warn.Render(padded)
	case "pattern":
		return Good.Render(padded)
	case "reference":
		return Dim.Render(padded)
	case "skill":
		return Title.Render(padded)
	default: // general and unknowns
		return Dim.Render(padded)
	}
}

// ---- tags ------------------------------------------------------------------

// Tags renders up to `max` tags prefixed with #, with a "+N" overflow.
// Returns empty string when there are no tags (caller controls whether to
// render a placeholder or omit the column entirely).
func Tags(tags []string, max int) string {
	if len(tags) == 0 {
		return ""
	}
	if max <= 0 || max >= len(tags) {
		return Dim.Render("#" + strings.Join(tags, " #"))
	}
	head := tags[:max]
	rest := len(tags) - max
	return Dim.Render("#"+strings.Join(head, " #")) + Dim.Render(fmt.Sprintf(" +%d", rest))
}

// ---- time ------------------------------------------------------------------

// Relative returns a short human-readable age like "3m ago", "2h ago",
// "5d ago". Falls back to YYYY-MM-DD for anything older than 30 days so we
// don't end up with "47d ago" on stale memories.
func Relative(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < 0:
		return t.Format("2006-01-02")
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("2006-01-02")
}

// ---- text helpers ----------------------------------------------------------

// TerminalWidth returns the current terminal width, defaulting to 100 when
// stdout isn't a TTY (e.g. piped to a file). A minimum of 60 is enforced
// so narrow terminals still produce readable output.
func TerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 100
	}
	if w < 60 {
		return 60
	}
	return w
}

// Truncate trims a string to `n` runes, appending an ellipsis when it was
// actually truncated. Multi-byte / emoji safe.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	// Fast path: ASCII-length guard so we don't pay for rune-counting when
	// the string is clearly short.
	if len(s) <= n {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

// SingleLine collapses \n, \r, \t into single spaces so multi-line content
// renders as one row in list views. Preserves character count so Truncate
// math stays accurate.
func SingleLine(s string) string {
	r := strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ")
	out := r.Replace(s)
	// Collapse runs of spaces introduced by the replacements.
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}

// Rule renders a horizontal separator sized to the terminal, colored dim.
// Useful for detail views so sections feel grouped.
func Rule() string {
	return Dim.Render(strings.Repeat("─", TerminalWidth()))
}

// Header renders a titled banner row: "title · subtitle" with title in
// accent and subtitle dim. Use at the top of `list` commands.
func Header(title, subtitle string) string {
	if subtitle == "" {
		return Title.Render(title)
	}
	return Title.Render(title) + " " + Dim.Render("· "+subtitle)
}

// Sep renders a dim middle-dot separator — used inside single-row metadata
// like "2h ago · Ramorie Frontend · #react".
func Sep() string { return Dim.Render(" · ") }
