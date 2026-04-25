package commands

import (
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/cli/display"
	"github.com/kutbudev/ramorie-cli/internal/cli/resolve"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/urfave/cli/v2"
)

// NewKanbanCmd renders a three-column board (TODO / IN_PROGRESS / COMPLETED)
// for a single project. Project arg accepts name, short ID prefix, or full UUID.
func NewKanbanCmd() *cli.Command {
	return &cli.Command{
		Name:      "kanban",
		Usage:     "Display tasks in a beautified kanban board",
		ArgsUsage: "[project]",
		Description: `Three-column board (TODO / IN PROGRESS / COMPLETED) for one project.

Project arg accepts name, short ID prefix, or full UUID.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project (name, short id, or UUID)"},
		},
		Action: func(c *cli.Context) error {
			arg := c.String("project")
			if arg == "" && c.NArg() > 0 {
				arg = c.Args().First()
			}
			if arg == "" {
				return fmt.Errorf("project is required. Usage: ramorie kanban -p <project> (name or UUID)")
			}

			client := api.NewClient()
			projectID, err := resolve.ResolveProject(arg, client)
			if err != nil {
				return err
			}

			todo, err := client.ListTasks(projectID, "TODO")
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}
			inProgress, err := client.ListTasks(projectID, "IN_PROGRESS")
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}
			completed, err := client.ListTasks(projectID, "COMPLETED")
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			renderBoard(arg, todo, inProgress, completed)
			return nil
		},
	}
}

func renderBoard(projectLabel string, todo, inProgress, completed []models.Task) {
	cols := [3]struct {
		title string
		tasks []models.Task
	}{
		{"📝 TODO", todo},
		{"🚀 IN PROGRESS", inProgress},
		{"✅ COMPLETED", completed},
	}

	totalWidth := display.TerminalWidth()
	if totalWidth < 90 {
		totalWidth = 90
	}
	colWidth := (totalWidth - 8) / 3
	if colWidth < 22 {
		colWidth = 22
	}
	if colWidth > 42 {
		colWidth = 42
	}

	subtitle := fmt.Sprintf("%d todo · %d in progress · %d done",
		len(todo), len(inProgress), len(completed))
	fmt.Println(display.Header("🗂  Kanban — "+projectLabel, subtitle))
	fmt.Println()

	fmt.Printf(" %s | %s | %s\n",
		padRight(display.Dim.Render(cols[0].title), colWidth),
		padRight(display.Dim.Render(cols[1].title), colWidth),
		padRight(display.Dim.Render(cols[2].title), colWidth))
	fmt.Printf(" %s-+-%s-+-%s\n",
		strings.Repeat("─", colWidth),
		strings.Repeat("─", colWidth),
		strings.Repeat("─", colWidth))

	rows := maxLen(cols[0].tasks, cols[1].tasks, cols[2].tasks)
	if rows == 0 {
		fmt.Println(" " + display.Dim.Render("  (no tasks in this project yet)"))
		return
	}
	for i := 0; i < rows; i++ {
		fmt.Printf(" %s | %s | %s\n",
			padRight(cellFor(cols[0].tasks, i, colWidth), colWidth),
			padRight(cellFor(cols[1].tasks, i, colWidth), colWidth),
			padRight(cellFor(cols[2].tasks, i, colWidth), colWidth))
	}

	fmt.Println()
	fmt.Println(display.Dim.Render(" priority: ") +
		display.PriorityBadge("H") + " high  " +
		display.PriorityBadge("M") + " med  " +
		display.PriorityBadge("L") + " low")
}

func cellFor(tasks []models.Task, i, width int) string {
	if i >= len(tasks) {
		return ""
	}
	t := tasks[i]
	title, _ := decryptTaskForCLI(&t)
	title = display.SingleLine(title)
	titleBudget := width - 13 // 3 (badge) + 1 + 8 (id) + 1 = 13
	if titleBudget < 8 {
		titleBudget = 8
	}
	return fmt.Sprintf("%s %s %s",
		display.PriorityBadge(t.Priority),
		display.Dim.Render(t.ID.String()[:8]),
		display.Truncate(title, titleBudget))
}

// padRight pads a (possibly ANSI-decorated) string to visible width n.
func padRight(s string, n int) string {
	visible := stripANSI(s)
	if len(visible) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(visible))
}

func stripANSI(s string) string {
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

func maxLen(a, b, c []models.Task) int {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	if len(c) > m {
		m = len(c)
	}
	return m
}
