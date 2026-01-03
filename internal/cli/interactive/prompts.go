package interactive

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kutbudev/ramorie-cli/pkg/models"
)

// SelectTask prompts user to select a task from a list
func SelectTask(tasks []models.Task, message string) (*models.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks available")
	}

	// Build options with format: "shortID - description (status)"
	options := make([]string, len(tasks))
	taskMap := make(map[string]*models.Task)

	for i, task := range tasks {
		shortID := task.ID[:8]
		status := getStatusIcon(string(task.Status))
		priority := getPriorityIcon(string(task.Priority))
		option := fmt.Sprintf("%s %s %s - %s", priority, status, shortID, task.Description)
		options[i] = option
		taskMap[option] = &tasks[i]
	}

	prompt := &survey.Select{
		Message:  message,
		Options:  options,
		PageSize: 10,
	}

	var selected string
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}

	return taskMap[selected], nil
}

// SelectMemory prompts user to select a memory from a list
func SelectMemory(memories []models.MemoryItem, message string) (*models.MemoryItem, error) {
	if len(memories) == 0 {
		return nil, fmt.Errorf("no memories available")
	}

	// Build options with format: "shortID - content preview"
	options := make([]string, len(memories))
	memoryMap := make(map[string]*models.MemoryItem)

	for i, memory := range memories {
		shortID := memory.ID.String()[:8]
		contentPreview := memory.Content
		if len(contentPreview) > 60 {
			contentPreview = contentPreview[:57] + "..."
		}
		option := fmt.Sprintf("🧠 %s - %s", shortID, contentPreview)
		options[i] = option
		memoryMap[option] = &memories[i]
	}

	prompt := &survey.Select{
		Message:  message,
		Options:  options,
		PageSize: 10,
	}

	var selected string
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}

	return memoryMap[selected], nil
}

// SelectProject prompts user to select a project
func SelectProject(projects []models.Project, message string) (*models.Project, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects available")
	}

	options := make([]string, len(projects))
	projectMap := make(map[string]*models.Project)

	for i, project := range projects {
		desc := ""
		if project.Description != nil {
			desc = *project.Description
		}
		option := fmt.Sprintf("%s - %s", project.Name, desc)
		options[i] = option
		projectMap[option] = &projects[i]
	}

	prompt := &survey.Select{
		Message:  message,
		Options:  options,
		PageSize: 10,
	}

	var selected string
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}

	return projectMap[selected], nil
}

// CreateTaskInteractive prompts for task creation details
func CreateTaskInteractive() (*models.Task, error) {
	task := &models.Task{}

	// Description
	descPrompt := &survey.Input{
		Message: "Task description:",
	}
	err := survey.AskOne(descPrompt, &task.Description, survey.WithValidator(survey.Required))
	if err != nil {
		return nil, err
	}

	// Priority
	priorityPrompt := &survey.Select{
		Message: "Priority:",
		Options: []string{"🔴 High", "🟡 Medium", "🟢 Low"},
		Default: "🟡 Medium",
	}
	var priorityChoice string
	err = survey.AskOne(priorityPrompt, &priorityChoice)
	if err != nil {
		return nil, err
	}

	switch priorityChoice {
	case "🔴 High":
		task.Priority = "H"
	case "🟡 Medium":
		task.Priority = "M"
	case "🟢 Low":
		task.Priority = "L"
	}

	// Status (default TODO)
	task.Status = "TODO"
	task.Progress = 0

	return task, nil
}

// AnnotateTaskInteractive prompts for annotation details
func AnnotateTaskInteractive() (string, error) {
	prompt := &survey.Multiline{
		Message: "Annotation content:",
	}

	var content string
	err := survey.AskOne(prompt, &content, survey.WithValidator(survey.Required))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(content), nil
}

// ConfirmAction prompts for confirmation of destructive actions
func ConfirmAction(message string, details string) (bool, error) {
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("%s\n%s\nAre you sure?", message, details),
	}

	var confirmed bool
	err := survey.AskOne(prompt, &confirmed)
	return confirmed, err
}

// SelectStatus prompts for status selection
func SelectStatus(message string, currentStatus string) (string, error) {
	statusOptions := map[string]string{
		"📋 TODO":        "TODO",
		"🚀 IN PROGRESS": "IN_PROGRESS",
		"👀 IN REVIEW":   "IN_REVIEW",
		"✅ COMPLETED":   "COMPLETED",
	}

	options := make([]string, 0, len(statusOptions))
	for display := range statusOptions {
		options = append(options, display)
	}

	prompt := &survey.Select{
		Message: message,
		Options: options,
	}

	var selected string
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return "", err
	}

	return statusOptions[selected], nil
}

// Helper functions
func getStatusIcon(status string) string {
	icons := map[string]string{
		"TODO":        "📋",
		"IN_PROGRESS": "🚀",
		"IN_REVIEW":   "👀",
		"COMPLETED":   "✅",
	}
	if icon, exists := icons[status]; exists {
		return icon
	}
	return "❓"
}

func getPriorityIcon(priority string) string {
	icons := map[string]string{
		"H": "🔴",
		"M": "🟡",
		"L": "🟢",
	}
	if icon, exists := icons[priority]; exists {
		return icon
	}
	return "⚪"
}
