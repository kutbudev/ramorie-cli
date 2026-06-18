package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
)

// Category is a sidebar entry — selects what the middle pane lists.
type Category int

const (
	CatTasks Category = iota
	CatMemories
	CatProjects
	CatOrganizations
	CatActivity
	CatKanban
	CatProfile
)

// Label returns the human-readable name for a category.
func (c Category) Label() string {
	switch c {
	case CatTasks:
		return "Tasks"
	case CatMemories:
		return "Memories"
	case CatProjects:
		return "Projects"
	case CatOrganizations:
		return "Organizations"
	case CatActivity:
		return "Activity"
	case CatKanban:
		return "Kanban"
	case CatProfile:
		return "Profile"
	}
	return ""
}

// CatKanban is intentionally omitted — the TUI's kanban view required a
// project filter and offered nothing the Tasks tab doesn't already do.
// The CatKanban constant + handlers stay in the package for now (cheap to
// keep, easy to re-enable) but no longer appear in the sidebar.
var allCategories = []Category{
	CatTasks, CatMemories, CatProjects, CatOrganizations,
	CatActivity, CatProfile,
}

// sidebarModel holds focus on a category list. Pure rendering — input dispatch
// happens in rootModel which calls movePrev/moveNext.
type sidebarModel struct {
	cursor  int
	width   int
	height  int
	focused bool
}

func newSidebar() sidebarModel {
	return sidebarModel{cursor: 0}
}

func (s *sidebarModel) movePrev() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *sidebarModel) moveNext() {
	if s.cursor < len(allCategories)-1 {
		s.cursor++
	}
}

func (s *sidebarModel) selectIndex(i int) bool {
	if i < 0 || i >= len(allCategories) {
		return false
	}
	changed := s.cursor != i
	s.cursor = i
	return changed
}

func categoryIndex(cat Category) int {
	for i, c := range allCategories {
		if c == cat {
			return i
		}
	}
	return -1
}

func (s sidebarModel) selected() Category {
	return allCategories[s.cursor]
}

func (s sidebarModel) View() string {
	innerW := maxInt(s.width-2, 1)
	lines := make([]string, 0, len(allCategories)+1)
	for i, cat := range allCategories {
		lines = append(lines, s.rowView(i, cat, innerW))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return titledPane("ramorie", "", body, s.width, s.height, s.focused)
}

// rowView renders one category row: a number gutter, a plain-unicode icon, and
// the label. The selected row paints a full-width selection bar when the pane
// is focused, or shows bold-only when it isn't (the lazygit two-state rule).
func (s sidebarModel) rowView(i int, cat Category, innerW int) string {
	// Plain text first so the selection bar's background is never severed by an
	// embedded ANSI reset.
	plain := fmt.Sprintf(" %d %s %s", i+1, categoryIcon(cat), cat.Label())

	if i == s.cursor {
		if s.focused {
			return display.SelRowStyle.Width(innerW).MaxWidth(innerW).Render(plain)
		}
		return lipgloss.NewStyle().Bold(true).Width(innerW).MaxWidth(innerW).Render(plain)
	}

	// Resting row: dim number gutter, accent icon, muted label.
	num := display.Dim.Render(fmt.Sprintf("%d", i+1))
	icon := lipgloss.NewStyle().Foreground(display.ColorAccent).Render(categoryIcon(cat))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(cat.Label())
	line := " " + num + " " + icon + " " + label
	return lipgloss.NewStyle().Width(innerW).MaxWidth(innerW).Render(line)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
