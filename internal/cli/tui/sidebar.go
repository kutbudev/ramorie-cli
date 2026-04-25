package tui

import (
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

func (s sidebarModel) selected() Category {
	return allCategories[s.cursor]
}

func (s sidebarModel) View() string {
	var lines []string
	headerStyle := display.Title.PaddingLeft(1).PaddingRight(1)
	lines = append(lines, headerStyle.Render("ramorie"))
	lines = append(lines, "")
	for i, cat := range allCategories {
		prefix := "  "
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == s.cursor {
			prefix = "▶ "
			if s.focused {
				style = style.Foreground(display.ColorAccent).Bold(true)
			} else {
				style = style.Foreground(display.ColorAccent)
			}
		} else {
			style = style.Foreground(lipgloss.Color("245"))
		}
		lines = append(lines, style.Render(prefix+cat.Label()))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)

	border := lipgloss.NormalBorder()
	borderColor := lipgloss.Color("240")
	if s.focused {
		borderColor = display.ColorAccent
	}
	container := lipgloss.NewStyle().
		Width(maxInt(s.width-2, 1)).
		Height(maxInt(s.height-2, 1)).
		BorderStyle(border).
		BorderForeground(borderColor)
	return container.Render(body)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
