package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// listItem wraps any entity for bubbles/list. The row is laid out by
// rowDelegate from these column parts — `title` is PLAIN text (no embedded
// ANSI) so the selection bar's background can never be severed by a reset.
type listItem struct {
	id         string
	title      string // plain, single-line title text
	sub        string
	filter     string
	badge      string         // plain badge text, e.g. "[H]" or "[decision]"
	badgeStyle lipgloss.Style // color applied to badge on non-selected rows
	rel        string         // right-aligned relative time, e.g. "2h ago"
	raw        interface{}    // original entity (Task, Memory, ...)
}

func (i listItem) FilterValue() string {
	if i.filter != "" {
		return i.filter
	}
	return i.title
}
func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.sub }

// rowDelegate renders one list row as columns (badge · id · title · relative
// time) with a full-width selection bar. `focused` mirrors the list pane's
// focus so the selected row shows the accent bar only when the pane is active
// (bold-only otherwise — the lazygit two-state rule).
type rowDelegate struct{ focused bool }

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(listItem)
	if !ok {
		return
	}
	width := maxInt(m.Width(), 1)
	inner := maxInt(width-2, 1) // 1-col pad each side
	selected := index == m.Index()

	// Special rows (kanban summary, profile hint, placeholder) carry only a
	// pre-styled title — render verbatim, no columns and no bar (their ANSI
	// would break a background fill anyway).
	if it.badge == "" && it.rel == "" {
		line := lipgloss.NewStyle().Width(width).MaxWidth(width).Render(" " + it.title)
		if selected {
			line = lipgloss.NewStyle().Bold(true).Width(width).MaxWidth(width).Render(" " + display.SingleLine(it.title))
		}
		fmt.Fprint(w, line)
		return
	}

	idStr := shortID(it.id)
	title := display.SingleLine(it.title)
	badgeW := lipgloss.Width(it.badge)
	relW := lipgloss.Width(it.rel)

	// Budget the title so badge / id / relative-time keep fixed columns.
	used := badgeW + 1 + len(idStr) + 2
	if relW > 0 {
		used += relW + 1
	}
	if b := inner - used; b >= 1 {
		title = display.Truncate(title, b)
	} else {
		title = display.Truncate(title, 1)
	}

	// Selected + focused: full-width accent bar, built from PLAIN text only.
	if selected && d.focused {
		left := it.badge + " " + idStr + "  " + title
		gap := inner - lipgloss.Width(left) - relW
		if gap < 0 {
			gap = 0
		}
		line := " " + left + strings.Repeat(" ", gap) + it.rel + " "
		fmt.Fprint(w, display.SelRowStyle.Width(width).MaxWidth(width).Render(line))
		return
	}

	// Resting (or unfocused-selected) row: colored columns.
	left := it.badgeStyle.Render(it.badge) + " " + display.Dim.Render(idStr) + "  " + title
	gap := inner - badgeW - 1 - len(idStr) - 2 - lipgloss.Width(title) - relW
	if gap < 0 {
		gap = 0
	}
	line := " " + left + strings.Repeat(" ", gap) + display.Dim.Render(it.rel) + " "
	style := lipgloss.NewStyle().Width(width).MaxWidth(width)
	if selected { // unfocused-selected → bold, no bar
		style = style.Bold(true)
	}
	fmt.Fprint(w, style.Render(line))
}

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
	w := maxInt(width-2, 10)
	h := maxInt(height-3, 1)
	l := list.New(nil, rowDelegate{}, w, h)
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
	// Reserve exactly one body row for the optional footer line. The
	// breadcrumb (drill-down frames) and the pagination footer (top-level
	// Tasks/Memories) are mutually exclusive, so a single reserved row always
	// suffices — the list fills to the bottom border with neither overflow nor
	// dead space.
	innerH := maxInt(height-3, 1)
	l.list.SetSize(maxInt(width-2, 10), innerH)
}

// setFocused mirrors pane focus into the row delegate so the selected row
// switches between the accent bar (focused) and bold-only (unfocused).
func (l *listModel) setFocused(b bool) {
	if l.focused == b {
		return
	}
	l.focused = b
	l.list.SetDelegate(rowDelegate{focused: b})
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
	badge, st := priorityBadgeParts(t.Priority)
	return listItem{
		id:         t.ID.String(),
		title:      display.SingleLine(title),
		badge:      badge,
		badgeStyle: st,
		rel:        display.Relative(t.UpdatedAt),
		filter:     strings.Join([]string{shortID(t.ID.String()), title, t.Priority, t.Status}, " "),
		raw:        t,
	}
}

// memoryToItem renders a single memory as a list cell.
func memoryToItem(m models.Memory) list.Item {
	content := display.SingleLine(decryptMemoryContent(&m))
	badge, st := typeBadgeParts(m.Type)
	return listItem{
		id:         m.ID.String(),
		title:      content,
		badge:      badge,
		badgeStyle: st,
		rel:        display.Relative(m.UpdatedAt),
		filter:     strings.Join([]string{shortID(m.ID.String()), content, m.Type}, " "),
		raw:        m,
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
		title := p.Name
		if desc := display.SingleLine(p.Description); desc != "" {
			title += "  " + desc
		}
		items = append(items, listItem{
			id:         p.ID.String(),
			title:      title,
			badge:      "[project]",
			badgeStyle: display.Dim,
			rel:        display.Relative(p.UpdatedAt),
			filter:     strings.Join([]string{shortID(p.ID.String()), p.Name, p.Description}, " "),
			raw:        p,
		})
	}
	l.applyItems(items)
}

// setOrgs renders organizations as one-line cells.
func (l *listModel) setOrgs(orgs []api.Organization) {
	items := make([]list.Item, 0, len(orgs))
	for i := range orgs {
		o := orgs[i]
		items = append(items, listItem{
			id:         o.ID,
			title:      o.Name,
			badge:      "[org]",
			badgeStyle: display.Dim,
			filter:     strings.Join([]string{shortID(o.ID), o.Name}, " "),
			raw:        o,
		})
	}
	l.applyItems(items)
}

// setActivity renders a flat activity feed.
func (l *listModel) setActivity(events []models.ActivityItem) {
	items := make([]list.Item, 0, len(events))
	for i := range events {
		e := events[i]
		badge, st := typeBadgeParts(e.EntityType)
		items = append(items, listItem{
			id:         e.EntityID.String(),
			title:      display.SingleLine(e.Summary),
			badge:      badge,
			badgeStyle: st,
			rel:        display.Relative(e.Timestamp),
			filter:     strings.Join([]string{e.EntityID.String(), e.EntityType, e.Summary}, " "),
			raw:        e,
		})
	}
	l.applyItems(items)
}

// setSearchResults renders hybrid-recall hits. Each row's badge is the hit
// type (memory/task/decision/…) and the raw entity is the api.FindItem so the
// detail pane can dispatch by type on selection.
func (l *listModel) setSearchResults(items []api.FindItem) {
	rows := make([]list.Item, 0, len(items))
	for i := range items {
		it := items[i]
		badge, st := typeBadgeParts(itemKind(it))
		title := display.SingleLine(it.Title)
		if title == "" {
			title = display.SingleLine(it.Preview)
		}
		rows = append(rows, listItem{
			id:         it.ID,
			title:      title,
			badge:      badge,
			badgeStyle: st,
			rel:        display.Relative(it.CreatedAt),
			filter:     it.ID + " " + it.Title + " " + it.Preview,
			raw:        it,
		})
	}
	if len(rows) == 0 {
		l.setPlaceholder("no matches — try a different query")
		return
	}
	l.applyItems(rows)
}

// itemKind picks the most descriptive label for a recall hit's badge.
func itemKind(it api.FindItem) string {
	if it.Kind != "" {
		return it.Kind
	}
	return it.Type
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

	// Pane title follows the active drill-down frame; count is the item total.
	title := l.cat.Label()
	if top := l.topFrame(); top.label != "" {
		title = top.label
	}
	count := ""
	if n := len(l.list.Items()); n > 0 && !l.loading && l.errMsg == "" {
		count = fmt.Sprintf("%d", n)
	}
	return titledPane(title, count, inner, l.width, l.height, l.focused)
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
