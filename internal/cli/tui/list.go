package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/api"
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

// navFrame is one entry in the drill-down stack. The list pane always
// displays the items of the top-most frame.
type navFrame struct {
	cat      Category
	label    string // breadcrumb segment ("Orgs", "Ramorie", "Backend")
	parentID string // org id when filtering projects, project id when filtering tasks, etc.
	items    []list.Item
}

// listModel is the middle pane — a scrollable list filtered by category, with
// an internal breadcrumb stack supporting one-frame-deep drill-down (Orgs →
// projects-of-org, Projects → tasks-of-project).
type listModel struct {
	cat     Category
	list    list.Model
	width   int
	height  int
	focused bool
	loading bool
	errMsg  string
	stack   []navFrame

	// Pagination state for the active top frame (tasks/memories only).
	page        int  // current page (1-based) loaded so far
	pageSize    int  // page size used in fetches; default 100
	hasMore     bool // true when the last fetch returned a full page
	loadingMore bool // true while a follow-up page is being fetched
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
	innerH := maxInt(height-4, 3)
	if len(l.stack) > 1 {
		// Reserve one row for the breadcrumb.
		innerH = maxInt(innerH-1, 1)
	}
	l.list.SetSize(maxInt(width-4, 10), innerH)
}

// resetStack clears the breadcrumb stack and seeds it with a single frame for
// the new active category. Also resets pagination state.
func (l *listModel) resetStack(cat Category, label string) {
	l.cat = cat
	l.stack = []navFrame{{cat: cat, label: label}}
	l.list.SetItems(nil)
	l.errMsg = ""
	l.page = 0
	l.pageSize = 100
	l.hasMore = false
	l.loadingMore = false
}

// pushFrame adds a new drill-down frame and clears the visible list (caller
// is expected to dispatch a loader that will populate it via setXxx).
func (l *listModel) pushFrame(cat Category, label, parentID string) {
	l.stack = append(l.stack, navFrame{cat: cat, label: label, parentID: parentID})
	l.cat = cat
	l.list.SetItems(nil)
	l.loading = true
	l.errMsg = ""
	// Resize to account for the breadcrumb row.
	l.resize(l.width, l.height)
}

// popFrame removes the top frame; returns true if a pop happened.
func (l *listModel) popFrame() bool {
	if len(l.stack) <= 1 {
		return false
	}
	l.stack = l.stack[:len(l.stack)-1]
	top := l.stack[len(l.stack)-1]
	l.cat = top.cat
	l.list.SetItems(top.items)
	l.loading = false
	l.errMsg = ""
	l.resize(l.width, l.height)
	return true
}

// depth returns the current breadcrumb depth (1 = top-level for the cat).
func (l listModel) depth() int { return len(l.stack) }

// topFrame returns the active frame (or zero-value if stack is empty).
func (l listModel) topFrame() navFrame {
	if len(l.stack) == 0 {
		return navFrame{}
	}
	return l.stack[len(l.stack)-1]
}

// taskToItem renders a single task as a list cell.
func taskToItem(t models.Task) list.Item {
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
	return listItem{
		id:    t.ID.String(),
		title: formatted,
		raw:   t,
	}
}

// memoryToItem renders a single memory as a list cell.
func memoryToItem(m models.Memory) list.Item {
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
	return listItem{
		id:    m.ID.String(),
		title: formatted,
		raw:   m,
	}
}

// setTasks replaces the list with the first page of tasks. page+hasMore drive
// the infinite-scroll state machine.
func (l *listModel) setTasks(tasks []models.Task, page int, hasMore bool) {
	items := make([]list.Item, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, taskToItem(t))
	}
	l.page = page
	l.hasMore = hasMore
	l.loadingMore = false
	l.applyItems(items)
}

// appendTasks adds another page of tasks to the existing list.
func (l *listModel) appendTasks(tasks []models.Task, page int, hasMore bool) {
	cur := l.list.Items()
	for _, t := range tasks {
		cur = append(cur, taskToItem(t))
	}
	l.page = page
	l.hasMore = hasMore
	l.loadingMore = false
	l.applyItems(cur)
}

// setMemories replaces the list with the first page of memories.
func (l *listModel) setMemories(mems []models.Memory, page int, hasMore bool) {
	items := make([]list.Item, 0, len(mems))
	for _, m := range mems {
		items = append(items, memoryToItem(m))
	}
	l.page = page
	l.hasMore = hasMore
	l.loadingMore = false
	l.applyItems(items)
}

// appendMemories adds another page of memories.
func (l *listModel) appendMemories(mems []models.Memory, page int, hasMore bool) {
	cur := l.list.Items()
	for _, m := range mems {
		cur = append(cur, memoryToItem(m))
	}
	l.page = page
	l.hasMore = hasMore
	l.loadingMore = false
	l.applyItems(cur)
}

// shouldFetchMore returns true when the cursor is within the bottom guard band
// AND another page is available AND no fetch is in flight.
func (l listModel) shouldFetchMore() bool {
	if !l.hasMore || l.loadingMore {
		return false
	}
	idx := l.list.Index()
	total := len(l.list.Items())
	// Trigger when within the last 3 visible items.
	return total > 0 && idx >= total-3
}

// setProjects renders projects as one-line cells and stores them in the top
// frame.
func (l *listModel) setProjects(projects []models.Project) {
	items := make([]list.Item, 0, len(projects))
	for i := range projects {
		p := projects[i]
		shortID := p.ID.String()
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		desc := display.SingleLine(p.Description)
		formatted := fmt.Sprintf("%s %s  %s",
			display.Dim.Render("[project]"),
			display.Dim.Render(shortID),
			display.Truncate(p.Name, 32),
		)
		if desc != "" {
			formatted += "  " + display.Dim.Render(display.Truncate(desc, 60))
		}
		items = append(items, listItem{
			id:    p.ID.String(),
			title: formatted,
			raw:   p,
		})
	}
	l.applyItems(items)
}

// setOrgs renders organizations as one-line cells.
func (l *listModel) setOrgs(orgs []api.Organization) {
	items := make([]list.Item, 0, len(orgs))
	for i := range orgs {
		o := orgs[i]
		shortID := o.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		formatted := fmt.Sprintf("%s %s  %s",
			display.Dim.Render("[org]"),
			display.Dim.Render(shortID),
			display.Truncate(o.Name, 40),
		)
		items = append(items, listItem{
			id:    o.ID,
			title: formatted,
			raw:   o,
		})
	}
	l.applyItems(items)
}

// setActivity renders a flat activity feed.
func (l *listModel) setActivity(events []models.ActivityItem) {
	items := make([]list.Item, 0, len(events))
	for i := range events {
		e := events[i]
		ts := e.Timestamp.Format("2006-01-02 15:04")
		summary := display.SingleLine(e.Summary)
		formatted := fmt.Sprintf("%s %s  %s",
			display.TypeBadge(e.EntityType),
			display.Dim.Render(ts),
			display.Truncate(summary, 80),
		)
		items = append(items, listItem{
			id:    e.EntityID.String(),
			title: formatted,
			raw:   e,
		})
	}
	l.applyItems(items)
}

// setKanbanSummary renders three compact rows showing bucket counts. The
// detail pane carries the actual three-column board.
func (l *listModel) setKanbanSummary(todo, inProgress, completed int) {
	items := []list.Item{
		listItem{id: "kanban-todo", title: fmt.Sprintf("%s  %d tasks",
			display.Dim.Render("📝 TODO        "), todo)},
		listItem{id: "kanban-ip", title: fmt.Sprintf("%s  %d tasks",
			display.Warn.Render("🚀 IN PROGRESS "), inProgress)},
		listItem{id: "kanban-done", title: fmt.Sprintf("%s  %d tasks",
			display.Good.Render("✅ COMPLETED   "), completed)},
	}
	l.applyItems(items)
}

// setProfileMode clears the list pane and shows a hint that the right pane
// holds the profile content.
func (l *listModel) setProfileMode() {
	items := []list.Item{listItem{
		id:    "profile",
		title: display.Dim.Render("(profile shown on the right →)"),
	}}
	l.applyItems(items)
}

// applyItems writes items into the bubbles list AND mirrors them into the top
// frame so popFrame can restore them.
func (l *listModel) applyItems(items []list.Item) {
	l.list.SetItems(items)
	l.loading = false
	l.errMsg = ""
	if len(l.stack) == 0 {
		l.stack = []navFrame{{cat: l.cat}}
	}
	l.stack[len(l.stack)-1].items = items
}

// setPlaceholder fills the list with a single "coming soon" item.
func (l *listModel) setPlaceholder(label string) {
	items := []list.Item{listItem{
		id:    "placeholder",
		title: display.Dim.Render(fmt.Sprintf("(%s)", label)),
		raw:   nil,
	}}
	l.applyItems(items)
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

// breadcrumb returns the rendered breadcrumb string for the current stack.
// Returns "" when depth <= 1 (no drill-down active).
func (l listModel) breadcrumb() string {
	if len(l.stack) <= 1 {
		return ""
	}
	parts := make([]string, 0, len(l.stack))
	for _, f := range l.stack {
		label := f.label
		if label == "" {
			label = f.cat.Label()
		}
		parts = append(parts, label)
	}
	return display.Dim.Render(strings.Join(parts, " › "))
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
		if crumb := l.breadcrumb(); crumb != "" {
			inner = crumb + "\n" + inner
		}
		if foot := l.paginationFooter(); foot != "" {
			inner += "\n" + foot
		}
	}
	return container.Render(inner)
}

// paginationFooter shows "n loaded · page N · more →" or "all loaded" when
// the cat supports pagination. Empty string for non-paginated cats.
func (l listModel) paginationFooter() string {
	if l.page == 0 {
		return ""
	}
	switch l.cat {
	case CatTasks, CatMemories:
		// fall through
	default:
		return ""
	}
	count := len(l.list.Items())
	if l.loadingMore {
		return display.Dim.Render(fmt.Sprintf("%d loaded · page %d · loading next…", count, l.page))
	}
	if l.hasMore {
		return display.Dim.Render(fmt.Sprintf("%d loaded · page %d · ↓ for more", count, l.page))
	}
	return display.Dim.Render(fmt.Sprintf("%d loaded · all pages", count))
}
