package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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

type cursorSettleMsg struct{ gen uint64 }

func cursorDebounce(gen uint64) tea.Cmd {
	return tea.Tick(75*time.Millisecond, func(time.Time) tea.Msg {
		return cursorSettleMsg{gen: gen}
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
	client      *api.Client
	keys        keyMap
	focus       pane
	sidebar     sidebarModel
	list        listModel
	detail      detailModel
	width       int
	height      int
	statusMsg   string
	projectID   string // active project filter (empty = all)
	projectName string

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
	yankTask           *models.Task
	yankTaskSubtasks   []models.Subtask
	yankTaskNotes      []models.Annotation
	yankTaskMems       []models.Memory
	yankTaskComments   []models.Comment
	yankMemory         *models.Memory
	yankMemoryTasks    []models.Task
	yankMemoryComments []models.Comment
	yankProject        *models.Project
	yankProjectTasks   []models.Task
	yankProjectMems    []models.Memory
	yankOrg            *api.Organization
	yankOrgProjects    []models.Project
	yankOrgEncryption  *api.OrgEncryptionStatus
	yankActivity       *models.ActivityItem

	// TUI chrome state.
	theme         string
	accentSpec    string // current accent spec ("auto"/"brand"/index/hex) for the A-cycle + persistence
	accentGlamour string // accent color threaded into the detail pane's glamour renderer
	helpOpen      bool
	caps          terminalCaps

	// Interactive overlay state (create / recall prompt, delete confirm).
	overlay        overlayKind
	input          textinput.Model
	promptIntent   promptIntent
	confirmVerb    string  // e.g. "Delete task abc123?"
	confirmCmd     tea.Cmd // action to run if the user confirms
	lastRecallTerm string  // last recall query, so post-action refresh re-runs it

	// Resize debounce: every WindowSizeMsg bumps resizeGen and schedules a
	// resizeSettleMsg with that generation. Only the matching settle fires
	// the heavy reflow (markdown re-render at the new width).
	resizeGen uint64

	// Cursor debounce: rapid j/k/page motion should move instantly but only
	// fetch the expensive right-pane detail after the user pauses.
	cursorGen uint64
}

func newRootModel(c *api.Client) rootModel {
	theme := ThemeAuto
	accentSpec := "auto"
	if cfg, err := config.LoadConfig(); err == nil && cfg != nil {
		if cfg.Theme != "" {
			theme = cfg.Theme
		}
		if cfg.Accent != "" {
			accentSpec = cfg.Accent
		}
	}
	ti := textinput.New()
	ti.Prompt = "› "
	ti.CharLimit = 512
	return rootModel{
		client:        c,
		keys:          defaultKeyMap(),
		focus:         paneSidebar,
		sidebar:       newSidebar(),
		loadedCat:     -1,
		theme:         theme,
		accentSpec:    accentSpec,
		accentGlamour: resolveAccent(accentSpec).Glamour,
		caps:          detectTerminal(),
		input:         ti,
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
	m.detail.accent = m.accentGlamour
	m.applyFocus()
}

func (m *rootModel) applyFocus() {
	m.sidebar.focused = m.focus == paneSidebar
	m.list.setFocused(m.focus == paneList)
	m.detail.focused = m.focus == paneDetail
}

func (m *rootModel) scheduleDetailLoad() tea.Cmd {
	m.cursorGen++
	return cursorDebounce(m.cursorGen)
}

func (m *rootModel) activateCategoryIndex(i int) tea.Cmd {
	if !m.sidebar.selectIndex(i) && m.sidebar.selected() == m.loadedCat {
		m.focus = paneList
		m.applyFocus()
		return m.loadDetailForSelection()
	}
	m.focus = paneList
	m.applyFocus()
	return m.loadForCategory()
}

func (m *rootModel) handleProjectShortcut() tea.Cmd {
	if m.focus == paneList && m.list.cat == CatProjects {
		if sel := m.list.selected(); sel != nil {
			if p, ok := sel.raw.(models.Project); ok {
				m.projectID = p.ID.String()
				m.projectName = p.Name
				m.statusMsg = "✓ project filter: " + p.Name
				m.sidebar.selectIndex(categoryIndex(CatTasks))
				m.loadedCat = -1
				m.focus = paneList
				m.applyFocus()
				return tea.Batch(m.loadForCategory(), clearStatusAfter(3*time.Second))
			}
		}
	}
	m.statusMsg = "pick a project, then press p again"
	return tea.Batch(m.activateCategoryIndex(categoryIndex(CatProjects)), clearStatusAfter(3*time.Second))
}

func (m *rootModel) clearProjectFilter() tea.Cmd {
	if m.projectID == "" {
		m.statusMsg = "showing all projects"
		return clearStatusAfter(2 * time.Second)
	}
	m.projectID = ""
	m.projectName = ""
	m.statusMsg = "✓ project filter cleared"
	return tea.Batch(m.loadForCategory(), clearStatusAfter(2*time.Second))
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
	m.detail.title = detailTitleFor(m.list.cat)

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
	case CatSearch:
		// Recall hit — dispatch the detail fetch by the hit's entity type.
		if it, ok := sel.raw.(api.FindItem); ok {
			if strings.EqualFold(it.Type, "task") {
				m.detail.title = "Task"
				m.detail.setLoading(true)
				return loadTaskDetail(m.client, sel.id)
			}
			m.detail.title = "Memory"
			m.detail.setLoading(true)
			return loadMemoryDetail(m.client, sel.id)
		}
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

	case cursorSettleMsg:
		if msg.gen != m.cursorGen {
			return m, nil
		}
		return m, m.loadDetailForSelection()

	case tea.KeyMsg:
		// Interactive overlays consume all keys while open.
		switch m.overlay {
		case overlayPrompt:
			return m.updatePrompt(msg)
		case overlayConfirm:
			return m.updateConfirm(msg)
		}

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

		// Accent cycle (global): A. Steps auto → brand → blue/magenta/cyan/…,
		// follows the terminal palette, updates chrome + markdown live.
		if key.Matches(msg, m.keys.Accent) {
			m.accentSpec = nextAccentSpec(m.accentSpec)
			pal := resolveAccent(m.accentSpec)
			display.SetAccent(pal.Accent, pal.Bright)
			m.accentGlamour = pal.Glamour
			m.detail.accent = pal.Glamour
			invalidateMarkdownCache()
			m.detail.setTheme(m.theme) // re-render current markdown with the new accent
			if cfg, err := config.LoadConfig(); err == nil && cfg != nil {
				cfg.Accent = m.accentSpec
				_ = config.SaveConfig(cfg)
			}
			m.statusMsg = "✓ accent: " + m.accentSpec
			return m, clearStatusAfter(2 * time.Second)
		}

		// Nerd-font toggle (global): I. Reloads so list badges + detail glyphs
		// re-render with the new icon set.
		if key.Matches(msg, m.keys.NerdToggle) {
			setNerdFont(!nerdFont)
			on := nerdFont
			if cfg, err := config.LoadConfig(); err == nil && cfg != nil {
				cfg.NerdFont = &on
				_ = config.SaveConfig(cfg)
			}
			if on {
				m.statusMsg = "✓ icons: nerd"
			} else {
				m.statusMsg = "✓ icons: unicode"
			}
			m.lastSelectedID = ""
			invalidateMarkdownCache()
			return m, tea.Batch(m.loadForCategory(), clearStatusAfter(2*time.Second))
		}

		if m.focus == paneList && m.list.list.SettingFilter() {
			var cmd tea.Cmd
			wasFiltering := m.list.list.SettingFilter()
			m.list.list, cmd = m.list.list.Update(msg)
			if wasFiltering && !m.list.list.SettingFilter() {
				return m, tea.Batch(cmd, m.scheduleDetailLoad())
			}
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Cat1):
			return m, m.activateCategoryIndex(0)
		case key.Matches(msg, m.keys.Cat2):
			return m, m.activateCategoryIndex(1)
		case key.Matches(msg, m.keys.Cat3):
			return m, m.activateCategoryIndex(2)
		case key.Matches(msg, m.keys.Cat4):
			return m, m.activateCategoryIndex(3)
		case key.Matches(msg, m.keys.Cat5):
			return m, m.activateCategoryIndex(4)
		case key.Matches(msg, m.keys.Cat6):
			return m, m.activateCategoryIndex(5)

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

		case key.Matches(msg, m.keys.PrevTab):
			if m.focus == paneList && m.list.depth() > 1 {
				m.list.popFrame()
				m.lastSelectedID = ""
				return m, m.loadDetailForSelection()
			}
			m.focus = (m.focus + 2) % 3
			m.applyFocus()
			if m.focus == paneList {
				return m, m.loadDetailForSelection()
			}
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m, m.loadForCategory()

		case key.Matches(msg, m.keys.Project):
			return m, m.handleProjectShortcut()

		case key.Matches(msg, m.keys.AllProjects):
			return m, m.clearProjectFilter()

		case key.Matches(msg, m.keys.Yank):
			res := m.yankCurrent()
			if res.err != nil {
				m.statusMsg = "✗ " + res.err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("✓ copied (%d chars)", res.chars)
			}
			return m, clearStatusAfter(3 * time.Second)

		case key.Matches(msg, m.keys.Recall):
			return m, m.startRecall()
		case key.Matches(msg, m.keys.New):
			return m, m.startCreate()
		case m.focus == paneList && key.Matches(msg, m.keys.Toggle):
			return m, m.toggleSelected()
		case m.focus != paneSidebar && key.Matches(msg, m.keys.StartTask):
			return m, m.startSelectedTask()
		case m.focus != paneSidebar && key.Matches(msg, m.keys.Delete):
			return m, m.askDelete()

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
			if key.Matches(msg, m.keys.Left) {
				if m.list.depth() > 1 {
					m.list.popFrame()
					m.lastSelectedID = ""
					return m, m.loadDetailForSelection()
				}
				m.focus = paneSidebar
				m.applyFocus()
				return m, nil
			}

			// Drill-down handling for h/← when we're inside a stacked frame.
			if key.Matches(msg, m.keys.Enter) || key.Matches(msg, m.keys.Right) {
				if consumed, cmd := m.handleEnterOnList(); consumed {
					return m, cmd
				}
				m.focus = paneDetail
				m.applyFocus()
				return m, nil
			}

			// Forward keys to the bubbles list (handles up/down/filter/etc).
			var cmd tea.Cmd
			m.list.list, cmd = m.list.list.Update(msg)
			// After cursor moves, refresh the detail pane.
			// Also: trigger an infinite-scroll page fetch when the cursor
			// reaches the bottom guard band of a paginated list.
			if key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) ||
				key.Matches(msg, m.keys.PrevPage) || key.Matches(msg, m.keys.NextPage) ||
				key.Matches(msg, m.keys.Top) || key.Matches(msg, m.keys.Bottom) {
				cmds := []tea.Cmd{cmd, m.scheduleDetailLoad()}
				if more := m.maybeFetchNextPage(); more != nil {
					cmds = append(cmds, more)
				}
				return m, tea.Batch(cmds...)
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

	case actionDoneMsg:
		if !msg.ok {
			m.statusMsg = "✗ " + msg.err.Error()
			return m, clearStatusAfter(4 * time.Second)
		}
		m.statusMsg = "✓ " + msg.verb
		cmds := []tea.Cmd{clearStatusAfter(3 * time.Second)}
		if msg.refresh {
			m.lastSelectedID = ""
			if m.list.cat == CatSearch && m.lastRecallTerm != "" {
				// Stay in the recall frame: re-run the query instead of
				// resetting the stack back to a base category.
				cmds = append(cmds, recallCmd(m.client, m.lastRecallTerm, m.projectID))
			} else {
				cmds = append(cmds, m.loadForCategory())
			}
		}
		return m, tea.Batch(cmds...)

	case recallLoadedMsg:
		if msg.err != nil {
			m.list.setError(msg.err)
			return m, nil
		}
		m.list.setSearchResults(msg.items)
		m.statusMsg = fmt.Sprintf("✓ %d hits for \"%s\"", len(msg.items), msg.term)
		return m, tea.Batch(m.loadDetailForSelection(), clearStatusAfter(4*time.Second))

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
		return lipgloss.JoinVertical(
			lipgloss.Left,
			helpOverlay(m.keys, m.width, maxInt(m.height-1, 1)),
			m.renderStatusBar(),
		)
	}
	switch m.overlay {
	case overlayPrompt:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			renderPromptOverlay(promptTitleFor(m.promptIntent), m.input.View(), m.width, maxInt(m.height-1, 1)),
			m.renderStatusBar(),
		)
	case overlayConfirm:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			renderConfirmOverlay(m.confirmVerb, m.width, maxInt(m.height-1, 1)),
			m.renderStatusBar(),
		)
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

// renderStatusBar draws the always-visible, context-aware footer: a left
// segment (focus · scope · counts) plus right-aligned key hints, with any
// transient status message taking the left-most slot.
func (m rootModel) renderStatusBar() string {
	w := maxInt(m.width, 1)

	scope := "all projects"
	if m.projectName != "" {
		scope = m.projectName
	} else if m.projectID != "" {
		scope = shortID(m.projectID)
	}
	focus := "sidebar"
	switch m.focus {
	case paneList:
		focus = strings.ToLower(m.list.cat.Label())
	case paneDetail:
		focus = "detail"
	}
	seg := []string{focus, scope}
	if m.focus == paneList && !m.list.list.SettingFilter() {
		if n := len(m.list.list.Items()); n > 0 {
			seg = append(seg, fmt.Sprintf("%d items", n))
		}
	}
	left := display.FooterSeg.Render(strings.Join(seg, " · "))

	status := ""
	if m.statusMsg != "" {
		status = lipgloss.NewStyle().
			Foreground(display.ColorGood).
			Background(display.ColorFooterBg).
			Render(m.statusMsg)
	}

	return renderFooter(w, left, m.footerHints(), status)
}

// footerHints returns the context-aware key-hint segments for the footer.
// Only the single most-likely "next action" key takes a semantic accent
// (warn/info); every other key uses the resting accent (the lazygit rule).
func (m rootModel) footerHints() []string {
	if m.helpOpen {
		return []string{keyHint("?", "close"), keyHint("esc", "close"), keyHint("q", "quit")}
	}
	switch m.overlay {
	case overlayPrompt:
		return []string{
			display.FooterDsc.Render("type"),
			keyHintAccent("↵", "submit", display.ColorInfo),
			keyHint("esc", "cancel"),
		}
	case overlayConfirm:
		return []string{
			keyHintAccent("y", "confirm", display.ColorWarn),
			keyHint("n", "cancel"),
		}
	}
	switch m.focus {
	case paneSidebar:
		return []string{
			keyHint("↑↓", "choose"), keyHint("↵", "open"),
			keyHint("1-6", "jump"), keyHint("s", "recall"),
			keyHint("?", "help"), keyHint("q", "quit"),
		}
	case paneList:
		if m.list.list.SettingFilter() {
			return []string{
				display.FooterDsc.Render("type to filter"),
				keyHintAccent("↵", "apply", display.ColorInfo),
				keyHint("esc", "cancel"),
			}
		}
		hints := []string{keyHint("↑↓", "move"), keyHint("↵", "open"), keyHint("/", "filter")}
		switch m.list.cat {
		case CatProjects:
			hints = append(hints, keyHintAccent("p", "set project", display.ColorWarn))
		case CatTasks:
			hints = append(hints, keyHint("space", "done"), keyHint("S", "start"),
				keyHint("n", "new"), keyHint("D", "del"), keyHint("p", "project"))
		case CatMemories:
			hints = append(hints, keyHint("n", "new"), keyHint("D", "del"), keyHint("p", "project"))
		case CatOrganizations:
			hints = append(hints, keyHint("↵", "projects"))
		}
		hints = append(hints,
			keyHint("s", "recall"), keyHint("c", "copy"),
			keyHint("?", "help"), keyHint("q", "quit"),
		)
		return hints
	case paneDetail:
		return []string{
			keyHint("↑↓", "scroll"), keyHint("^u^d", "page"),
			keyHint("h", "back"), keyHint("c", "copy"),
			keyHint("s", "recall"), keyHint("?", "help"), keyHint("q", "quit"),
		}
	}
	return []string{keyHint("?", "help"), keyHint("q", "quit")}
}

// detailTitleFor maps a sidebar category to the right-pane title shown in the
// detail pane's top border.
func detailTitleFor(c Category) string {
	switch c {
	case CatTasks:
		return "Task"
	case CatMemories:
		return "Memory"
	case CatProjects:
		return "Project"
	case CatOrganizations:
		return "Organization"
	case CatActivity:
		return "Activity"
	case CatKanban:
		return "Kanban"
	case CatProfile:
		return "Profile"
	}
	return "Detail"
}

// ---- interactive actions ---------------------------------------------------

// activeCat is the category the action keys operate on: the focused list's
// category, or the sidebar cursor's category when the sidebar has focus.
func (m rootModel) activeCat() Category {
	if m.focus == paneSidebar {
		return m.sidebar.selected()
	}
	return m.list.cat
}

func promptTitleFor(intent promptIntent) string {
	switch intent {
	case promptCreateTask:
		return "New task"
	case promptCreateMemory:
		return "New memory"
	case promptRecall:
		return "Recall · hybrid find"
	}
	return ""
}

// openPrompt switches into the single-line prompt overlay.
func (m *rootModel) openPrompt(intent promptIntent, placeholder string) tea.Cmd {
	m.overlay = overlayPrompt
	m.promptIntent = intent
	m.input.Reset()
	m.input.Placeholder = placeholder
	m.input.Width = maxInt(minInt(m.width-16, 56), 16)
	return m.input.Focus()
}

func (m *rootModel) startRecall() tea.Cmd {
	return m.openPrompt(promptRecall, "search term…")
}

func (m *rootModel) startCreate() tea.Cmd {
	switch m.activeCat() {
	case CatTasks:
		if m.projectID == "" {
			m.statusMsg = "press p to pick a project before creating"
			return clearStatusAfter(3 * time.Second)
		}
		return m.openPrompt(promptCreateTask, "task title…")
	case CatMemories:
		if m.projectID == "" {
			m.statusMsg = "press p to pick a project before creating"
			return clearStatusAfter(3 * time.Second)
		}
		return m.openPrompt(promptCreateMemory, "memory content…")
	}
	m.statusMsg = "new works in Tasks or Memories"
	return clearStatusAfter(3 * time.Second)
}

// updatePrompt routes keys while the text prompt is open.
func (m rootModel) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.overlay = overlayNone
		m.input.Blur()
		return m, nil
	case "enter":
		return m.submitPrompt()
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m rootModel) submitPrompt() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.input.Value())
	intent := m.promptIntent
	m.overlay = overlayNone
	m.input.Blur()
	if val == "" {
		return m, nil
	}
	switch intent {
	case promptRecall:
		m.focus = paneList
		m.applyFocus()
		m.list.pushFrame(CatSearch, "Search: "+display.Truncate(val, 24), "")
		m.lastSelectedID = ""
		m.lastRecallTerm = val
		m.statusMsg = "searching…"
		return m, recallCmd(m.client, val, m.projectID)
	case promptCreateTask:
		m.statusMsg = "creating task…"
		return m, createTaskCmd(m.client, m.projectID, val)
	case promptCreateMemory:
		m.statusMsg = "creating memory…"
		return m, createMemoryCmd(m.client, m.projectID, val)
	}
	return m, nil
}

// updateConfirm routes keys while the yes/no delete confirm is open.
func (m rootModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		cmd := m.confirmCmd
		m.overlay = overlayNone
		m.confirmCmd = nil
		m.statusMsg = "deleting…"
		return m, cmd
	case "n", "N", "esc":
		m.overlay = overlayNone
		m.confirmCmd = nil
		return m, nil
	}
	return m, nil
}

// selectedEntity resolves the highlighted row to an (id, kind) pair, working
// across normal category frames (raw is a models.Task/Memory) AND recall
// results (raw is an api.FindItem). task is non-nil only when the full task
// entity is in hand (so toggle can read its status). kind is "task"/"memory"/"".
func (m rootModel) selectedEntity() (id, kind string, task *models.Task, ok bool) {
	sel := m.list.selected()
	if sel == nil || sel.id == "" || sel.id == "placeholder" {
		return "", "", nil, false
	}
	switch r := sel.raw.(type) {
	case models.Task:
		t := r
		return sel.id, "task", &t, true
	case models.Memory:
		return sel.id, "memory", nil, true
	case api.FindItem:
		k := "memory"
		if strings.EqualFold(r.Type, "task") {
			k = "task"
		}
		return sel.id, k, nil, true
	}
	return sel.id, "", nil, false
}

func (m *rootModel) toggleSelected() tea.Cmd {
	id, kind, task, ok := m.selectedEntity()
	if !ok || kind != "task" {
		m.statusMsg = "complete works on a task"
		return clearStatusAfter(2 * time.Second)
	}
	// Reopen only when we know the task is already completed; from a recall
	// hit (no status in hand) the action completes.
	if task != nil && strings.EqualFold(task.Status, "COMPLETED") {
		m.statusMsg = "reopening…"
		return reopenTaskCmd(m.client, id)
	}
	m.statusMsg = "completing…"
	return completeTaskCmd(m.client, id)
}

func (m *rootModel) startSelectedTask() tea.Cmd {
	id, kind, _, ok := m.selectedEntity()
	if !ok || kind != "task" {
		m.statusMsg = "start works on a task"
		return clearStatusAfter(2 * time.Second)
	}
	m.statusMsg = "starting…"
	return startTaskCmd(m.client, id)
}

func (m *rootModel) askDelete() tea.Cmd {
	id, kind, _, ok := m.selectedEntity()
	if !ok {
		return nil
	}
	switch kind {
	case "task":
		m.overlay = overlayConfirm
		m.confirmVerb = "Delete task " + shortID(id) + "?"
		m.confirmCmd = deleteTaskCmd(m.client, id)
	case "memory":
		m.overlay = overlayConfirm
		m.confirmVerb = "Delete memory " + shortID(id) + "?"
		m.confirmCmd = deleteMemoryCmd(m.client, id)
	default:
		m.statusMsg = "delete works on tasks or memories"
		return clearStatusAfter(2 * time.Second)
	}
	return nil
}

// Compile-time assertion: rootModel implements tea.Model.
var _ tea.Model = rootModel{}
