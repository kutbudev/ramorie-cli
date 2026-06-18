package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
)

// overlay.go renders the modal overlays that drive the interactive actions:
// a single-line text prompt (create / recall) and a yes-no confirm (delete).
// State lives on rootModel; these helpers only paint.

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayPrompt
	overlayConfirm
)

// promptIntent is what a submitted prompt value should do.
type promptIntent int

const (
	promptCreateTask promptIntent = iota
	promptCreateMemory
	promptRecall
)

// modalBox centers a titled rounded box over a dimmed dotted backdrop.
func modalBox(title, body, footer string, width, height int) string {
	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		display.PaneTitleOn.Render(title),
		"",
		body,
		"",
		lipgloss.NewStyle().Foreground(display.ColorMuted).Render(footer),
	)
	box := lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(display.ColorAccent).
		Width(maxInt(minInt(width-8, 64), 24)).
		Render(inner)
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceChars("·"),
		lipgloss.WithWhitespaceForeground(display.ColorBorderInactive),
	)
}

func renderPromptOverlay(title, inputView string, width, height int) string {
	return modalBox(title, inputView, "↵ submit · esc cancel", width, height)
}

func renderConfirmOverlay(question string, width, height int) string {
	body := lipgloss.NewStyle().Foreground(display.ColorError).Render(question)
	return modalBox("Confirm", body, "y confirm · n / esc cancel", width, height)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
