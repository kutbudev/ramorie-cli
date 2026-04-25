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

// projectsLoadedMsg is fired after ListProjects completes. orgID is the filter
// used (empty string = all projects); used so the model can route a frame push
// back to the right place.
type projectsLoadedMsg struct {
	items []models.Project
	orgID string
	err   error
}

// projectDetailLoadedMsg is fired after the bundle of detail calls for a project.
type projectDetailLoadedMsg struct {
	projectID string
	project   *models.Project
	tasks     []models.Task
	memories  []models.Memory
	err       error
}

// orgsLoadedMsg is fired after ListOrganizations completes.
type orgsLoadedMsg struct {
	items []api.Organization
	err   error
}

// orgDetailLoadedMsg is fired after the bundle of detail calls for an org.
type orgDetailLoadedMsg struct {
	orgID      string
	org        *api.Organization
	projects   []models.Project
	encryption *api.OrgEncryptionStatus
	err        error
}

// activityLoadedMsg is fired after GetActivityHistory completes.
type activityLoadedMsg struct {
	items []models.ActivityItem
	err   error
}

// kanbanLoadedMsg is fired after the three parallel ListTasks calls (one per
// status bucket) complete.
type kanbanLoadedMsg struct {
	projectID  string
	todo       []models.Task
	inProgress []models.Task
	completed  []models.Task
	err        error
}

// profileLoadedMsg is fired after the 5-way parallel fan-out for the Profile
// pane completes. Soft-failures (e.g. agents endpoint 500) leave the field nil
// and surface only on the primary err for GetProfile.
type profileLoadedMsg struct {
	profile *models.UserProfile
	orgs    []api.Organization
	agents  []models.AgentProfile
	stats   *models.AgentEventStats
	oauth   []models.OAuthAccount
	err     error
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

// loadProjects fetches projects, optionally scoped to an org.
func loadProjects(c *api.Client, orgID string) tea.Cmd {
	return func() tea.Msg {
		var items []models.Project
		var err error
		if orgID == "" {
			items, err = c.ListProjects()
		} else {
			items, err = c.ListProjects(orgID)
		}
		return projectsLoadedMsg{items: items, orgID: orgID, err: err}
	}
}

// loadProjectDetail fans out GetProject + ListTasks + ListMemories.
func loadProjectDetail(c *api.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		var (
			wg   sync.WaitGroup
			p    *models.Project
			ts   []models.Task
			ms   []models.Memory
			err1 error
			err2 error
			err3 error
		)
		wg.Add(3)
		go func() { defer wg.Done(); p, err1 = c.GetProject(projectID) }()
		go func() { defer wg.Done(); ts, err2 = c.ListTasks(projectID, "") }()
		go func() { defer wg.Done(); ms, err3 = c.ListMemories(projectID, "") }()
		wg.Wait()
		// Soft-fail tasks/memories — only surface GetProject errors.
		_ = err2
		_ = err3
		return projectDetailLoadedMsg{
			projectID: projectID,
			project:   p,
			tasks:     ts,
			memories:  ms,
			err:       err1,
		}
	}
}

// loadOrgs fetches all organizations the user is a member of.
func loadOrgs(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		items, err := c.ListOrganizations()
		return orgsLoadedMsg{items: items, err: err}
	}
}

// loadOrgDetail fans out GetOrganization + ListProjects(orgID) +
// GetOrgEncryptionStatus.
func loadOrgDetail(c *api.Client, orgID string) tea.Cmd {
	return func() tea.Msg {
		var (
			wg   sync.WaitGroup
			org  *api.Organization
			ps   []models.Project
			enc  *api.OrgEncryptionStatus
			err1 error
			err2 error
			err3 error
		)
		wg.Add(3)
		go func() { defer wg.Done(); org, err1 = c.GetOrganization(orgID) }()
		go func() { defer wg.Done(); ps, err2 = c.ListProjects(orgID) }()
		go func() { defer wg.Done(); enc, err3 = c.GetOrgEncryptionStatus(orgID) }()
		wg.Wait()
		// Soft-fail projects + encryption status.
		_ = err2
		_ = err3
		return orgDetailLoadedMsg{
			orgID:      orgID,
			org:        org,
			projects:   ps,
			encryption: enc,
			err:        err1,
		}
	}
}

// loadActivity fetches the recent activity feed.
func loadActivity(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		items, err := c.GetActivityHistory(7, 50, "")
		return activityLoadedMsg{items: items, err: err}
	}
}

// loadKanban fans out three parallel ListTasks calls (one per status bucket)
// for the given project.
func loadKanban(c *api.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		if projectID == "" {
			return kanbanLoadedMsg{projectID: "", err: nil}
		}
		var (
			wg   sync.WaitGroup
			todo []models.Task
			ip   []models.Task
			done []models.Task
			err1 error
			err2 error
			err3 error
		)
		wg.Add(3)
		go func() { defer wg.Done(); todo, err1 = c.ListTasks(projectID, "TODO") }()
		go func() { defer wg.Done(); ip, err2 = c.ListTasks(projectID, "IN_PROGRESS") }()
		go func() { defer wg.Done(); done, err3 = c.ListTasks(projectID, "COMPLETED") }()
		wg.Wait()
		// Surface the first non-nil error.
		err := err1
		if err == nil {
			err = err2
		}
		if err == nil {
			err = err3
		}
		return kanbanLoadedMsg{
			projectID:  projectID,
			todo:       todo,
			inProgress: ip,
			completed:  done,
			err:        err,
		}
	}
}

// loadProfile fans out 5 parallel calls. Only GetProfile is treated as a hard
// dependency — secondary endpoints (agents, oauth, stats, orgs) degrade
// silently to empty / nil so the page still renders.
func loadProfile(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		var (
			wg     sync.WaitGroup
			prof   *models.UserProfile
			orgs   []api.Organization
			agents []models.AgentProfile
			stats  *models.AgentEventStats
			oauth  []models.OAuthAccount
			err1   error
			err2   error
			err3   error
			err4   error
			err5   error
		)
		wg.Add(5)
		go func() { defer wg.Done(); prof, err1 = c.GetProfile() }()
		go func() { defer wg.Done(); orgs, err2 = c.ListOrganizations() }()
		go func() { defer wg.Done(); agents, err3 = c.ListAgents() }()
		go func() { defer wg.Done(); stats, err4 = c.GetAgentEventStats() }()
		go func() { defer wg.Done(); oauth, err5 = c.ListOAuthAccounts() }()
		wg.Wait()
		_ = err2
		_ = err3
		_ = err4
		_ = err5
		return profileLoadedMsg{
			profile: prof,
			orgs:    orgs,
			agents:  agents,
			stats:   stats,
			oauth:   oauth,
			err:     err1,
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
