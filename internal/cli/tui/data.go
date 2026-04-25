package tui

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// ---- Messages -------------------------------------------------------------

// tasksLoadedMsg is fired after ListTasks completes.
type tasksLoadedMsg struct {
	items []models.Task
	err   error
}

// memoriesLoadedMsg is fired after ListMemories completes.
type memoriesLoadedMsg struct {
	items []models.Memory
	err   error
}

// taskDetailLoadedMsg is fired after the bundle of detail calls for a task.
type taskDetailLoadedMsg struct {
	taskID      string
	task        *models.Task
	subtasks    []models.Subtask
	annotations []models.Annotation
	linkedMems  []models.Memory
	comments    []models.Comment
	err         error
}

// memoryDetailLoadedMsg is fired after the bundle of detail calls for a memory.
type memoryDetailLoadedMsg struct {
	memoryID    string
	memory      *models.Memory
	linkedTasks []models.Task
	comments    []models.Comment
	err         error
}

// ---- Loaders --------------------------------------------------------------

// loadTasks returns a tea.Cmd that fetches tasks for projectID
// (empty string = all projects).
func loadTasks(c *api.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		items, err := c.ListTasks(projectID, "")
		return tasksLoadedMsg{items: items, err: err}
	}
}

// loadMemories returns a tea.Cmd that fetches memories for projectID.
func loadMemories(c *api.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		items, err := c.ListMemories(projectID, "")
		return memoriesLoadedMsg{items: items, err: err}
	}
}

// loadTaskDetail fans out the four detail-related calls in parallel and
// folds them into one message.
func loadTaskDetail(c *api.Client, taskID string) tea.Cmd {
	return func() tea.Msg {
		var (
			wg   sync.WaitGroup
			task *models.Task
			subs []models.Subtask
			ann  []models.Annotation
			mems []models.Memory
			cmts []models.Comment
			err1 error
			err2 error
			err3 error
			err4 error
			err5 error
		)
		wg.Add(5)
		go func() {
			defer wg.Done()
			task, err1 = c.GetTask(taskID)
		}()
		go func() {
			defer wg.Done()
			subs, err2 = c.ListSubtasks(taskID)
		}()
		go func() {
			defer wg.Done()
			ann, err3 = c.ListAnnotations(taskID)
		}()
		go func() {
			defer wg.Done()
			mems, err4 = c.ListTaskMemories(taskID)
		}()
		go func() {
			defer wg.Done()
			cmts, err5 = c.ListEntityComments("task", taskID)
		}()
		wg.Wait()

		// Primary error is the GetTask one; the rest degrade to empty lists.
		err := err1
		if err == nil {
			// Surface a non-nil secondary error only if we'd otherwise show
			// nothing for this task. For now just prefer the primary.
			_ = err2
			_ = err3
			_ = err4
			_ = err5
		}

		return taskDetailLoadedMsg{
			taskID:      taskID,
			task:        task,
			subtasks:    subs,
			annotations: ann,
			linkedMems:  mems,
			comments:    cmts,
			err:         err,
		}
	}
}

// loadMemoryDetail fans out the three detail-related calls.
func loadMemoryDetail(c *api.Client, memoryID string) tea.Cmd {
	return func() tea.Msg {
		var (
			wg   sync.WaitGroup
			mem  *models.Memory
			tks  []models.Task
			cmts []models.Comment
			err1 error
			err2 error
			err3 error
		)
		wg.Add(3)
		go func() {
			defer wg.Done()
			mem, err1 = c.GetMemory(memoryID)
		}()
		go func() {
			defer wg.Done()
			tks, err2 = c.ListMemoryTasks(memoryID)
		}()
		go func() {
			defer wg.Done()
			cmts, err3 = c.ListEntityComments("memory", memoryID)
		}()
		wg.Wait()
		_ = err2
		_ = err3
		return memoryDetailLoadedMsg{
			memoryID:    memoryID,
			memory:      mem,
			linkedTasks: tks,
			comments:    cmts,
			err:         err1,
		}
	}
}

// ---- Helpers --------------------------------------------------------------

// decryptTask returns plaintext title/description for a task, mirroring
// internal/cli/commands.decryptTaskForCLI. Duplicated here to avoid a
// commands→tui reverse import; refactor candidate for Phase 3.
func decryptTask(t *models.Task) (title, description string) {
	if t == nil {
		return "", ""
	}
	if !t.IsEncrypted {
		return t.Title, t.Description
	}
	if !crypto.IsVaultUnlocked() {
		title = "[Vault Locked]"
		description = "[Vault Locked]"
		if t.Title != "" && t.Title != "[Encrypted]" {
			title = t.Title
		}
		if t.Description != "" && t.Description != "[Encrypted]" {
			description = t.Description
		}
		return title, description
	}
	if t.EncryptedTitle != "" {
		dec, err := crypto.DecryptContent(t.EncryptedTitle, t.TitleNonce, true)
		if err != nil {
			title = "[Decryption Failed]"
		} else {
			title = dec
		}
	} else {
		title = t.Title
	}
	if t.EncryptedDescription != "" {
		dec, err := crypto.DecryptContent(t.EncryptedDescription, t.DescriptionNonce, true)
		if err != nil {
			description = "[Decryption Failed]"
		} else {
			description = dec
		}
	} else {
		description = t.Description
	}
	return title, description
}

// decryptMemoryContent returns plaintext content for a memory.
func decryptMemoryContent(m *models.Memory) string {
	if m == nil {
		return ""
	}
	if !m.IsEncrypted {
		return m.Content
	}
	if !crypto.IsVaultUnlocked() {
		if m.Content != "" && m.Content != "[Encrypted]" {
			return m.Content
		}
		return "[Vault Locked]"
	}
	if m.EncryptedContent != "" {
		dec, err := crypto.DecryptContent(m.EncryptedContent, m.ContentNonce, true)
		if err != nil {
			return "[Decryption Failed]"
		}
		return dec
	}
	return m.Content
}
