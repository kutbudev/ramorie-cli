package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
)

// helpOverlay renders a centered modal grouping every binding into four
// sections (Navigation / Panes / Actions / Global), each a header plus a
// right-aligned key column and a dim description column. width/height are the
// full TUI dimensions; the modal is centered over a dimmed dotted backdrop.
func helpOverlay(_ keyMap, width, height int) string {
	type kv struct{ k, d string }
	type group struct {
		title string
		rows  []kv
	}

	nav := group{"Navigation", []kv{
		{"↑/k", "up"}, {"↓/j", "down"}, {"g/G", "top/bottom"},
		{"^u/^d", "page"}, {"/", "filter"},
	}}
	panes := group{"Panes", []kv{
		{"⇥", "next pane"}, {"S-⇥", "prev pane"}, {"←/h", "back"},
		{"→/l", "forward"}, {"↵", "open"}, {"esc", "back"},
	}}
	actions := group{"Actions", []kv{
		{"s", "recall (find)"}, {"n", "new task/memory"}, {"space", "complete task"},
		{"S", "start task"}, {"D", "delete"}, {"p", "project filter"},
		{"P", "all projects"}, {"r", "refresh"}, {"c", "copy"},
		{"t", "theme"}, {"A", "accent"}, {"I", "icons"},
	}}
	global := group{"Global", []kv{
		{"1-6", "jump category"}, {"?", "help"}, {"q", "quit"},
	}}

	renderGroup := func(g group) string {
		rows := make([]string, 0, len(g.rows)+1)
		rows = append(rows, helpSection.Render(g.title))
		for _, r := range g.rows {
			rows = append(rows, kvRow(r.k, r.d))
		}
		return lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	colLeft := lipgloss.JoinVertical(lipgloss.Left, renderGroup(nav), "", renderGroup(actions))
	colRight := lipgloss.JoinVertical(lipgloss.Left, renderGroup(panes), "", renderGroup(global))
	groups := lipgloss.JoinHorizontal(lipgloss.Top, colLeft, "      ", colRight)

	box := lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(display.ColorAccent)

	titled := lipgloss.JoinVertical(
		lipgloss.Left,
		display.PaneTitleOn.Render("Keybindings"),
		"",
		groups,
		"",
		lipgloss.NewStyle().Foreground(display.ColorMuted).Render("press ? or esc to close"),
	)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		box.Render(titled),
		lipgloss.WithWhitespaceChars("·"),
		lipgloss.WithWhitespaceForeground(display.ColorBorderInactive),
	)
}

var (
	helpSection = lipgloss.NewStyle().Foreground(display.ColorAccentBright).Bold(true)
	helpKey     = lipgloss.NewStyle().Foreground(display.ColorAccent).Bold(true).Width(7).Align(lipgloss.Right)
	helpDesc    = lipgloss.NewStyle().Foreground(display.ColorMuted).PaddingLeft(2)
)

func kvRow(k, desc string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, helpKey.Render(k), helpDesc.Render(desc))
}
