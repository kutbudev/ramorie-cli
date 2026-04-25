package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
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

// renderTaskDetail builds a key:value formatted block for a task plus
// counts of related entities.
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

	b.WriteString(display.Title.Render(display.SingleLine(title)))
	b.WriteString("\n\n")
	b.WriteString(display.Label.Render("ID:        "))
	b.WriteString(t.ID.String())
	b.WriteString("\n")
	b.WriteString(display.Label.Render("Status:    "))
	b.WriteString(display.StatusLabel(t.Status))
	b.WriteString("\n")
	b.WriteString(display.Label.Render("Priority:  "))
	b.WriteString(display.PriorityBadge(t.Priority))
	b.WriteString("\n")
	if t.Project != nil {
		b.WriteString(display.Label.Render("Project:   "))
		b.WriteString(t.Project.Name)
		b.WriteString("\n")
	}
	b.WriteString(display.Label.Render("Created:   "))
	b.WriteString(display.Relative(t.CreatedAt))
	b.WriteString("\n")
	b.WriteString(display.Label.Render("Updated:   "))
	b.WriteString(display.Relative(t.UpdatedAt))
	b.WriteString("\n\n")

	if desc != "" {
		b.WriteString(display.Label.Render("Description"))
		b.WriteString("\n")
		b.WriteString(desc)
		b.WriteString("\n\n")
	}

	b.WriteString(display.Dim.Render(fmt.Sprintf(
		"(%d subtasks · %d notes · %d linked memories · %d comments)",
		len(subtasks), len(annotations), len(linkedMems), len(comments),
	)))
	return b.String()
}

// renderMemoryDetail builds the detail view for a memory.
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
	b.WriteString(display.TypeBadge(m.Type))
	b.WriteString(" ")
	b.WriteString(display.Title.Render(m.ID.String()))
	b.WriteString("\n\n")
	b.WriteString(display.Label.Render("Created:   "))
	b.WriteString(display.Relative(m.CreatedAt))
	b.WriteString("\n")
	b.WriteString(display.Label.Render("Updated:   "))
	b.WriteString(display.Relative(m.UpdatedAt))
	b.WriteString("\n")
	b.WriteString(display.Label.Render("Access:    "))
	b.WriteString(fmt.Sprintf("%d", m.AccessCount))
	b.WriteString("\n\n")

	b.WriteString(display.Label.Render("Content"))
	b.WriteString("\n")
	b.WriteString(content)
	b.WriteString("\n\n")

	b.WriteString(display.Dim.Render(fmt.Sprintf(
		"(%d linked tasks · %d comments)", len(linkedTasks), len(comments),
	)))
	return b.String()
}
