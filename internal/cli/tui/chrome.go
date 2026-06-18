package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
)

// chrome.go holds the shared visual primitives for the TUI: the titled rounded
// pane used by all three columns, the context-aware footer key-hint bar, and a
// few small column helpers. Centralizing them keeps the sidebar / list / detail
// panes visually identical and makes the lazygit-style look a one-line call.

// titledPane renders a rounded box whose TOP border embeds `title` (left, inset
// one column) and an optional `count` (right). totalW/totalH are the OUTER
// dimensions of the pane. The body content region is exactly totalW-2 wide and
// totalH-2 tall; each pane sizes its inner widget to fill that region (the
// detail viewport at totalH-2, the list at totalH-3 to leave a footer row).
//
// lipgloss v1 has no Style.BorderTitle (that's v2), so the top border line is
// spliced together by hand. All width math goes through lipgloss.Width so the
// embedded ANSI never throws the column count off.
func titledPane(title, count, body string, totalW, totalH int, focused bool) string {
	b := lipgloss.RoundedBorder() // ╭ ─ ╮ │ ╰ ╯

	bc := display.ColorBorderInactive
	ts := display.PaneTitleOff
	if focused {
		bc = display.ColorBorderActive
		ts = display.PaneTitleOn
	}
	bs := lipgloss.NewStyle().Foreground(bc)
	innerW := maxInt(totalW-2, 1) // cells between the two vertical borders

	// --- top border line: ╭─ Title ─────────────── 12 ─╮
	// Count first so the title can be truncated against the remaining width,
	// guaranteeing the top line never overflows innerW on narrow panes.
	countSeg := ""
	if count != "" {
		countSeg = bs.Render(" ") + display.PaneCount.Render(count) + bs.Render(" ─")
	}
	titleSeg := ""
	if title != "" {
		avail := innerW - lipgloss.Width(countSeg) - 3 // "─ " prefix + " " suffix
		if avail < 1 {
			avail = 1
		}
		titleSeg = bs.Render("─ ") + ts.Render(display.Truncate(title, avail)) + bs.Render(" ")
	}
	dash := innerW - lipgloss.Width(titleSeg) - lipgloss.Width(countSeg)
	if dash < 0 {
		dash = 0
	}
	top := bs.Render(b.TopLeft) + titleSeg +
		bs.Render(strings.Repeat(b.Top, dash)) + countSeg + bs.Render(b.TopRight)

	// --- body box: left + right + bottom borders only (top is the manual
	// line above). Height = totalH-2 so the assembled pane is exactly totalH:
	// 1 (manual top) + (totalH-2) content rows + 1 (bottom border).
	bodyStyle := lipgloss.NewStyle().
		Border(b, false, true, true, true).
		BorderForeground(bc).
		Width(innerW).
		Height(maxInt(totalH-2, 1))

	return lipgloss.JoinVertical(lipgloss.Left, top, bodyStyle.Render(body))
}

// ---- footer ---------------------------------------------------------------

// keyHint renders one footer entry: an accent-bold key followed by a dim
// description, e.g. "↵ open". Both halves carry the footer background so the
// bar stays continuous when segments are concatenated.
func keyHint(k, desc string) string {
	return display.FooterKey.Render(k) + display.FooterDsc.Render(" "+desc)
}

// keyHintAccent renders a "what to press next" hint whose key uses a semantic
// accent (warn/info) instead of the resting accent — the lazygit rule that
// reserves color for the single most-likely next action.
func keyHintAccent(k, desc string, fg lipgloss.TerminalColor) string {
	key := lipgloss.NewStyle().
		Foreground(fg).
		Background(display.ColorFooterBg).
		Bold(true).
		Render(k)
	return key + display.FooterDsc.Render(" "+desc)
}

// renderFooter assembles the full-width footer bar: an optional transient
// status (left-most), a left segment (focus · scope · counts), and a
// right-aligned, context-aware list of key hints. Every piece carries the
// footer background so the fill is unbroken; hints are trimmed from the end
// with an ellipsis when the terminal is too narrow.
func renderFooter(width int, left string, hints []string, status string) string {
	if width < 1 {
		width = 1
	}
	sep := display.FooterSep.Render(" │ ")
	ell := display.FooterDsc.Render("…")

	fits := func(hs []string) bool {
		reserve := 4
		if status != "" {
			reserve += lipgloss.Width(status) + 2
		}
		return lipgloss.Width(left)+lipgloss.Width(strings.Join(hs, sep))+reserve <= width
	}

	// Trim to fit, but PROTECT the final two hints (the global "? help / q
	// quit" pair) — drop from the middle so the quit hint never disappears.
	if !fits(hints) {
		tail := 2
		if len(hints) < tail {
			tail = len(hints)
		}
		head := hints[:len(hints)-tail]
		keep := hints[len(hints)-tail:]
		for {
			trial := append(append(append([]string{}, head...), ell), keep...)
			if fits(trial) || len(head) == 0 {
				hints = trial
				break
			}
			head = head[:len(head)-1]
		}
	}

	right := strings.Join(hints, sep)
	lw := lipgloss.Width(left) + lipgloss.Width(right)
	if status != "" {
		lw += lipgloss.Width(status) + 2
	}
	gap := width - lw - 2
	if gap < 1 {
		gap = 1
	}

	var sb strings.Builder
	sb.WriteString(display.FooterBar.Render(" "))
	if status != "" {
		sb.WriteString(status)
		sb.WriteString(display.FooterBar.Render("  "))
	}
	sb.WriteString(left)
	sb.WriteString(display.FooterBar.Render(strings.Repeat(" ", gap)))
	sb.WriteString(right)
	sb.WriteString(display.FooterBar.Render(" "))

	// Final clamp guarantees a full-width bg and trims any overflow.
	return display.FooterBar.Width(width).MaxWidth(width).Render(sb.String())
}

// ---- small column helpers -------------------------------------------------

// categoryIcon returns a plain-unicode glyph for a sidebar category. No
// nerd-font dependency — every rune is in the box-drawing / geometric blocks.
func categoryIcon(c Category) string {
	switch c {
	case CatTasks:
		return "◎"
	case CatMemories:
		return "✱"
	case CatProjects:
		return "▤"
	case CatOrganizations:
		return "⬢"
	case CatActivity:
		return "◷"
	case CatKanban:
		return "▦"
	case CatProfile:
		return "⊙"
	}
	return "·"
}

// priorityBadgeParts returns the plain badge text plus the style to color it
// with. Split (vs display.PriorityBadge which returns pre-colored ANSI) so the
// list delegate can drop the color on a selected row — the selection bar's
// background would otherwise be severed by the badge's ANSI reset.
func priorityBadgeParts(p string) (string, lipgloss.Style) {
	switch strings.ToUpper(strings.TrimSpace(p)) {
	case "H", "HIGH", "P1", "P0":
		return "[H]", display.Err
	case "M", "MEDIUM", "P2":
		return "[M]", display.Warn
	case "L", "LOW", "P3", "P4":
		return "[L]", display.Dim
	}
	return "[?]", display.Dim
}

// typeBadgeParts is the memory/activity-type analogue of priorityBadgeParts.
func typeBadgeParts(t string) (string, lipgloss.Style) {
	label := strings.ToLower(strings.TrimSpace(t))
	if label == "" {
		label = "general"
	}
	txt := "[" + label + "]"
	switch label {
	case "decision":
		return txt, display.Info
	case "bug_fix":
		return txt, display.Err
	case "preference":
		return txt, display.Warn
	case "pattern":
		return txt, display.Good
	case "reference":
		return txt, display.Dim
	case "skill":
		return txt, display.Title
	}
	return txt, display.Dim
}
