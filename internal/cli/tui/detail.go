package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// Markdown-safe glyph helpers — used in detail render functions instead of
// the lipgloss-rendered display.* badges. We can't embed lipgloss ANSI inside
// a markdown source string because glamour parses it as inline text and
// strips the ESC byte, leaving raw "[38;2;...m" garbage on screen. These
// helpers emit pure unicode + the markdown source decides the styling.

func mdPriority(p string) string {
	switch p {
	case "H":
		return "🔴 H"
	case "M":
		return "🟡 M"
	case "L":
		return "🟢 L"
	}
	return p
}

func mdStatusLabel(s string) string {
	switch s {
	case "COMPLETED":
		return "✓ COMPLETED"
	case "IN_PROGRESS":
		return "◐ IN_PROGRESS"
	case "TODO":
		return "○ TODO"
	case "BLOCKED":
		return "✗ BLOCKED"
	}
	return s
}

func mdStatusIcon(s string) string {
	switch s {
	case "COMPLETED":
		return "✓"
	case "IN_PROGRESS":
		return "◐"
	case "TODO":
		return "○"
	case "BLOCKED":
		return "✗"
	}
	return "·"
}

func mdTypeBadge(t string) string {
	if t == "" {
		return ""
	}
	return "`[" + t + "]`"
}

// detailModel is the right pane — a scrollable rendering of the currently
// selected entity.
//
// Content flows in as a markdown source string from the various
// render*Detail() functions. setContent() runs it through glamour using the
// current width + theme and feeds the resulting ANSI text to the viewport.
//
// Some content (kanban grid, error/loading) is NOT markdown — those callers
// use setRawContent() which bypasses glamour.
type detailModel struct {
	vp       viewport.Model
	width    int
	height   int
	focused  bool
	loading  bool
	content  string // pre-rendered text fed into the viewport
	errMsg   string
	caps     terminalCaps
	theme    string
	// lastContent is the most recent markdown source passed to setContent().
	// Stored so we can re-render on width or theme changes.
	lastContent    string
	lastIsMarkdown bool
	// renderToken increments on every setContent so stale async render
	// outputs (cursor already moved on) are discarded by applyRendered.
	renderToken uint64
}

func newDetail(width, height int) detailModel {
	vp := viewport.New(maxInt(width-4, 10), maxInt(height-4, 3))
	return detailModel{
		vp:     vp,
		width:  width,
		height: height,
		theme:  ThemeAuto,
	}
}

func (d *detailModel) resize(width, height int) {
	d.width = width
	d.height = height
	d.vp.Width = maxInt(width-4, 10)
	d.vp.Height = maxInt(height-4, 3)
	// Re-render markdown to the new wrap width.
	if d.lastIsMarkdown && d.lastContent != "" {
		d.content = d.renderMD(d.lastContent)
	}
	d.vp.SetContent(d.content)
}

// renderedMsg carries an async glamour render result back to rootModel.
// rootModel calls detailModel.applyRendered, which drops stale tokens.
type renderedMsg struct {
	token  uint64
	output string
}

// setContent shows `s` (markdown source) on the right pane.
//
// Fast path: if `s` is short, has no markdown markers, OR is already in the
// glamour output cache, render is synchronous — the user sees the final
// result instantly.
//
// Slow path: heavier markdown (with code fences, long content, …) is shown
// as RAW text immediately so the cursor never feels blocked, AND a tea.Cmd
// is returned that runs glamour in a goroutine. When the goroutine produces
// a renderedMsg, rootModel calls applyRendered to swap in the prettified
// output. If the cursor moved on before the result arrived, the token
// mismatches and the result is dropped.
func (d *detailModel) setContent(s string) tea.Cmd {
	d.lastContent = s
	d.lastIsMarkdown = true
	d.renderToken++
	tok := d.renderToken
	d.loading = false
	d.errMsg = ""

	if s == "" {
		d.content = ""
		d.vp.SetContent("")
		d.vp.GotoTop()
		return nil
	}

	// Cache lookup uses the same key shape renderMarkdown uses internally;
	// we duplicate the lookup here so we can decide sync vs async without
	// running the renderer.
	bw := bucketize(d.vp.Width)
	if bw < widthBucket {
		bw = widthBucket
	}
	if v, ok := mdCache.Load(mdKey(s, bw, d.theme)); ok {
		d.content = v.(string)
		d.vp.SetContent(d.content)
		d.vp.GotoTop()
		return nil
	}

	// Trivial content: skip glamour, show raw, no work.
	if !worthRendering(s) {
		d.content = s
		d.vp.SetContent(s)
		d.vp.GotoTop()
		return nil
	}

	// Heavy content: show raw immediately, render in background.
	d.content = s
	d.vp.SetContent(s)
	d.vp.GotoTop()
	theme := d.theme
	width := d.vp.Width
	return func() tea.Msg {
		out := renderMarkdown(s, width, theme)
		return renderedMsg{token: tok, output: out}
	}
}

// applyRendered swaps in an async render result if the token matches the
// most recent setContent call. Stale results are dropped.
func (d *detailModel) applyRendered(token uint64, out string) {
	if token != d.renderToken {
		return
	}
	d.content = out
	d.vp.SetContent(out)
}

// setRawContent feeds pre-formatted (non-markdown) text directly to the
// viewport. Used by the kanban board and any other view that builds its own
// ANSI layout.
func (d *detailModel) setRawContent(s string) {
	d.lastContent = s
	d.lastIsMarkdown = false
	d.content = s
	d.vp.SetContent(s)
	d.vp.GotoTop()
	d.loading = false
	d.errMsg = ""
}

// setTheme swaps the active glamour theme. Caller is responsible for
// invalidating the global cache before calling. Re-renders the most recent
// markdown content (if any).
func (d *detailModel) setTheme(theme string) {
	d.theme = theme
	if d.lastIsMarkdown && d.lastContent != "" {
		d.content = d.renderMD(d.lastContent)
		d.vp.SetContent(d.content)
	}
}

func (d *detailModel) renderMD(src string) string {
	src = linkifyText(src, d.caps)
	return renderMarkdown(src, maxInt(d.width-4, 10), d.theme)
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

// mdSection writes a "## Title\n\n" header into the builder.
func mdSection(b *strings.Builder, title string) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
}

// renderTaskDetail builds the rich detail block for a task: header, metadata,
// description, subtasks, notes, linked memories and comments.
//
// Output is pure markdown; the detail pane runs it through glamour.
func renderTaskDetail(
	t *models.Task,
	subtasks []models.Subtask,
	annotations []models.Annotation,
	linkedMems []models.Memory,
	comments []models.Comment,
) string {
	if t == nil {
		return "_no task_"
	}
	title, desc := decryptTask(t)
	var b strings.Builder

	// Header line: [task] <id>   <priority> · <status>
	fmt.Fprintf(&b, "`[task]` `%s`  %s · %s\n",
		shortID(t.ID.String()),
		mdPriority(t.Priority),
		mdStatusLabel(t.Status),
	)
	fmt.Fprintf(&b, "# %s\n\n", display.SingleLine(title))

	// Metadata block.
	if t.Project != nil && t.Project.Name != "" {
		fmt.Fprintf(&b, "**Project:** %s  \n", t.Project.Name)
	}
	fmt.Fprintf(&b, "**Created:** %s · **Updated:** %s  \n",
		display.Relative(t.CreatedAt), display.Relative(t.UpdatedAt))
	if tags := tagSlice(t.Tags); len(tags) > 0 {
		fmt.Fprintf(&b, "**Tags:** %s  \n", strings.Join(tags, " · "))
	}
	b.WriteString("\n")

	// Description.
	if desc != "" {
		mdSection(&b, "Description")
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
		mdSection(&b, fmt.Sprintf("Subtasks (%d/%d)", done, len(subtasks)))
		for _, s := range subtasks {
			marker := "- [ ]"
			if s.Completed == 1 || strings.EqualFold(s.Status, "COMPLETED") {
				marker = "- [x]"
			}
			fmt.Fprintf(&b, "%s %s\n", marker, s.Description)
		}
		b.WriteString("\n")
	}

	// Notes (annotations).
	if len(annotations) > 0 {
		mdSection(&b, fmt.Sprintf("Notes (%d)", len(annotations)))
		for _, a := range annotations {
			ts := a.CreatedAt.Format("2006-01-02 15:04")
			fmt.Fprintf(&b, "- _[%s]_ %s\n", ts, decryptAnnotation(&a))
		}
		b.WriteString("\n")
	}

	// Linked memories.
	if len(linkedMems) > 0 {
		mdSection(&b, fmt.Sprintf("Linked Memories (%d)", len(linkedMems)))
		for _, m := range linkedMems {
			fmt.Fprintf(&b, "- %s `%s` %s\n",
				mdTypeBadge(m.Type),
				shortID(m.ID.String()),
				display.SingleLine(decryptMemoryContent(&m)),
			)
		}
		b.WriteString("\n")
	}

	// Comments.
	if len(comments) > 0 {
		mdSection(&b, fmt.Sprintf("Comments (%d)", len(comments)))
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
		return "_no memory_"
	}
	content := decryptMemoryContent(m)
	var b strings.Builder

	// Header line.
	fmt.Fprintf(&b, "%s `%s`  _★ access %d · %s_\n",
		mdTypeBadge(m.Type),
		shortID(m.ID.String()),
		m.AccessCount, display.Relative(m.UpdatedAt),
	)
	first := display.SingleLine(content)
	fmt.Fprintf(&b, "# %s\n\n", display.Truncate(first, 80))

	if m.Project != nil && m.Project.Name != "" {
		fmt.Fprintf(&b, "**Project:** %s  \n", m.Project.Name)
	}
	if tags := tagSlice(m.Tags); len(tags) > 0 {
		fmt.Fprintf(&b, "**Tags:** %s  \n", strings.Join(tags, " · "))
	}
	fmt.Fprintf(&b, "**Created:** %s\n\n", display.Relative(m.CreatedAt))

	// Full content.
	mdSection(&b, "Content")
	b.WriteString(content)
	b.WriteString("\n\n")

	if len(comments) > 0 {
		mdSection(&b, fmt.Sprintf("Comments (%d)", len(comments)))
		b.WriteString(renderCommentTree(comments))
		b.WriteString("\n")
	}

	if len(linkedTasks) > 0 {
		mdSection(&b, fmt.Sprintf("Linked Tasks (%d)", len(linkedTasks)))
		for _, t := range linkedTasks {
			title, _ := decryptTask(&t)
			fmt.Fprintf(&b, "- %s `%s` %s\n",
				mdStatusIcon(t.Status),
				shortID(t.ID.String()),
				display.SingleLine(title),
			)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// renderProjectDetail shows project settings — metadata only. Task/memory
// browsing belongs under their own sidebar tabs (Tasks / Memories with `p`
// to filter). Member management would go here when the backend adds the
// `/projects/:id/members` endpoint.
func renderProjectDetail(
	p *models.Project,
	_ []models.Task,
	_ []models.Memory,
) string {
	if p == nil {
		return "_no project_"
	}
	var b strings.Builder

	fmt.Fprintf(&b, "`[project]` `%s`\n", p.ID.String())
	fmt.Fprintf(&b, "# %s\n\n", p.Name)

	mdSection(&b, "Settings")
	if p.Organization != nil && p.Organization.Name != "" {
		fmt.Fprintf(&b, "- **Organization:** %s\n", p.Organization.Name)
	} else {
		b.WriteString("- **Organization:** _(personal)_\n")
	}
	fmt.Fprintf(&b, "- **Created:** %s\n", display.Relative(p.CreatedAt))
	fmt.Fprintf(&b, "- **Updated:** %s\n", display.Relative(p.UpdatedAt))

	if p.Description != "" {
		b.WriteString("\n")
		mdSection(&b, "Description")
		b.WriteString(p.Description)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	mdSection(&b, "Members")
	b.WriteString("_(project-level membership not yet exposed by the backend)_\n")

	b.WriteString("\n")
	b.WriteString("> use the **Tasks** or **Memories** tabs (`p` to filter by this project) to browse content.\n")
	return strings.TrimRight(b.String(), "\n")
}

// renderOrgDetail summarizes an organization plus its projects.
func renderOrgDetail(
	org *api.Organization,
	projects []models.Project,
	enc *api.OrgEncryptionStatus,
) string {
	if org == nil {
		return "_no organization_"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "`[org]` `%s`\n", shortID(org.ID))
	fmt.Fprintf(&b, "# %s\n\n", org.Name)

	if org.Description != "" {
		fmt.Fprintf(&b, "**Description:** %s  \n", org.Description)
	}
	if org.OwnerID != "" {
		fmt.Fprintf(&b, "**Owner:** `%s`  \n", shortID(org.OwnerID))
	}
	if enc != nil {
		if enc.IsEnabled {
			fmt.Fprintf(&b, "**Encryption:** ✓ Enabled (v%d)  \n", enc.EncryptionVersion)
		} else {
			b.WriteString("**Encryption:** × Disabled  \n")
		}
	}
	fmt.Fprintf(&b, "**Created:** %s\n\n", display.Relative(org.CreatedAt))

	const cap = 20
	pShown := len(projects)
	if pShown > cap {
		pShown = cap
	}
	mdSection(&b, fmt.Sprintf("Projects (%d)", len(projects)))
	for i := 0; i < pShown; i++ {
		p := projects[i]
		fmt.Fprintf(&b, "- `%s` %s\n", shortID(p.ID.String()), p.Name)
	}
	if len(projects) == 0 {
		b.WriteString("_(none)_\n")
	}
	b.WriteString("\n")
	b.WriteString("> ↵ enter to drill into this org's projects")
	return strings.TrimRight(b.String(), "\n")
}

// renderActivityDetail expands a single activity feed item.
func renderActivityDetail(item models.ActivityItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s `%s`\n\n", mdTypeBadge(item.EntityType), item.EntityID.String())
	fmt.Fprintf(&b, "**Timestamp:** %s (%s)\n",
		item.Timestamp.Format("2006-01-02 15:04:05"),
		display.Relative(item.Timestamp),
	)
	if item.ProjectID != nil {
		fmt.Fprintf(&b, "**Project ID:** `%s`\n", item.ProjectID.String())
	}
	b.WriteString("\n")
	mdSection(&b, "Summary")
	b.WriteString(item.Summary)
	return b.String()
}

// renderKanbanDetail draws a 3-column board inside the detail pane.
// Mirrors the logic of internal/cli/commands/kanban.renderBoard but writes to
// a string and uses the pane's pixel width instead of terminal width.
//
// NOTE: Output is NOT markdown — caller should use setRawContent.
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
		return "_no profile_"
	}
	var b strings.Builder

	fmt.Fprintf(&b, "`[profile]` %s\n", p.Email)
	name := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if name != "" {
		fmt.Fprintf(&b, "# %s\n\n", name)
	} else {
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "**ID:** `%s`  \n", p.ID)
	if p.APIKey != "" {
		fmt.Fprintf(&b, "**API Key:** `%s`  \n", maskAPIKey(p.APIKey))
	}
	if p.ActiveOrganizationID != nil && *p.ActiveOrganizationID != "" {
		activeName := ""
		for _, o := range orgs {
			if o.ID == *p.ActiveOrganizationID {
				activeName = o.Name
				break
			}
		}
		fmt.Fprintf(&b, "**Active Org:** `%s`", shortID(*p.ActiveOrganizationID))
		if activeName != "" {
			fmt.Fprintf(&b, " %s", activeName)
		}
		b.WriteString("  \n")
	}
	if p.CreatedAt != nil {
		fmt.Fprintf(&b, "**Created:** %s", p.CreatedAt.Format("2006-01-02"))
		if p.UpdatedAt != nil {
			fmt.Fprintf(&b, " · **Updated:** %s", p.UpdatedAt.Format("2006-01-02"))
		}
		b.WriteString("  \n")
	}
	b.WriteString("\n")

	if len(orgs) > 0 {
		mdSection(&b, fmt.Sprintf("Organizations (%d)", len(orgs)))
		for _, o := range orgs {
			fmt.Fprintf(&b, "- %s · `%s`\n", o.Name, shortID(o.ID))
		}
		b.WriteString("\n")
	}

	// Top agents from stats (preferred), fall back to ListAgents.
	if stats != nil && len(stats.TopAgents) > 0 {
		shown := stats.TopAgents
		if len(shown) > 10 {
			shown = shown[:10]
		}
		mdSection(&b, fmt.Sprintf("Top Agents (%d)", len(shown)))
		for _, a := range shown {
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
				last = " · last " + display.Relative(*a.LastEventAt)
			}
			typeStr := a.AgentType
			if typeStr == "" {
				typeStr = "agent"
			}
			fmt.Fprintf(&b, "- `[%s]` %s · %d events%s · %s\n",
				typeStr, label, a.EventCount, last, active)
		}
		b.WriteString("\n")
	} else if len(agents) > 0 {
		shown := agents
		if len(shown) > 10 {
			shown = shown[:10]
		}
		mdSection(&b, fmt.Sprintf("Agents (%d)", len(shown)))
		for _, a := range shown {
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
				last = " · last " + display.Relative(*a.LastEventAt)
			}
			fmt.Fprintf(&b, "- `[%s]` %s · %d events%s · %s\n",
				a.AgentType, label, a.TotalEvents, last, active)
		}
		b.WriteString("\n")
	}

	if stats != nil {
		mdSection(&b, "Stats")
		fmt.Fprintf(&b, "Total events: **%d** · Last 24h: **%d** · Last 7d: **%d**\n\n",
			stats.TotalEvents, stats.EventsLast24h, stats.EventsLast7d)
		if len(stats.EventsByType) > 0 {
			fmt.Fprintf(&b, "**By type:** %s\n\n", formatCountMap(stats.EventsByType))
		}
		if len(stats.EventsBySource) > 0 {
			fmt.Fprintf(&b, "**By source:** %s\n\n", formatCountMap(stats.EventsBySource))
		}
	}

	if len(oauth) > 0 {
		mdSection(&b, fmt.Sprintf("OAuth Accounts (%d)", len(oauth)))
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

// renderCommentTree renders top-level comments and one level of replies as
// markdown blockquotes (replies are nested deeper).
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

// renderOneComment formats one comment as a markdown blockquote. nested=true
// uses a doubled `>>` to nest the reply visually.
func renderOneComment(c models.Comment, nested bool) string {
	author := commentAuthor(c)
	stamp := c.CreatedAt.Format("2006-01-02")
	prefix := "> "
	if nested {
		prefix = "> > "
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s**@%s** _%s_\n", prefix, author, stamp)
	for _, line := range strings.Split(strings.TrimRight(decryptComment(&c), "\n"), "\n") {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
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
