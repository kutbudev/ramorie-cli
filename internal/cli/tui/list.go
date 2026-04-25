package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// listItem wraps any entity for bubbles/list.
type listItem struct {
	id    string
	title string
	sub   string
	raw   interface{} // original entity (Task, Memory, ...)
}

func (i listItem) FilterValue() string { return i.title }
func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.sub }

// listModel is the middle pane — a scrollable list filtered by category.
type listModel struct {
	cat     Category
	list    list.Model
	width   int
	height  int
	focused bool
	loading bool
	errMsg  string
}

func newList(cat Category, width, height int) listModel {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.ShowDescription = false

	w := maxInt(width-4, 10)
	h := maxInt(height-4, 3)
	l := list.New(nil, delegate, w, h)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return listModel{
		cat:    cat,
		list:   l,
		width:  width,
		height: height,
	}
}

func (l *listModel) resize(width, height int) {
	l.width = width
	l.height = height
	l.list.SetSize(maxInt(width-4, 10), maxInt(height-4, 3))
}

// setItemsForCategory replaces list contents based on the category data.
func (l *listModel) setTasks(tasks []models.Task) {
	items := make([]list.Item, 0, len(tasks))
	for i := range tasks {
		t := tasks[i]
		title, _ := decryptTask(&t)
		shortID := t.ID.String()
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		formatted := fmt.Sprintf("%s %s  %s",
			display.PriorityBadge(t.Priority),
			display.Dim.Render(shortID),
			display.SingleLine(title),
		)
		items = append(items, listItem{
			id:    t.ID.String(),
			title: formatted,
			sub:   "",
			raw:   t,
		})
	}
	l.list.SetItems(items)
	l.loading = false
	l.errMsg = ""
}

func (l *listModel) setMemories(mems []models.Memory) {
	items := make([]list.Item, 0, len(mems))
	for i := range mems {
		m := mems[i]
		shortID := m.ID.String()
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		content := display.SingleLine(decryptMemoryContent(&m))
		formatted := fmt.Sprintf("%s %s  %s",
			display.TypeBadge(m.Type),
			display.Dim.Render(shortID),
			display.Truncate(content, 80),
		)
		items = append(items, listItem{
			id:    m.ID.String(),
			title: formatted,
			sub:   "",
			raw:   m,
		})
	}
	l.list.SetItems(items)
	l.loading = false
	l.errMsg = ""
}

// setPlaceholder fills the list with a single "coming soon" item.
func (l *listModel) setPlaceholder(label string) {
	items := []list.Item{listItem{
		id:    "placeholder",
		title: display.Dim.Render(fmt.Sprintf("(%s)", label)),
		raw:   nil,
	}}
	l.list.SetItems(items)
	l.loading = false
	l.errMsg = ""
}

// setError stores an error to render in place of the list.
func (l *listModel) setError(err error) {
	l.errMsg = err.Error()
	l.loading = false
	l.list.SetItems(nil)
}

// selected returns a pointer to the currently-highlighted listItem (or nil).
func (l listModel) selected() *listItem {
	it, ok := l.list.SelectedItem().(listItem)
	if !ok {
		return nil
	}
	return &it
}

// View renders the bordered list pane.
func (l listModel) View() string {
	border := lipgloss.NormalBorder()
	borderColor := lipgloss.Color("240")
	if l.focused {
		borderColor = display.ColorAccent
	}
	container := lipgloss.NewStyle().
		Width(maxInt(l.width-2, 1)).
		Height(maxInt(l.height-2, 1)).
		BorderStyle(border).
		BorderForeground(borderColor)

	var inner string
	switch {
	case l.loading:
		inner = display.Dim.Render("Loading…")
	case l.errMsg != "":
		inner = display.Err.Render(l.errMsg)
	default:
		inner = l.list.View()
	}
	return container.Render(inner)
}
