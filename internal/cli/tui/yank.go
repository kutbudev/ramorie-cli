// Package tui — yank.go: serializes the currently-open right-pane entity as
// full markdown and writes it to the system clipboard. No truncation; the
// output is meant to be pasted into a chat, doc, or LLM prompt as ground
// truth.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/google/uuid"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// yankResult is the outcome of a copy attempt — surfaced to the status bar.
type yankResult struct {
	chars int
	err   error
}

// yankCurrent serializes the right-pane entity to markdown and writes it
// to the system clipboard. Returns chars written + any error (clipboard
// binary missing, no entity loaded, etc.). Never panics.
func (m *rootModel) yankCurrent() yankResult {
	md, err := m.serializeCurrent()
	if err != nil {
		return yankResult{0, err}
	}
	if err := clipboard.WriteAll(md); err != nil {
		return yankResult{0, fmt.Errorf("clipboard not available — install xclip/xsel/pbcopy: %w", err)}
	}
	return yankResult{chars: len(md)}
}

// serializeCurrent picks the right serializer based on the active sidebar
// category and the entity currently loaded into the detail pane. Returns
// a helpful error if nothing is loaded or if we're in a view (kanban) that
// doesn't represent a single entity.
func (m *rootModel) serializeCurrent() (string, error) {
	cat := m.sidebar.selected()
	switch cat {
	case CatTasks:
		if m.yankTask == nil {
			return "", fmt.Errorf("no task loaded")
		}
		return serializeTask(
			m.yankTask, m.yankTaskSubtasks, m.yankTaskNotes,
			m.yankTaskMems, m.yankTaskComments,
		), nil
	case CatMemories:
		if m.yankMemory == nil {
			return "", fmt.Errorf("no memory loaded")
		}
		return serializeMemory(
			m.yankMemory, m.yankMemoryTasks, m.yankMemoryComments,
		), nil
	case CatProjects:
		if m.yankProject == nil {
			return "", fmt.Errorf("no project loaded")
		}
		return serializeProject(
			m.yankProject, m.yankProjectTasks, m.yankProjectMems,
		), nil
	case CatOrganizations:
		if m.yankOrg == nil {
			return "", fmt.Errorf("no organization loaded")
		}
		return serializeOrg(
			m.yankOrg, m.yankOrgProjects, m.yankOrgEncryption,
		), nil
	case CatActivity:
		if m.yankActivity == nil {
			return "", fmt.Errorf("no activity item loaded")
		}
		return serializeActivity(*m.yankActivity), nil
	case CatProfile:
		if m.profile == nil || m.profile.profile == nil {
			return "", fmt.Errorf("profile not loaded")
		}
		return serializeProfile(
			m.profile.profile, m.profile.orgs, m.profile.agents,
			m.profile.stats, m.profile.oauth,
		), nil
	case CatKanban:
		return "", fmt.Errorf("kanban view doesn't support copy — focus a single task first")
	}
	return "", fmt.Errorf("unknown category")
}

// --- per-entity serializers -----------------------------------------------

// serializeTask renders the full markdown for a task. No truncation.
func serializeTask(
	t *models.Task,
	subs []models.Subtask,
	notes []models.Annotation,
	mems []models.Memory,
	comments []models.Comment,
) string {
	title, desc := decryptTask(t)
	var b strings.Builder

	fmt.Fprintf(&b, "# [task] %s\n\n", title)
	fmt.Fprintf(&b, "- ID: %s\n", t.ID.String())
	fmt.Fprintf(&b, "- Status: %s\n", t.Status)
	fmt.Fprintf(&b, "- Priority: %s\n", t.Priority)
	if t.Project != nil && t.Project.Name != "" {
		fmt.Fprintf(&b, "- Project: %s\n", t.Project.Name)
	} else if t.ProjectID != uuid.Nil {
		fmt.Fprintf(&b, "- Project: %s\n", t.ProjectID.String())
	}
	fmt.Fprintf(&b, "- Created: %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Updated: %s\n", t.UpdatedAt.Format("2006-01-02 15:04"))
	if tags := tagSlice(t.Tags); len(tags) > 0 {
		fmt.Fprintf(&b, "- Tags: %s\n", strings.Join(tags, ", "))
	}
	b.WriteString("\n")

	if d := strings.TrimSpace(desc); d != "" {
		b.WriteString("## Description\n\n")
		b.WriteString(d)
		b.WriteString("\n\n")
	}

	if len(subs) > 0 {
		completed := 0
		for _, s := range subs {
			if s.Completed > 0 || strings.EqualFold(s.Status, "COMPLETED") {
				completed++
			}
		}
		fmt.Fprintf(&b, "## Subtasks (%d/%d)\n\n", completed, len(subs))
		for _, s := range subs {
			done := s.Completed > 0 || strings.EqualFold(s.Status, "COMPLETED")
			mark := " "
			if done {
				mark = "x"
			}
			fmt.Fprintf(&b, "- [%s] %s\n", mark, s.Description)
		}
		b.WriteString("\n")
	}

	if len(notes) > 0 {
		fmt.Fprintf(&b, "## Notes (%d)\n\n", len(notes))
		for _, n := range notes {
			fmt.Fprintf(&b, "**%s** — %s\n\n",
				n.CreatedAt.Format("2006-01-02 15:04"), decryptAnnotation(&n))
		}
	}

	if len(mems) > 0 {
		fmt.Fprintf(&b, "## Linked Memories (%d)\n\n", len(mems))
		for i := range mems {
			mm := &mems[i]
			short := shortID(mm.ID.String())
			content := decryptMemoryContent(mm)
			firstLine := strings.SplitN(strings.TrimSpace(content), "\n", 2)[0]
			if len(firstLine) > 80 {
				firstLine = firstLine[:77] + "..."
			}
			fmt.Fprintf(&b, "- [%s] %s %s\n",
				strings.ToLower(mm.Type), short, firstLine)
		}
		b.WriteString("\n")
	}

	if len(comments) > 0 {
		fmt.Fprintf(&b, "## Comments (%d)\n\n", len(comments))
		writeCommentTreeMD(&b, comments)
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// serializeMemory renders the full markdown for a memory.
func serializeMemory(
	mm *models.Memory,
	linkedTasks []models.Task,
	comments []models.Comment,
) string {
	content := decryptMemoryContent(mm)
	heading := strings.SplitN(strings.TrimSpace(content), "\n", 2)[0]
	if len(heading) > 100 {
		heading = heading[:97] + "..."
	}
	if heading == "" {
		heading = "(empty)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# [memory · %s] %s\n\n", mm.Type, heading)
	fmt.Fprintf(&b, "- ID: %s\n", mm.ID.String())
	fmt.Fprintf(&b, "- Type: %s\n", mm.Type)
	if mm.Project != nil && mm.Project.Name != "" {
		fmt.Fprintf(&b, "- Project: %s\n", mm.Project.Name)
	} else if mm.ProjectID != uuid.Nil {
		fmt.Fprintf(&b, "- Project: %s\n", mm.ProjectID.String())
	}
	if tags := tagSlice(mm.Tags); len(tags) > 0 {
		fmt.Fprintf(&b, "- Tags: %s\n", strings.Join(tags, ", "))
	}
	fmt.Fprintf(&b, "- Access count: %d\n", mm.AccessCount)
	fmt.Fprintf(&b, "- Created: %s\n", mm.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Updated: %s\n", mm.UpdatedAt.Format("2006-01-02 15:04"))
	b.WriteString("\n")

	if c := strings.TrimSpace(content); c != "" {
		b.WriteString("## Content\n\n")
		b.WriteString(c)
		b.WriteString("\n\n")
	}

	if len(comments) > 0 {
		fmt.Fprintf(&b, "## Comments (%d)\n\n", len(comments))
		writeCommentTreeMD(&b, comments)
	}

	if len(linkedTasks) > 0 {
		fmt.Fprintf(&b, "## Linked Tasks (%d)\n\n", len(linkedTasks))
		for i := range linkedTasks {
			t := &linkedTasks[i]
			title, _ := decryptTask(t)
			fmt.Fprintf(&b, "- %s %s %s\n",
				statusGlyph(t.Status), shortID(t.ID.String()),
				strings.SplitN(title, "\n", 2)[0])
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// serializeProject renders the full markdown for a project.
func serializeProject(
	p *models.Project,
	tasks []models.Task,
	mems []models.Memory,
) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# [project] %s\n\n", p.Name)
	fmt.Fprintf(&b, "- ID: %s\n", p.ID.String())
	if p.Organization != nil && p.Organization.Name != "" {
		fmt.Fprintf(&b, "- Org: %s\n", p.Organization.Name)
	} else {
		fmt.Fprintf(&b, "- Org: Personal\n")
	}
	fmt.Fprintf(&b, "- Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Updated: %s\n", p.UpdatedAt.Format("2006-01-02 15:04"))
	b.WriteString("\n")

	if d := strings.TrimSpace(p.Description); d != "" {
		b.WriteString("## Description\n\n")
		b.WriteString(d)
		b.WriteString("\n\n")
	}

	const cap = 20
	tShown := len(tasks)
	if tShown > cap {
		tShown = cap
	}
	if len(tasks) > 0 {
		fmt.Fprintf(&b, "## Recent Tasks (%d shown of %d)\n\n", tShown, len(tasks))
		for i := 0; i < tShown; i++ {
			t := tasks[i]
			title, _ := decryptTask(&t)
			fmt.Fprintf(&b, "- %s [%s] %s %s\n",
				statusGlyph(t.Status), t.Priority, shortID(t.ID.String()),
				strings.SplitN(title, "\n", 2)[0])
		}
		b.WriteString("\n")
	}

	mShown := len(mems)
	if mShown > cap {
		mShown = cap
	}
	if len(mems) > 0 {
		fmt.Fprintf(&b, "## Recent Memories (%d shown of %d)\n\n", mShown, len(mems))
		for i := 0; i < mShown; i++ {
			mm := mems[i]
			content := decryptMemoryContent(&mm)
			first := strings.SplitN(strings.TrimSpace(content), "\n", 2)[0]
			if len(first) > 80 {
				first = first[:77] + "..."
			}
			fmt.Fprintf(&b, "- [%s] %s %s\n",
				strings.ToLower(mm.Type), shortID(mm.ID.String()), first)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// serializeOrg renders the full markdown for an organization.
func serializeOrg(
	o *api.Organization,
	projs []models.Project,
	enc *api.OrgEncryptionStatus,
) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# [org] %s\n\n", o.Name)
	fmt.Fprintf(&b, "- ID: %s\n", o.ID)
	if o.Description != "" {
		fmt.Fprintf(&b, "- Description: %s\n", o.Description)
	}
	if o.OwnerID != "" {
		fmt.Fprintf(&b, "- Owner: %s\n", o.OwnerID)
	} else {
		fmt.Fprintf(&b, "- Owner: (unknown)\n")
	}
	if enc != nil {
		if enc.IsEnabled {
			fmt.Fprintf(&b, "- Encryption: ✓ Enabled (v%d)\n", enc.EncryptionVersion)
		} else {
			b.WriteString("- Encryption: × Disabled\n")
		}
	}
	fmt.Fprintf(&b, "- Created: %s\n", o.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Updated: %s\n", o.UpdatedAt.Format("2006-01-02 15:04"))
	b.WriteString("\n")

	if len(projs) > 0 {
		fmt.Fprintf(&b, "## Projects (%d)\n\n", len(projs))
		for _, p := range projs {
			fmt.Fprintf(&b, "- %s %s\n", shortID(p.ID.String()), p.Name)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// serializeActivity renders the full markdown for one activity feed item.
func serializeActivity(a models.ActivityItem) string {
	var b strings.Builder

	first := strings.SplitN(strings.TrimSpace(a.Summary), "\n", 2)[0]
	if first == "" {
		first = "no summary"
	}
	fmt.Fprintf(&b, "# [activity · %s] %s\n\n", a.EntityType, first)
	fmt.Fprintf(&b, "- Entity ID: %s\n", a.EntityID.String())
	fmt.Fprintf(&b, "- Entity Type: %s\n", a.EntityType)
	if a.ProjectID != nil {
		fmt.Fprintf(&b, "- Project ID: %s\n", a.ProjectID.String())
	}
	fmt.Fprintf(&b, "- Timestamp: %s\n", a.Timestamp.Format("2006-01-02 15:04:05"))
	b.WriteString("\n")

	if s := strings.TrimSpace(a.Summary); s != "" {
		b.WriteString("## Summary\n\n")
		b.WriteString(s)
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// serializeProfile renders the full markdown for the user profile bundle.
func serializeProfile(
	p *models.UserProfile,
	orgs []api.Organization,
	agents []models.AgentProfile,
	stats *models.AgentEventStats,
	oauth []models.OAuthAccount,
) string {
	var b strings.Builder

	fullName := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if fullName == "" {
		fullName = "(unnamed)"
	}
	fmt.Fprintf(&b, "# [profile] %s <%s>\n\n", fullName, p.Email)
	fmt.Fprintf(&b, "- ID: %s\n", p.ID)
	if p.APIKey != "" {
		fmt.Fprintf(&b, "- API Key (masked): %s\n", maskAPIKey(p.APIKey))
	}
	if p.ActiveOrganizationID != nil && *p.ActiveOrganizationID != "" {
		fmt.Fprintf(&b, "- Active Org: %s\n", *p.ActiveOrganizationID)
	}
	if p.CreatedAt != nil {
		fmt.Fprintf(&b, "- Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04"))
	}
	if p.UpdatedAt != nil {
		fmt.Fprintf(&b, "- Updated: %s\n", p.UpdatedAt.Format("2006-01-02 15:04"))
	}
	b.WriteString("\n")

	if len(orgs) > 0 {
		fmt.Fprintf(&b, "## Organizations (%d)\n\n", len(orgs))
		for _, o := range orgs {
			fmt.Fprintf(&b, "- %s\n", o.Name)
		}
		b.WriteString("\n")
	}

	// Top agents from stats first, fall back to ListAgents.
	if stats != nil && len(stats.TopAgents) > 0 {
		fmt.Fprintf(&b, "## Top Agents (%d)\n\n", len(stats.TopAgents))
		for _, a := range stats.TopAgents {
			label := a.DisplayName
			if label == "" {
				label = a.AgentName
			}
			active := "× inactive"
			if a.IsActive {
				active = "✓ active"
			}
			last := ""
			if a.LastEventAt != nil {
				last = " · last " + a.LastEventAt.Format("2006-01-02 15:04")
			}
			typ := a.AgentType
			if typ == "" {
				typ = "agent"
			}
			fmt.Fprintf(&b, "- [%s] %s · %d events%s · %s\n",
				typ, label, a.EventCount, last, active)
		}
		b.WriteString("\n")
	} else if len(agents) > 0 {
		fmt.Fprintf(&b, "## Agents (%d)\n\n", len(agents))
		for _, a := range agents {
			label := a.DisplayName
			if label == "" {
				label = a.AgentName
			}
			active := "× inactive"
			if a.IsActive {
				active = "✓ active"
			}
			last := ""
			if a.LastEventAt != nil {
				last = " · last " + a.LastEventAt.Format("2006-01-02 15:04")
			}
			fmt.Fprintf(&b, "- [%s] %s · %d events%s · %s\n",
				a.AgentType, label, a.TotalEvents, last, active)
		}
		b.WriteString("\n")
	}

	if stats != nil {
		b.WriteString("## Stats\n\n")
		fmt.Fprintf(&b, "- Total events: %d\n", stats.TotalEvents)
		fmt.Fprintf(&b, "- Last 24h: %d\n", stats.EventsLast24h)
		fmt.Fprintf(&b, "- Last 7d: %d\n", stats.EventsLast7d)
		if len(stats.EventsByType) > 0 {
			fmt.Fprintf(&b, "- By type: %s\n", formatCountMap(stats.EventsByType))
		}
		if len(stats.EventsBySource) > 0 {
			fmt.Fprintf(&b, "- By source: %s\n", formatCountMap(stats.EventsBySource))
		}
		b.WriteString("\n")
	}

	if len(oauth) > 0 {
		fmt.Fprintf(&b, "## OAuth Accounts (%d)\n\n", len(oauth))
		for _, o := range oauth {
			line := "- " + o.Provider
			if o.Email != "" {
				line += " · " + o.Email
			} else if o.Username != "" {
				line += " · " + o.Username
			}
			if o.LastLoginAt != nil {
				line += " · last login " + o.LastLoginAt.Format("2006-01-02")
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// --- helpers ---------------------------------------------------------------

// statusGlyph maps a task status to a 1-char marker for inline list lines.
func statusGlyph(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED", "DONE":
		return "✓"
	case "IN_PROGRESS":
		return "◐"
	case "TODO":
		return "○"
	default:
		return "·"
	}
}

// writeCommentTreeMD writes top-level comments + replies as markdown.
// Replies are rendered as blockquote (>) blocks, mirroring the spec.
func writeCommentTreeMD(b *strings.Builder, comments []models.Comment) {
	childrenByParent := map[string][]models.Comment{}
	tops := []models.Comment{}
	for _, c := range comments {
		if c.ParentID != nil && *c.ParentID != "" {
			childrenByParent[*c.ParentID] = append(childrenByParent[*c.ParentID], c)
		} else {
			tops = append(tops, c)
		}
	}

	for i, c := range tops {
		writeOneCommentMD(b, c, false)
		replies := c.Replies
		if len(replies) == 0 {
			replies = childrenByParent[c.ID]
		}
		// Stable order — server may not sort.
		sort.SliceStable(replies, func(i, j int) bool {
			return replies[i].CreatedAt.Before(replies[j].CreatedAt)
		})
		for _, r := range replies {
			b.WriteString("\n")
			writeOneCommentMD(b, r, true)
		}
		if i < len(tops)-1 {
			b.WriteString("\n")
		}
	}
}

// writeOneCommentMD writes one comment header + body. nested=true prefixes
// every line with "> " so the comment renders as a blockquote (markdown
// reply quoting).
func writeOneCommentMD(b *strings.Builder, c models.Comment, nested bool) {
	author := commentAuthor(c)
	stamp := c.CreatedAt.Format("2006-01-02 15:04")

	prefix := ""
	if nested {
		prefix = "> "
	}

	fmt.Fprintf(b, "%s**%s · %s**\n", prefix, author, stamp)
	if nested {
		b.WriteString(">\n")
	} else {
		b.WriteString("\n")
	}
	for _, line := range strings.Split(strings.TrimRight(decryptComment(&c), "\n"), "\n") {
		fmt.Fprintf(b, "%s%s\n", prefix, line)
	}
}
