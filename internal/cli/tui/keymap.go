package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap collects all the bindings the TUI listens for. Centralized so the
// help bar and Update dispatch agree on what's bound where.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Enter   key.Binding
	Back    key.Binding
	Tab     key.Binding
	Search  key.Binding
	Project key.Binding
	Refresh key.Binding
	Yank    key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "back")),
		Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "forward")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "open")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("⇥", "next pane")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Project: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "project")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Yank:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
