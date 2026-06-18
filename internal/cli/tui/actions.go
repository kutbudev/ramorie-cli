package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kutbudev/ramorie-cli/internal/api"
)

// actions.go holds the write-side and recall commands that make the TUI
// interactive: completing / starting / deleting tasks, deleting memories,
// creating tasks & memories, and the backend hybrid recall. Each is a
// tea.Cmd that performs one API call off the UI goroutine and reports back
// through a typed message that rootModel.Update folds in (usually by
// refreshing the active category).

// actionDoneMsg reports the result of a mutating action. ok=false carries an
// error; refresh asks rootModel to reload the current category afterwards.
type actionDoneMsg struct {
	verb    string // human verb for the status line, e.g. "completed"
	ok      bool
	err     error
	refresh bool
}

func actionOK(verb string) tea.Msg { return actionDoneMsg{verb: verb, ok: true, refresh: true} }
func actionErr(e error) tea.Msg    { return actionDoneMsg{ok: false, err: e} }

func completeTaskCmd(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.CompleteTask(id); err != nil {
			return actionErr(err)
		}
		return actionOK("completed")
	}
}

func reopenTaskCmd(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if _, err := c.UpdateTask(id, map[string]interface{}{"status": "TODO"}); err != nil {
			return actionErr(err)
		}
		return actionOK("reopened")
	}
}

func startTaskCmd(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.StartTask(id); err != nil {
			return actionErr(err)
		}
		return actionOK("started")
	}
}

func deleteTaskCmd(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteTask(id); err != nil {
			return actionErr(err)
		}
		return actionOK("deleted")
	}
}

func deleteMemoryCmd(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteMemory(id); err != nil {
			return actionErr(err)
		}
		return actionOK("deleted")
	}
}

func createTaskCmd(c *api.Client, projectID, title string) tea.Cmd {
	return func() tea.Msg {
		if _, err := c.CreateTask(projectID, title, "", "M"); err != nil {
			return actionErr(err)
		}
		return actionOK("task created")
	}
}

func createMemoryCmd(c *api.Client, projectID, content string) tea.Cmd {
	return func() tea.Msg {
		if _, err := c.CreateMemoryWithType(projectID, content, "general"); err != nil {
			return actionErr(err)
		}
		return actionOK("memory created")
	}
}

// recallLoadedMsg carries hybrid-recall results back to the list pane.
type recallLoadedMsg struct {
	term  string
	items []api.FindItem
	err   error
}

func recallCmd(c *api.Client, term, projectID string) tea.Cmd {
	return func() tea.Msg {
		resp, err := c.FindMemories(api.FindMemoriesOptions{
			Term:    term,
			Project: projectID,
			Limit:   30,
		})
		if err != nil {
			return recallLoadedMsg{term: term, err: err}
		}
		return recallLoadedMsg{term: term, items: resp.Items}
	}
}
