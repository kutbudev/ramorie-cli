package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
	"golang.org/x/term"
	"gorm.io/gorm"
)

// NewKanbanCmd creates the kanban command
func NewKanbanCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "kanban",
		Short: "Display tasks in a beautiful kanban board",
		Long:  "Show tasks organized by status in a full-width kanban board layout",
		Run: func(cmd *cobra.Command, args []string) {
			// Get active project
			var project models.Project
			result := db.Where("is_active = ? AND deleted_at IS NULL", true).First(&project)
			if result.Error != nil {
				fmt.Println("❌ No active project found")
				fmt.Println("💡 Use 'jbraincli use <project>' to set an active project")
				return
			}

			// Get all tasks for the active project
			var tasks []models.Task
			err := db.Where("project_id = ?", project.ID).Find(&tasks).Error
			if err != nil {
				fmt.Printf("❌ Error fetching tasks: %v\n", err)
				return
			}

			// Display kanban board
			displayKanbanBoard(tasks, project.Name)
		},
	}
}

func displayKanbanBoard(tasks []models.Task, projectName string) {
	// Get terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 120 // Default width
	}

	// Organize tasks by status
	statusColumns := map[string][]models.Task{
		"TODO":        {},
		"IN_PROGRESS": {},
		"IN_REVIEW":   {},
		"COMPLETED":   {},
	}

	for _, task := range tasks {
		if _, exists := statusColumns[task.Status]; exists {
			statusColumns[task.Status] = append(statusColumns[task.Status], task)
		}
	}

	// Calculate column width (4 columns + borders + padding)
	columnWidth := (width - 8) / 4 // 8 chars for borders and spacing

	// Header
	fmt.Printf("\n🎯 %s - Kanban Board\n\n", projectName)

	// Print top border
	printKanbanBorder(columnWidth, "top")

	// Print column headers
	fmt.Print("│")
	printCenteredText("📋 TODO", columnWidth)
	fmt.Print("│")
	printCenteredText("🚀 IN PROGRESS", columnWidth)
	fmt.Print("│")
	printCenteredText("👀 IN REVIEW", columnWidth)
	fmt.Print("│")
	printCenteredText("✅ COMPLETED", columnWidth)
	fmt.Println("│")

	// Print separator
	printKanbanBorder(columnWidth, "middle")

	// Find max tasks in any column for row count
	maxTasks := 0
	for _, tasks := range statusColumns {
		if len(tasks) > maxTasks {
			maxTasks = len(tasks)
		}
	}

	// Print task rows
	statuses := []string{"TODO", "IN_PROGRESS", "IN_REVIEW", "COMPLETED"}
	for i := 0; i < maxTasks; i++ {
		fmt.Print("│")
		for _, status := range statuses {
			tasks := statusColumns[status]
			if i < len(tasks) {
				taskText := formatTaskForKanban(tasks[i], columnWidth-2)
				fmt.Printf(" %-*s", columnWidth-2, taskText)
			} else {
				fmt.Printf(" %-*s", columnWidth-2, "")
			}
			fmt.Print(" │")
		}
		fmt.Println()
	}

	// Print bottom border
	printKanbanBorder(columnWidth, "bottom")

	// Print summary
	fmt.Printf("\n📊 Summary: %d TODO • %d IN PROGRESS • %d IN REVIEW • %d COMPLETED\n\n",
		len(statusColumns["TODO"]),
		len(statusColumns["IN_PROGRESS"]),
		len(statusColumns["IN_REVIEW"]),
		len(statusColumns["COMPLETED"]))
}

func printKanbanBorder(columnWidth int, position string) {
	var left, right, horizontal, junction string

	switch position {
	case "top":
		left, right, horizontal, junction = "┌", "┐", "─", "┬"
	case "middle":
		left, right, horizontal, junction = "├", "┤", "─", "┼"
	case "bottom":
		left, right, horizontal, junction = "└", "┘", "─", "┴"
	}

	fmt.Print(left)
	for i := 0; i < 4; i++ {
		fmt.Print(strings.Repeat(horizontal, columnWidth))
		if i < 3 {
			fmt.Print(junction)
		}
	}
	fmt.Println(right)
}

func printCenteredText(text string, width int) {
	textLen := len(text)
	if textLen >= width {
		fmt.Printf(" %-*s", width-2, truncateString(text, width-2))
		return
	}

	padding := (width - textLen) / 2
	fmt.Printf("%*s%s%*s", padding, "", text, width-textLen-padding, "")
}

func formatTaskForKanban(task models.Task, maxWidth int) string {
	// Priority indicator
	priorityIcon := map[string]string{
		"H": "🔴",
		"M": "🟡", 
		"L": "🟢",
	}

	icon := "⚪"
	if p, exists := priorityIcon[task.Priority]; exists {
		icon = p
	}

	// Short ID (first 8 chars)
	shortID := task.ID.String()[:8]
	
	// Format: icon + short ID + description
	prefix := fmt.Sprintf("%s %s ", icon, shortID)
	availableWidth := maxWidth - len(prefix)
	
	if availableWidth <= 0 {
		return truncateString(prefix, maxWidth)
	}

	description := truncateString(task.Description, availableWidth)
	return prefix + description
} 