package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// clearStatusMsg fires after a delay to wipe the transient status message
// (e.g. "✓ copied (n chars)") off the bottom bar.
type clearStatusMsg struct{}

// clearStatusAfter returns a tea.Cmd that emits a clearStatusMsg after d.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// resizeSettleMsg fires after the WindowSizeMsg burst settles.
// Carries a generation counter so stale ticks are dropped when a newer
// resize landed in the meantime — only the latest tick triggers the heavy
// reflow (markdown re-render at the new width).
type resizeSettleMsg struct{ gen uint64 }

// resizeDebounce schedules a resizeSettleMsg ~50ms in the future. Multiple
// rapid WindowSizeMsg events collapse into a single heavy reflow.
func resizeDebounce(gen uint64) tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return resizeSettleMsg{gen: gen}
	})
}

// pane identifies which of the three columns currently has keyboard focus.
type pane int

const (
	paneSidebar pane = iota
	paneList
	paneDetail
)

// rootModel owns the sidebar + list + detail panes and routes input to them.
type rootModel struct {
	client    *api.Client
	keys      keyMap
	focus     pane
	sidebar   sidebarModel
	list      listModel
	detail    detailModel
	width     int
	height    int
	statusMsg string
	projectID string // active project filter (empty = all)

	// Last-loaded category, so we can avoid reloading on noop transitions.
	loadedCat Category

	// Cache of the currently-shown task/memory IDs to avoid duplicate detail
	// fetches when the cursor doesn't move to a new item.
	lastSelectedID string

	// kanban buckets cached so the detail view can re-render without a
	// re-fetch when geometry changes.
	kanbanTodo       []models.Task
	kanbanInProgress []models.Task
	kanbanCompleted  []models.Task

	// profile bundle cached.
	profile *profileLoadedMsg

	// Yank cache — raw entities + their satellites currently shown in the
	// detail pane. Populated whenever a *DetailLoadedMsg arrives so the `c`
	// key can serialize the live entity to markdown without refetching.
	yankTask          *models.Task
	yankTaskSubtasks  []models.Subtask
	yankTaskNotes     []models.Annotation
	yankTaskMems      []models.Memory
	yankTaskComments  []models.Comment
	yankMemory        *models.Memory
	yankMemoryTasks   []models.Task
	yankMemoryComments []models.Comment
	yankProject       *models.Project
	yankProjectTasks  []models.Task
	yankProjectMems   []models.Memory
	yankOrg           *api.Organization
	yankOrgProjects   []models.Project
	yankOrgEncryption *api.OrgEncryptionStatus
	yankActivity      *models.ActivityItem

	// TUI chrome state.
	theme    string
	helpOpen bool
	caps     terminalCaps

	// Resize debounce: every WindowSizeMsg bumps resizeGen and schedules a
	// resizeSettleMsg with that generation. Only the matching settle fires
	// the heavy reflow (markdown re-render at the new width).
	resizeGen uint64
}

func newRootModel(c *api.Client) rootModel {
	theme := ThemeAuto
	if cfg, err := config.LoadConfig(); err == nil && cfg != nil && cfg.Theme != "" {
		theme = cfg.Theme
	}
	return rootModel{
		client:    c,
		keys:      defaultKeyMap(),
		focus:     paneSidebar,
		sidebar:   newSidebar(),
		loadedCat: -1,
		theme:     theme,
		caps:      detectTerminal(),
	}
}

func (m rootModel) Init() tea.Cmd {
	// Initial categorical load happens on the first WindowSizeMsg, since we
	// need a sized list pane first. Returning nil keeps Init lightweight.
	return nil
}

// layout sets pane geometry given the current terminal size.
func (m *rootModel) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	sw := m.width / 5
	if sw < 18 {
		sw = 18
	}
	lw := (m.width * 2) / 5
	if lw < 30 {
		lw = 30
	}
	dw := m.width - sw - lw
	if dw < 30 {
		dw = 30
	}
	h := m.height - 1 // status bar row

	m.sidebar.width, m.sidebar.height = sw, h
	if m.list.list.Items() == nil && m.list.width == 0 {
		m.list = newList(m.sidebar.selected(), lw, h)
	} else {
		m.list.resize(lw, h)
	}
	if m.detail.width == 0 {
		m.detail = newDetail(dw, h)
		m.detail.theme = m.theme
		m.detail.caps = m.caps
	} else {
		m.detail.resize(dw, h)
	}
	m.applyFocus()
}

func (m *rootModel) applyFocus() {
	m.sidebar.focused = m.focus == paneSidebar
	m.list.focused = m.focus == paneList
	m.detail.focused = m.focus == paneDetail
}

// loadForCategory issues the appropriate data-fetch cmd for the active
// sidebar category and resets the list pane's nav stack.
func (m *rootModel) loadForCategory() tea.Cmd {
	cat := m.sidebar.selected()
	m.loadedCat = cat
	m.list.resetStack(cat, cat.Label())
	m.list.loading = true
	m.list.errMsg = ""
	m.lastSelectedID = ""
	m.detail.setContent("")

	switch cat {
	case CatTasks:
		return loadTasks(m.client, m.projectID)
	case CatMemories:
		return loadMemories(m.client, m.projectID)
	case CatProjects:
		return loadProjects(m.client, "")
	case CatOrganizations:
		return loadOrgs(m.client)
	case CatActivity:
		return loadActivity(m.client)
	case CatKanban:
		if m.projectID == "" {
			m.list.setPlaceholder("press 'p' to pick a project for the kanban board")
			return nil
		}
		return loadKanban(m.client, m.projectID)
	case CatProfile:
		m.list.setProfileMode()
		return loadProfile(m.client)
	}
	return nil
}

// maybeFetchNextPage triggers an append-mode fetch when the cursor reaches
// the bottom guard band of a paginated list AND another page is available.
// Returns nil when no fetch should happen (kept simple — only Tasks and
// Memories paginate today).
func (m *rootModel) maybeFetchNextPage() tea.Cmd {
	if !m.list.shouldFetchMore() {
		return nil
	}
	// Only paginate top-level Tasks/Memories. Drill frames (e.g. tasks of a
	// project, projects of an org) keep the simple single-page behavior for
	// now — they're filtered down enough to fit in the default 100.
	if m.list.depth() != 1 {
		return nil
	}
	switch m.list.cat {
	case CatTasks:
		m.list.loadingMore = true
		m.statusMsg = "↓ loading more tasks…"
		return loadTasksPage(m.client, m.projectID, m.list.page+1, true)
	case CatMemories:
		m.list.loadingMore = true
		m.statusMsg = "↓ loading more memories…"
		return loadMemoriesPage(m.client, m.projectID, m.list.page+1, true)
	}
	return nil
}

// loadDetailForSelection issues a detail fetch for whatever's currently
// highlighted in the list, dispatching by category AND drill-down depth.
func (m *rootModel) loadDetailForSelection() tea.Cmd {
	sel := m.list.selected()
	if sel == nil || sel.id == "placeholder" {
		m.detail.setContent("")
		m.lastSelectedID = ""
		return nil
	}
	if sel.id == m.lastSelectedID {
		return nil
	}
	m.lastSelectedID = sel.id

	switch m.list.cat {
	case CatTasks:
		m.detail.setLoading(true)
		return loadTaskDetail(m.client, sel.id)
	case CatMemories:
		m.detail.setLoading(true)
		return loadMemoryDetail(m.client, sel.id)
	case CatProjects:
		m.detail.setLoading(true)
		return loadProjectDetail(m.client, sel.id)
	case CatOrganizations:
		m.detail.setLoading(true)
		return loadOrgDetail(m.client, sel.id)
	case CatActivity:
		// Inline render — no extra fetch.
		if item, ok := sel.raw.(models.ActivityItem); ok {
			it := item
			m.yankActivity = &it
			return m.detail.setContent(renderActivityDetail(item))
		}
		return nil
	case CatKanban:
		// Always render the same board regardless of which row is selected.
		m.detail.setRawContent(renderKanbanDetail(
			m.detail.width, m.kanbanTodo, m.kanbanInProgress, m.kanbanCompleted,
		))
		return nil
	case CatProfile:
		// Already populated by profileLoadedMsg.
		return nil
	}
	return nil
}

// handleEnterOnList implements the drill-down rules for `enter` (or →) on a
// list row. Returns (consumed, cmd). When consumed=false the caller should
// fall through to the existing "zoom into detail pane" behavior.
func (m *rootModel) handleEnterOnList() (bool, tea.Cmd) {
	sel := m.list.selected()
	if sel == nil || sel.id == "placeholder" {
		return false, nil
	}
	switch m.list.cat {
	case CatOrganizations:
		// Push: projects-of-this-org.
		if org, ok := sel.raw.(api.Organization); ok {
			m.list.pushFrame(CatProjects, org.Name, org.ID)
			m.lastSelectedID = ""
			return true, loadProjects(m.client, org.ID)
		}
	case CatProjects:
		// Projects sidebar shows project settings only — no task/memory drill.
		// To browse a project's content, the user switches to Tasks or
		// Memories and uses `p` to filter. Returning false lets the default
		// "zoom into detail pane" behavior take over.
		return false, nil
	}
	return false, nil
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		first := m.width == 0
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		// Re-render kanban board on resize so its column widths follow.
		// (Cheap — string formatting only, no glamour.)
		if m.list.cat == CatKanban && m.detail.content != "" {
			m.detail.setRawContent(renderKanbanDetail(
				m.detail.width, m.kanbanTodo, m.kanbanInProgress, m.kanbanCompleted,
			))
		}
		if first {
			// First sized frame: kick the initial categorical load AND
			// schedule an initial reflow tick (in case lastContent
			// arrives before the next resize).
			m.resizeGen++
			return m, tea.Batch(m.loadForCategory(), resizeDebounce(m.resizeGen))
		}
		// Debounce the heavy markdown reflow until the resize burst settles.
		m.resizeGen++
		return m, resizeDebounce(m.resizeGen)

	case resizeSettleMsg:
		// Drop stale ticks — only the latest generation triggers the
		// heavy markdown re-render. This collapses interactive-resize
		// bursts (many WindowSizeMsg events) into a single reflow.
		if msg.gen != m.resizeGen {
			return m, nil
		}
		m.detail.reflow()
		return m, nil

	case tea.KeyMsg:
		// Help overlay: ? toggles, esc/q closes; while open, swallow other keys.
		if key.Matches(msg, m.keys.Help) {
			m.helpOpen = !m.helpOpen
			return m, nil
		}
		if m.helpOpen {
			if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) {
				m.helpOpen = false
			}
			return m, nil
		}

		// Theme cycle (global): t.
		if key.Matches(msg, m.keys.Theme) {
			m.theme = nextTheme(m.theme)
			invalidateMarkdownCache()
			m.detail.setTheme(m.theme)
			// Persist (best effort).
			if cfg, err := config.LoadConfig(); err == nil && cfg != nil {
				cfg.Theme = m.theme
				_ = config.SaveConfig(cfg)
			}
			m.statusMsg = "✓ theme: " + m.theme
			return m, clearStatusAfter(2 * time.Second)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			// Cycle focus forward.
			m.focus = (m.focus + 1) % 3
			m.applyFocus()
			if m.focus == paneList {
				// Tab is also a category-commit gesture: if the sidebar
				// cursor sits on a different category than what's loaded,
				// fetch its data on the way past.
				if m.sidebar.selected() != m.loadedCat {
					return m, m.loadForCategory()
				}
				return m, m.loadDetailForSelection()
			}
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m, m.loadForCategory()

		case key.Matches(msg, m.keys.Yank):
			res := m.yankCurrent()
			if res.err != nil {
				m.statusMsg = "✗ " + res.err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("✓ copied (%d chars)", res.chars)
			}
			return m, clearStatusAfter(3 * time.Second)

		case key.Matches(msg, m.keys.Back):
			// Esc: pop drill-down frame first if we're in one.
			if m.focus == paneList && m.list.depth() > 1 {
				m.list.popFrame()
				m.lastSelectedID = ""
				return m, m.loadDetailForSelection()
			}
			if m.focus > paneSidebar {
				m.focus--
				m.applyFocus()
			}
			return m, nil
		}

		// Pane-scoped key handling.
		switch m.focus {
		case paneSidebar:
			switch {
			case key.Matches(msg, m.keys.Up):
				// Cursor-only — no backend hit. The category is committed
				// when the user presses Enter / Right (the explicit
				// "select" gesture, mirroring mc/ranger/yazi navigators).
				m.sidebar.movePrev()
				return m, nil
			case key.Matches(msg, m.keys.Down):
				m.sidebar.moveNext()
				return m, nil
			case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter):
				// Commit the highlighted category: load its data (only if
				// it actually changed since the last commit) and move
				// focus into the list pane.
				m.focus = paneList
				m.applyFocus()
				if m.sidebar.selected() != m.loadedCat {
					return m, m.loadForCategory()
				}
				return m, m.loadDetailForSelection()
			}

		case paneList:
			// Drill-down handling for h/← when we're inside a stacked frame.
			if key.Matches(msg, m.keys.Left) && m.list.depth() > 1 {
				m.list.popFrame()
				m.lastSelectedID = ""
				return m, m.loadDetailForSelection()
			}

			// Enter / → potentially drills further.
			if key.Matches(msg, m.keys.Enter) || key.Matches(msg, m.keys.Right) {
				if consumed, cmd := m.handleEnterOnList(); consumed {
					return m, cmd
				}
			}

			// Forward keys to the bubbles list (handles up/down/filter/etc).
			var cmd tea.Cmd
			m.list.list, cmd = m.list.list.Update(msg)
			// After cursor moves, refresh the detail pane.
			// Also: trigger an infinite-scroll page fetch when the cursor
			// reaches the bottom guard band of a paginated list.
			if key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) {
				cmds := []tea.Cmd{cmd, m.loadDetailForSelection()}
				if more := m.maybeFetchNextPage(); more != nil {
					cmds = append(cmds, more)
				}
				return m, tea.Batch(cmds...)
			}
			if key.Matches(msg, m.keys.Right) || key.Matches(msg, m.keys.Enter) {
				m.focus = paneDetail
				m.applyFocus()
				return m, cmd
			}
			if key.Matches(msg, m.keys.Left) {
				m.focus = paneSidebar
				m.applyFocus()
				return m, cmd
			}
			return m, cmd

		case paneDetail:
			var cmd tea.Cmd
			m.detail.vp, cmd = m.detail.vp.Update(msg)
			if key.Matches(msg, m.keys.Left) {
				m.focus = paneList
				m.applyFocus()
			}
			return m, cmd
		}

	case tasksLoadedMsg:
		if msg.err != nil {
			if msg.append {
				m.list.loadingMore = false
				m.statusMsg = "✗ next page failed: " + msg.err.Error()
				return m, clearStatusAfter(3 * time.Second)
			}
			m.list.setError(msg.err)
			return m, nil
		}
		if msg.append {
			m.list.appendTasks(msg.items, msg.page, msg.hasMore)
			m.statusMsg = fmt.Sprintf("✓ +%d tasks (page %d)", len(msg.items), msg.page)
			return m, clearStatusAfter(2 * time.Second)
		}
		m.list.setTasks(msg.items, msg.page, msg.hasMore)
		return m, m.loadDetailForSelection()

	case memoriesLoadedMsg:
		if msg.err != nil {
			if msg.append {
				m.list.loadingMore = false
				m.statusMsg = "✗ next page failed: " + msg.err.Error()
				return m, clearStatusAfter(3 * time.Second)
			}
			m.list.setError(msg.err)
			return m, nil
		}
		if msg.append {
			m.list.appendMemories(msg.items, msg.page, msg.hasMore)
			m.statusMsg = fmt.Sprintf("✓ +%d memories (page %d)", len(msg.items), msg.page)
			return m, clearStatusAfter(2 * time.Second)
		}
		m.list.setMemories(msg.items, msg.page, msg.hasMore)
		return m, m.loadDetailForSelection()

	case projectsLoadedMsg:
		if msg.err != nil {
			m.list.setError(msg.err)
			return m, nil
		}
		m.list.setProjects(msg.items)
		return m, m.loadDetailForSelection()

	case orgsLoadedMsg:
		if msg.err != nil {
			m.list.setError(msg.err)
			return m, nil
		}
		m.list.setOrgs(msg.items)
		return m, m.loadDetailForSelection()

	case activityLoadedMsg:
		if msg.err != nil {
			m.list.setError(msg.err)
			return m, nil
		}
		m.list.setActivity(msg.items)
		return m, m.loadDetailForSelection()

	case kanbanLoadedMsg:
		if msg.err != nil {
			m.list.setError(msg.err)
			return m, nil
		}
		m.kanbanTodo = msg.todo
		m.kanbanInProgress = msg.inProgress
		m.kanbanCompleted = msg.completed
		m.list.setKanbanSummary(len(msg.todo), len(msg.inProgress), len(msg.completed))
		m.detail.setRawContent(renderKanbanDetail(
			m.detail.width, m.kanbanTodo, m.kanbanInProgress, m.kanbanCompleted,
		))
		return m, nil

	case profileLoadedMsg:
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		// Cache for future re-renders.
		mm := msg
		m.profile = &mm
		cmd := m.detail.setContent(renderProfileDetail(
			msg.profile, msg.orgs, msg.agents, msg.stats, msg.oauth,
		))
		return m, cmd

	case taskDetailLoadedMsg:
		if msg.taskID != m.lastSelectedID {
			// Stale message from a previous selection — ignore.
			return m, nil
		}
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		// Stash the raw entities so `c` can serialize them.
		m.yankTask = msg.task
		m.yankTaskSubtasks = msg.subtasks
		m.yankTaskNotes = msg.annotations
		m.yankTaskMems = msg.linkedMems
		m.yankTaskComments = msg.comments
		cmd := m.detail.setContent(renderTaskDetail(
			msg.task, msg.subtasks, msg.annotations, msg.linkedMems, msg.comments,
		))
		return m, cmd

	case memoryDetailLoadedMsg:
		if msg.memoryID != m.lastSelectedID {
			return m, nil
		}
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		m.yankMemory = msg.memory
		m.yankMemoryTasks = msg.linkedTasks
		m.yankMemoryComments = msg.comments
		cmd := m.detail.setContent(renderMemoryDetail(
			msg.memory, msg.linkedTasks, msg.comments,
		))
		return m, cmd

	case projectDetailLoadedMsg:
		if msg.projectID != m.lastSelectedID {
			return m, nil
		}
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		m.yankProject = msg.project
		m.yankProjectTasks = msg.tasks
		m.yankProjectMems = msg.memories
		cmd := m.detail.setContent(renderProjectDetail(
			msg.project, msg.tasks, msg.memories,
		))
		return m, cmd

	case orgDetailLoadedMsg:
		if msg.orgID != m.lastSelectedID {
			return m, nil
		}
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		m.yankOrg = msg.org
		m.yankOrgProjects = msg.projects
		m.yankOrgEncryption = msg.encryption
		cmd := m.detail.setContent(renderOrgDetail(
			msg.org, msg.projects, msg.encryption,
		))
		return m, cmd

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case renderedMsg:
		// Async glamour render finished; swap in if still relevant.
		m.detail.applyRendered(msg.token, msg.output)
		return m, nil
	}

	return m, nil
}

func (m rootModel) View() string {
	if m.width == 0 {
		return "loading…"
	}
	if m.helpOpen {
		return helpOverlay(m.keys, m.width, m.height)
	}
	return m.normalView()
}

func (m rootModel) normalView() string {
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.sidebar.View(),
		m.list.View(),
		m.detail.View(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, body, m.renderStatusBar())
}

func (m rootModel) renderStatusBar() string {
	help := "h/← back · j/k up/dn · l/→ forward · ↵ open · ⇥ pane · / search · p project · r refresh · c copy · t theme · ? help · q quit"
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(help)
	if m.statusMsg != "" {
		bar = lipgloss.NewStyle().Foreground(display.ColorGood).Render(m.statusMsg) + "  " + bar
	}
	return bar
}

// Compile-time assertion: rootModel implements tea.Model.
var _ tea.Model = rootModel{}
