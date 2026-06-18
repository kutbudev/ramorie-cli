package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap collects all the bindings the TUI listens for. Centralized so the
// help bar and Update dispatch agree on what's bound where.
type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Back        key.Binding
	Tab         key.Binding
	PrevTab     key.Binding
	PrevPage    key.Binding
	NextPage    key.Binding
	Top         key.Binding
	Bottom      key.Binding
	Search      key.Binding
	Project     key.Binding
	AllProjects key.Binding
	Refresh     key.Binding
	Yank        key.Binding
	Theme       key.Binding
	Help        key.Binding
	Quit        key.Binding
	New         key.Binding // create task / memory
	Toggle      key.Binding // complete / reopen task
	StartTask   key.Binding // mark task in-progress
	Delete      key.Binding // delete task / memory (confirmed)
	Recall      key.Binding // backend hybrid find/recall
	Cat1        key.Binding
	Cat2        key.Binding
	Cat3        key.Binding
	Cat4        key.Binding
	Cat5        key.Binding
	Cat6        key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "back")),
		Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "forward")),
		Enter:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "open")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Tab:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("⇥", "next pane")),
		PrevTab:     key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-⇥", "prev pane")),
		PrevPage:    key.NewBinding(key.WithKeys("pgup", "ctrl+u", "u", "b"), key.WithHelp("pgup/^u", "page up")),
		NextPage:    key.NewBinding(key.WithKeys("pgdown", "ctrl+d", "d", "f"), key.WithHelp("pgdn/^d", "page down")),
		Top:         key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "top")),
		Bottom:      key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "bottom")),
		Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Project:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "project")),
		AllProjects: key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "all projects")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Yank:        key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		Theme:       key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		New:         key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Toggle:      key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "complete")),
		StartTask:   key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "start")),
		Delete:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete")),
		Recall:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "recall")),
		Cat1:        key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "tasks")),
		Cat2:        key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "memories")),
		Cat3:        key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "projects")),
		Cat4:        key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "orgs")),
		Cat5:        key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "activity")),
		Cat6:        key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "profile")),
	}
}
