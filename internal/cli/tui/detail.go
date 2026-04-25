package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// detailModel is the right pane — a scrollable rendering of the currently
// selected entity.
type detailModel struct {
	vp      viewport.Model
	width   int
	height  int
	focused bool
	loading bool
	content string
	errMsg  string
}

func newDetail(width, height int) detailModel {
	vp := viewport.New(maxInt(width-4, 10), maxInt(height-4, 3))
	return detailModel{vp: vp, width: width, height: height}
}

func (d *detailModel) resize(width, height int) {
	d.width = width
	d.height = height
	d.vp.Width = maxInt(width-4, 10)
	d.vp.Height = maxInt(height-4, 3)
	d.vp.SetContent(d.content)
}

func (d *detailModel) setContent(s string) {
	d.content = s
	d.vp.SetContent(s)
	d.vp.GotoTop()
	d.loading = false
	d.errMsg = ""
}

func (d *detailModel) setLoading(b bool) { d.loading = b }
func (d *detailModel) setError(err error) {
	d.errMsg = err.Error()
	d.loading = false
}

// View renders the bordered detail pane.
func (d detailModel) View() string {
	border := lipgloss.NormalBorder()
	borderColor := lipgloss.Color("240")
	if d.focused {
		borderColor = display.ColorAccent
	}
	container := lipgloss.NewStyle().
		Width(maxInt(d.width-2, 1)).
		Height(maxInt(d.height-2, 1)).
		BorderStyle(border).
		BorderForeground(borderColor)

	var inner string
	switch {
	case d.loading:
		inner = display.Dim.Render("Loading…")
	case d.errMsg != "":
		inner = display.Err.Render(d.errMsg)
	case d.content == "":
		inner = display.Dim.Render("(nothing selected)")
	default:
		inner = d.vp.View()
	}
	return container.Render(inner)
}

// ---- Render helpers -------------------------------------------------------

// shortID returns the first 8 chars of an id, or the whole thing if shorter.
func shortID(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// sectionHead renders a "Title\n───" two-line header used by every detail
// view to chunk content.
func sectionHead(title string) string {
	return display.Label.Render(title) + "\n" + display.Dim.Render("───")
}

// renderTaskDetail builds the rich detail block for a task: header, metadata,
// description, subtasks, notes, linked memories and comments.
func renderTaskDetail(
	t *models.Task,
	subtasks []models.Subtask,
	annotations []models.Annotation,
	linkedMems []models.Memory,
	comments []models.Comment,
) string {
	if t == nil {
		return display.Dim.Render("(no task)")
	}
	title, desc := decryptTask(t)
	var b strings.Builder

	// Header line: [task] <id>   <priority> · <status>
	b.WriteString(display.Dim.Render("[task]"))
	b.WriteString(" ")
	b.WriteString(display.Dim.Render(shortID(t.ID.String())))
	b.WriteString("    ")
	b.WriteString(display.PriorityBadge(t.Priority))
	b.WriteString(" · ")
	b.WriteString(display.StatusLabel(t.Status))
	b.WriteString("\n")
	b.WriteString(display.Title.Render(display.SingleLine(title)))
	b.WriteString("\n\n")

	// Metadata block.
	if t.Project != nil && t.Project.Name != "" {
		b.WriteString(display.Label.Render("Project: "))
		b.WriteString(t.Project.Name)
		b.WriteString("\n")
	}
	b.WriteString(display.Label.Render("Created: "))
	b.WriteString(display.Relative(t.CreatedAt))
	b.WriteString(" · ")
	b.WriteString(display.Label.Render("Updated: "))
	b.WriteString(display.Relative(t.UpdatedAt))
	b.WriteString("\n")
	if tags := tagSlice(t.Tags); len(tags) > 0 {
		b.WriteString(display.Label.Render("Tags: "))
		b.WriteString(strings.Join(tags, " · "))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Description.
	if desc != "" {
		b.WriteString(sectionHead("Description"))
		b.WriteString("\n")
		b.WriteString(desc)
		b.WriteString("\n\n")
	}

	// Subtasks.
	if len(subtasks) > 0 {
		done := 0
		for _, s := range subtasks {
			if s.Completed == 1 || strings.EqualFold(s.Status, "COMPLETED") {
				done++
			}
		}
		b.WriteString(sectionHead(fmt.Sprintf("Subtasks (%d/%d)", done, len(subtasks))))
		b.WriteString("\n")
		for _, s := range subtasks {
			marker := "○"
			if s.Completed == 1 || strings.EqualFold(s.Status, "COMPLETED") {
				marker = display.Good.Render("✓")
			} else {
				marker = display.Dim.Render("○")
			}
			b.WriteString(marker)
			b.WriteString(" ")
			b.WriteString(s.Description)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Notes (annotations).
	if len(annotations) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("Notes (%d)", len(annotations))))
		b.WriteString("\n")
		for _, a := range annotations {
			ts := a.CreatedAt.Format("2006-01-02 15:04")
			b.WriteString(display.Dim.Render("[" + ts + "] "))
			b.WriteString(a.Content)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Linked memories.
	if len(linkedMems) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("Linked Memories (%d)", len(linkedMems))))
		b.WriteString("\n")
		for _, m := range linkedMems {
			b.WriteString(display.TypeBadge(m.Type))
			b.WriteString(" ")
			b.WriteString(display.Dim.Render(shortID(m.ID.String())))
			b.WriteString("  ")
			b.WriteString(display.SingleLine(decryptMemoryContent(&m)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Comments.
	if len(comments) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("Comments (%d)", len(comments))))
		b.WriteString("\n")
		b.WriteString(renderCommentTree(comments))
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// renderMemoryDetail builds the detail view for a memory, including comment
// thread + linked tasks.
func renderMemoryDetail(
	m *models.Memory,
	linkedTasks []models.Task,
	comments []models.Comment,
) string {
	if m == nil {
		return display.Dim.Render("(no memory)")
	}
	content := decryptMemoryContent(m)
	var b strings.Builder

	// Header line.
	b.WriteString(display.TypeBadge(m.Type))
	b.WriteString(" ")
	b.WriteString(display.Dim.Render(shortID(m.ID.String())))
	b.WriteString("    ")
	b.WriteString(display.Dim.Render(fmt.Sprintf("★ access %d · %s",
		m.AccessCount, display.Relative(m.UpdatedAt))))
	b.WriteString("\n")
	// First line of content as the title-ish.
	first := display.SingleLine(content)
	b.WriteString(display.Title.Render(display.Truncate(first, 80)))
	b.WriteString("\n\n")

	if m.Project != nil && m.Project.Name != "" {
		b.WriteString(display.Label.Render("Project: "))
		b.WriteString(m.Project.Name)
		b.WriteString("\n")
	}
	if tags := tagSlice(m.Tags); len(tags) > 0 {
		b.WriteString(display.Label.Render("Tags: "))
		b.WriteString(strings.Join(tags, " · "))
		b.WriteString("\n")
	}
	b.WriteString(display.Label.Render("Created: "))
	b.WriteString(display.Relative(m.CreatedAt))
	b.WriteString("\n\n")

	// Full content.
	b.WriteString(sectionHead("Content"))
	b.WriteString("\n")
	b.WriteString(content)
	b.WriteString("\n\n")

	if len(comments) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("Comments (%d)", len(comments))))
		b.WriteString("\n")
		b.WriteString(renderCommentTree(comments))
		b.WriteString("\n")
	}

	if len(linkedTasks) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("Linked Tasks (%d)", len(linkedTasks))))
		b.WriteString("\n")
		for _, t := range linkedTasks {
			title, _ := decryptTask(&t)
			b.WriteString(display.StatusIcon(t.Status))
			b.WriteString(" ")
			b.WriteString(display.Dim.Render(shortID(t.ID.String())))
			b.WriteString("  ")
			b.WriteString(display.SingleLine(title))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// renderProjectDetail summarizes a project plus a capped list of recent tasks
// and memories.
func renderProjectDetail(
	p *models.Project,
	tasks []models.Task,
	memories []models.Memory,
) string {
	if p == nil {
		return display.Dim.Render("(no project)")
	}
	var b strings.Builder

	b.WriteString(display.Dim.Render("[project]"))
	b.WriteString(" ")
	b.WriteString(display.Dim.Render(shortID(p.ID.String())))
	b.WriteString("\n")
	b.WriteString(display.Title.Render(p.Name))
	b.WriteString("\n\n")

	if p.Organization != nil && p.Organization.Name != "" {
		b.WriteString(display.Label.Render("Org: "))
		b.WriteString(p.Organization.Name)
		b.WriteString(" · ")
	}
	b.WriteString(display.Label.Render("Created: "))
	b.WriteString(display.Relative(p.CreatedAt))
	b.WriteString("\n\n")

	if p.Description != "" {
		b.WriteString(sectionHead("Description"))
		b.WriteString("\n")
		b.WriteString(p.Description)
		b.WriteString("\n\n")
	}

	const cap = 10
	tShown := len(tasks)
	if tShown > cap {
		tShown = cap
	}
	b.WriteString(sectionHead(fmt.Sprintf("Recent Tasks (showing %d of %d)", tShown, len(tasks))))
	b.WriteString("\n")
	for i := 0; i < tShown; i++ {
		t := tasks[i]
		title, _ := decryptTask(&t)
		b.WriteString(display.StatusIcon(t.Status))
		b.WriteString(" ")
		b.WriteString(display.PriorityBadge(t.Priority))
		b.WriteString(" ")
		b.WriteString(display.Dim.Render(shortID(t.ID.String())))
		b.WriteString("  ")
		b.WriteString(display.SingleLine(title))
		b.WriteString("\n")
	}
	if len(tasks) == 0 {
		b.WriteString(display.Dim.Render("(none)"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	mShown := len(memories)
	if mShown > cap {
		mShown = cap
	}
	b.WriteString(sectionHead(fmt.Sprintf("Recent Memories (showing %d of %d)", mShown, len(memories))))
	b.WriteString("\n")
	for i := 0; i < mShown; i++ {
		m := memories[i]
		b.WriteString(display.TypeBadge(m.Type))
		b.WriteString(" ")
		b.WriteString(display.Dim.Render(shortID(m.ID.String())))
		b.WriteString("  ")
		b.WriteString(display.SingleLine(decryptMemoryContent(&m)))
		b.WriteString("\n")
	}
	if len(memories) == 0 {
		b.WriteString(display.Dim.Render("(none)"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(display.Dim.Render("↵ enter to drill into this project's tasks"))
	return strings.TrimRight(b.String(), "\n")
}

// renderOrgDetail summarizes an organization plus its projects.
func renderOrgDetail(
	org *api.Organization,
	projects []models.Project,
	enc *api.OrgEncryptionStatus,
) string {
	if org == nil {
		return display.Dim.Render("(no organization)")
	}
	var b strings.Builder
	b.WriteString(display.Dim.Render("[org]"))
	b.WriteString(" ")
	b.WriteString(display.Dim.Render(shortID(org.ID)))
	b.WriteString("\n")
	b.WriteString(display.Title.Render(org.Name))
	b.WriteString("\n\n")

	if org.Description != "" {
		b.WriteString(display.Label.Render("Description: "))
		b.WriteString(org.Description)
		b.WriteString("\n")
	}
	if org.OwnerID != "" {
		b.WriteString(display.Label.Render("Owner: "))
		b.WriteString(display.Dim.Render(shortID(org.OwnerID)))
		b.WriteString("\n")
	}
	if enc != nil {
		b.WriteString(display.Label.Render("Encryption: "))
		if enc.IsEnabled {
			b.WriteString(display.Good.Render(fmt.Sprintf("✓ Enabled (v%d)", enc.EncryptionVersion)))
		} else {
			b.WriteString(display.Dim.Render("× Disabled"))
		}
		b.WriteString("\n")
	}
	b.WriteString(display.Label.Render("Created: "))
	b.WriteString(display.Relative(org.CreatedAt))
	b.WriteString("\n\n")

	const cap = 20
	pShown := len(projects)
	if pShown > cap {
		pShown = cap
	}
	b.WriteString(sectionHead(fmt.Sprintf("Projects (%d)", len(projects))))
	b.WriteString("\n")
	for i := 0; i < pShown; i++ {
		p := projects[i]
		b.WriteString(display.Dim.Render(shortID(p.ID.String())))
		b.WriteString("  ")
		b.WriteString(p.Name)
		b.WriteString("\n")
	}
	if len(projects) == 0 {
		b.WriteString(display.Dim.Render("(none)"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(display.Dim.Render("↵ enter to drill into this org's projects"))
	return strings.TrimRight(b.String(), "\n")
}

// renderActivityDetail expands a single activity feed item.
func renderActivityDetail(item models.ActivityItem) string {
	var b strings.Builder
	b.WriteString(display.TypeBadge(item.EntityType))
	b.WriteString(" ")
	b.WriteString(display.Dim.Render(item.EntityID.String()))
	b.WriteString("\n\n")
	b.WriteString(display.Label.Render("Timestamp: "))
	b.WriteString(item.Timestamp.Format("2006-01-02 15:04:05"))
	b.WriteString(" (")
	b.WriteString(display.Relative(item.Timestamp))
	b.WriteString(")\n")
	if item.ProjectID != nil {
		b.WriteString(display.Label.Render("Project ID: "))
		b.WriteString(item.ProjectID.String())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(sectionHead("Summary"))
	b.WriteString("\n")
	b.WriteString(item.Summary)
	return b.String()
}

// renderKanbanDetail draws a 3-column board inside the detail pane.
// Mirrors the logic of internal/cli/commands/kanban.renderBoard but writes to
// a string and uses the pane's pixel width instead of terminal width.
func renderKanbanDetail(width int, todo, inProgress, completed []models.Task) string {
	cols := [3]struct {
		title string
		tasks []models.Task
	}{
		{"📝 TODO", todo},
		{"🚀 IN PROGRESS", inProgress},
		{"✅ COMPLETED", completed},
	}

	if width < 60 {
		width = 60
	}
	colWidth := (width - 8) / 3
	if colWidth < 18 {
		colWidth = 18
	}

	var b strings.Builder
	b.WriteString(display.Title.Render("🗂  Kanban"))
	b.WriteString(" ")
	b.WriteString(display.Dim.Render(fmt.Sprintf("%d todo · %d in progress · %d done",
		len(todo), len(inProgress), len(completed))))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf(" %s | %s | %s\n",
		padRightTUI(display.Dim.Render(cols[0].title), colWidth),
		padRightTUI(display.Dim.Render(cols[1].title), colWidth),
		padRightTUI(display.Dim.Render(cols[2].title), colWidth)))
	b.WriteString(fmt.Sprintf(" %s-+-%s-+-%s\n",
		strings.Repeat("─", colWidth),
		strings.Repeat("─", colWidth),
		strings.Repeat("─", colWidth)))

	rows := maxLenTUI(cols[0].tasks, cols[1].tasks, cols[2].tasks)
	if rows == 0 {
		b.WriteString(" ")
		b.WriteString(display.Dim.Render("(no tasks in this project yet)"))
		return b.String()
	}
	for i := 0; i < rows; i++ {
		b.WriteString(fmt.Sprintf(" %s | %s | %s\n",
			padRightTUI(kanbanCell(cols[0].tasks, i, colWidth), colWidth),
			padRightTUI(kanbanCell(cols[1].tasks, i, colWidth), colWidth),
			padRightTUI(kanbanCell(cols[2].tasks, i, colWidth), colWidth)))
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderProfileDetail renders the user profile + agents + stats + oauth in
// one big page.
func renderProfileDetail(
	p *models.UserProfile,
	orgs []api.Organization,
	agents []models.AgentProfile,
	stats *models.AgentEventStats,
	oauth []models.OAuthAccount,
) string {
	if p == nil {
		return display.Dim.Render("(no profile)")
	}
	var b strings.Builder

	b.WriteString(display.Dim.Render("[profile]"))
	b.WriteString(" ")
	b.WriteString(p.Email)
	b.WriteString("\n")
	name := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if name != "" {
		b.WriteString(display.Title.Render(name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(display.Label.Render("ID: "))
	b.WriteString(p.ID)
	b.WriteString("\n")
	if p.APIKey != "" {
		b.WriteString(display.Label.Render("API Key: "))
		b.WriteString(maskAPIKey(p.APIKey))
		b.WriteString("\n")
	}
	if p.ActiveOrganizationID != nil && *p.ActiveOrganizationID != "" {
		activeName := ""
		for _, o := range orgs {
			if o.ID == *p.ActiveOrganizationID {
				activeName = o.Name
				break
			}
		}
		b.WriteString(display.Label.Render("Active Org: "))
		b.WriteString(display.Dim.Render(shortID(*p.ActiveOrganizationID)))
		if activeName != "" {
			b.WriteString(" ")
			b.WriteString(activeName)
		}
		b.WriteString("\n")
	}
	if p.CreatedAt != nil {
		b.WriteString(display.Label.Render("Created: "))
		b.WriteString(p.CreatedAt.Format("2006-01-02"))
		if p.UpdatedAt != nil {
			b.WriteString(" · ")
			b.WriteString(display.Label.Render("Updated: "))
			b.WriteString(p.UpdatedAt.Format("2006-01-02"))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(orgs) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("Organizations (%d)", len(orgs))))
		b.WriteString("\n")
		for _, o := range orgs {
			b.WriteString("- ")
			b.WriteString(o.Name)
			b.WriteString(" · ")
			b.WriteString(display.Dim.Render(shortID(o.ID)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Top agents from stats (preferred), fall back to ListAgents.
	if stats != nil && len(stats.TopAgents) > 0 {
		shown := stats.TopAgents
		if len(shown) > 10 {
			shown = shown[:10]
		}
		b.WriteString(sectionHead(fmt.Sprintf("Top Agents (%d)", len(shown))))
		b.WriteString("\n")
		for _, a := range shown {
			label := a.DisplayName
			if label == "" {
				label = a.AgentName
			}
			active := display.Dim.Render("× inactive")
			if a.IsActive {
				active = display.Good.Render("✓ active")
			}
			last := ""
			if a.LastEventAt != nil {
				last = " · last " + display.Relative(*a.LastEventAt)
			}
			typeStr := a.AgentType
			if typeStr == "" {
				typeStr = "agent"
			}
			b.WriteString(fmt.Sprintf("[%s] %s · %d events%s · %s\n",
				typeStr, label, a.EventCount, last, active))
		}
		b.WriteString("\n")
	} else if len(agents) > 0 {
		shown := agents
		if len(shown) > 10 {
			shown = shown[:10]
		}
		b.WriteString(sectionHead(fmt.Sprintf("Agents (%d)", len(shown))))
		b.WriteString("\n")
		for _, a := range shown {
			label := a.DisplayName
			if label == "" {
				label = a.AgentName
			}
			active := display.Dim.Render("× inactive")
			if a.IsActive {
				active = display.Good.Render("✓ active")
			}
			last := ""
			if a.LastEventAt != nil {
				last = " · last " + display.Relative(*a.LastEventAt)
			}
			b.WriteString(fmt.Sprintf("[%s] %s · %d events%s · %s\n",
				a.AgentType, label, a.TotalEvents, last, active))
		}
		b.WriteString("\n")
	}

	if stats != nil {
		b.WriteString(sectionHead("Stats"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Total events: %d · Last 24h: %d · Last 7d: %d\n",
			stats.TotalEvents, stats.EventsLast24h, stats.EventsLast7d))
		if len(stats.EventsByType) > 0 {
			b.WriteString("By type: ")
			b.WriteString(formatCountMap(stats.EventsByType))
			b.WriteString("\n")
		}
		if len(stats.EventsBySource) > 0 {
			b.WriteString("By source: ")
			b.WriteString(formatCountMap(stats.EventsBySource))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(oauth) > 0 {
		b.WriteString(sectionHead(fmt.Sprintf("OAuth Accounts (%d)", len(oauth))))
		b.WriteString("\n")
		for _, o := range oauth {
			line := "- " + o.Provider
			if o.Email != "" {
				line += "  " + o.Email
			} else if o.Username != "" {
				line += "  " + o.Username
			}
			if o.LastLoginAt != nil {
				line += " · last login " + display.Relative(*o.LastLoginAt)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// ---- Render helpers (private) -------------------------------------------------

// tagSlice coerces the loose Tags interface{} field on Task / Memory into
// a clean []string. Tags can come back as []interface{} of strings or as
// a map[string]interface{} envelope from older servers.
func tagSlice(v interface{}) []string {
	out := []string{}
	switch tv := v.(type) {
	case []interface{}:
		for _, x := range tv {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range tv {
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// renderCommentTree renders top-level comments and one level of replies.
// Uses comment.Replies if populated; otherwise falls back to filtering by
// ParentID.
func renderCommentTree(comments []models.Comment) string {
	var b strings.Builder
	// Build a quick parent-id map for the fallback branch.
	childrenByParent := map[string][]models.Comment{}
	tops := []models.Comment{}
	for _, c := range comments {
		if c.ParentID != nil && *c.ParentID != "" {
			childrenByParent[*c.ParentID] = append(childrenByParent[*c.ParentID], c)
		} else {
			tops = append(tops, c)
		}
	}

	for _, c := range tops {
		b.WriteString(renderOneComment(c, false))
		// Prefer embedded replies if the server populated them; otherwise the
		// fallback map.
		replies := c.Replies
		if len(replies) == 0 {
			replies = childrenByParent[c.ID]
		}
		for _, r := range replies {
			b.WriteString(renderOneComment(r, true))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderOneComment formats one comment line + body. nested=true indents and
// prefixes with a tree branch glyph.
func renderOneComment(c models.Comment, nested bool) string {
	author := commentAuthor(c)
	stamp := c.CreatedAt.Format("2006-01-02")
	prefix := ""
	bodyIndent := "  "
	if nested {
		prefix = "  └ "
		bodyIndent = "      "
	}
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString(display.Dim.Render("@" + author + " " + stamp))
	b.WriteString("\n")
	for _, line := range strings.Split(strings.TrimRight(c.Content, "\n"), "\n") {
		b.WriteString(bodyIndent)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// commentAuthor picks the best label available on a Comment author.
func commentAuthor(c models.Comment) string {
	if c.Author != nil {
		if c.Author.Name != "" {
			return c.Author.Name
		}
		if c.Author.Email != "" {
			return c.Author.Email
		}
		if c.Author.ID != "" {
			return shortID(c.Author.ID)
		}
	}
	return "anon"
}

// maskAPIKey returns first 4 chars + ellipsis + last 8 chars (mirrors the
// frontend's Account page).
func maskAPIKey(s string) string {
	if len(s) <= 12 {
		return strings.Repeat("•", len(s))
	}
	return s[:4] + "…" + s[len(s)-8:]
}

// formatCountMap renders a map[string]int as "{a: 1, b: 2}" with stable
// (alphabetical) ordering.
func formatCountMap(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d", k, m[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// kanbanCell formats one cell in the kanban grid.
func kanbanCell(tasks []models.Task, i, width int) string {
	if i >= len(tasks) {
		return ""
	}
	t := tasks[i]
	title, _ := decryptTask(&t)
	title = display.SingleLine(title)
	titleBudget := width - 13
	if titleBudget < 8 {
		titleBudget = 8
	}
	return fmt.Sprintf("%s %s %s",
		display.PriorityBadge(t.Priority),
		display.Dim.Render(t.ID.String()[:8]),
		display.Truncate(title, titleBudget))
}

// padRightTUI pads to visible width. Local copy to avoid pulling in the
// commands package (which would create a tui→commands→tui import cycle once
// the kanban command is updated to share helpers).
func padRightTUI(s string, n int) string {
	visible := stripANSITUI(s)
	if len(visible) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(visible))
}

func stripANSITUI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func maxLenTUI(a, b, c []models.Task) int {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	if len(c) > m {
		m = len(c)
	}
	return m
}
