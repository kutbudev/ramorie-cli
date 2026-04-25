package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

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
}

func newRootModel(c *api.Client) rootModel {
	return rootModel{
		client:    c,
		keys:      defaultKeyMap(),
		focus:     paneSidebar,
		sidebar:   newSidebar(),
		loadedCat: -1,
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
// sidebar category.
func (m *rootModel) loadForCategory() tea.Cmd {
	cat := m.sidebar.selected()
	m.loadedCat = cat
	m.list.cat = cat
	m.list.loading = true
	m.list.errMsg = ""
	m.lastSelectedID = ""
	m.detail.setContent("")

	switch cat {
	case CatTasks:
		return loadTasks(m.client, m.projectID)
	case CatMemories:
		return loadMemories(m.client, m.projectID)
	default:
		// Phase 3 placeholders.
		m.list.setPlaceholder("coming in Phase 3")
		return nil
	}
}

// loadDetailForSelection issues a detail fetch for whatever's currently
// highlighted in the list.
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
	m.detail.setLoading(true)

	switch m.list.cat {
	case CatTasks:
		return loadTaskDetail(m.client, sel.id)
	case CatMemories:
		return loadMemoryDetail(m.client, sel.id)
	}
	return nil
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		first := m.width == 0
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		if first {
			return m, m.loadForCategory()
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			// Cycle focus forward.
			m.focus = (m.focus + 1) % 3
			m.applyFocus()
			if m.focus == paneList {
				return m, m.loadDetailForSelection()
			}
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m, m.loadForCategory()

		case key.Matches(msg, m.keys.Back):
			// Move focus left toward the sidebar.
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
				m.sidebar.movePrev()
				return m, m.loadForCategory()
			case key.Matches(msg, m.keys.Down):
				m.sidebar.moveNext()
				return m, m.loadForCategory()
			case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter):
				m.focus = paneList
				m.applyFocus()
				return m, m.loadDetailForSelection()
			}

		case paneList:
			// Forward keys to the bubbles list (handles up/down/filter/etc).
			var cmd tea.Cmd
			m.list.list, cmd = m.list.list.Update(msg)
			// After cursor moves, refresh the detail pane.
			if key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) {
				return m, tea.Batch(cmd, m.loadDetailForSelection())
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
			m.list.setError(msg.err)
			return m, nil
		}
		m.list.setTasks(msg.items)
		// Auto-load detail for the first item.
		return m, m.loadDetailForSelection()

	case memoriesLoadedMsg:
		if msg.err != nil {
			m.list.setError(msg.err)
			return m, nil
		}
		m.list.setMemories(msg.items)
		return m, m.loadDetailForSelection()

	case taskDetailLoadedMsg:
		if msg.taskID != m.lastSelectedID {
			// Stale message from a previous selection — ignore.
			return m, nil
		}
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		m.detail.setContent(renderTaskDetail(
			msg.task, msg.subtasks, msg.annotations, msg.linkedMems, msg.comments,
		))
		return m, nil

	case memoryDetailLoadedMsg:
		if msg.memoryID != m.lastSelectedID {
			return m, nil
		}
		if msg.err != nil {
			m.detail.setError(msg.err)
			return m, nil
		}
		m.detail.setContent(renderMemoryDetail(
			msg.memory, msg.linkedTasks, msg.comments,
		))
		return m, nil
	}

	return m, nil
}

func (m rootModel) View() string {
	if m.width == 0 {
		return "loading…"
	}
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.sidebar.View(),
		m.list.View(),
		m.detail.View(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, body, m.renderStatusBar())
}

func (m rootModel) renderStatusBar() string {
	help := "h/← back · j/k up/dn · l/→ forward · ↵ open · ⇥ pane · / search · p project · r refresh · c copy · ? help · q quit"
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(help)
	if m.statusMsg != "" {
		bar = lipgloss.NewStyle().Foreground(display.ColorGood).Render(m.statusMsg) + "  " + bar
	}
	return bar
}

// Compile-time assertion: rootModel implements tea.Model.
var _ tea.Model = rootModel{}

// Silence unused warnings for models import in this file (used elsewhere).
var _ = models.Task{}
