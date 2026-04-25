// Package tui implements the `ramorie ui` interactive terminal navigator —
// a 3-pane (sidebar / list / detail) browser for tasks and memories.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kutbudev/ramorie-cli/internal/api"
)

// Run starts the TUI. Blocks until the user quits.
func Run() error {
	client := api.NewClient()
	m := newRootModel(client)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
