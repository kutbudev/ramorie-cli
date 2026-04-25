package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// helpOverlay renders a centered modal listing every key.Binding from km.
// width/height = full TUI dimensions; the modal is centered.
func helpOverlay(km keyMap, width, height int) string {
	bindings := []key.Binding{
		km.Up, km.Down, km.Left, km.Right,
		km.Enter, km.Back, km.Tab,
		km.Search, km.Project, km.Refresh,
		km.Yank, km.Theme, km.Help, km.Quit,
	}

	rows := make([]string, 0, len(bindings))
	for _, b := range bindings {
		h := b.Help()
		rows = append(rows, lipglossHelpRow(h.Key, h.Desc))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, rows...)

	box := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	titled := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Keybindings"),
		"",
		body,
		"",
		lipgloss.NewStyle().Faint(true).Render("press ? or esc to close"),
	)
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		box.Render(titled),
	)
}

func lipglossHelpRow(k, desc string) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Width(14)
	descStyle := lipgloss.NewStyle().Faint(true)
	return keyStyle.Render(k) + descStyle.Render(desc)
}
