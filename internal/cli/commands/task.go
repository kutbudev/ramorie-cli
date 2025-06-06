package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/internal/cli/interactive"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
	"golang.org/x/term"
	"gorm.io/gorm"
)

// NewTaskCmd creates the task command with all subcommands
func NewTaskCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
		Long:  "Create, list, update, and manage tasks",
	}

	// Add subcommands with database
	cmd.AddCommand(newTaskCreateCmd(db))
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskStartCmd())
	cmd.AddCommand(newTaskDoneCmd())
	cmd.AddCommand(newTaskInfoCmd())
	cmd.AddCommand(newTaskProgressCmd())
	cmd.AddCommand(newTaskDeleteCmd())
	cmd.AddCommand(newTaskModifyCmd())

	return cmd
}

// task create
func newTaskCreateCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create [description]",
		Short:   "Create a new task via API",
		Aliases: []string{"add"},
		Run: func(cmd *cobra.Command, args []string) {
			isInteractive, _ := cmd.Flags().GetBool("interactive")
			priority, _ := cmd.Flags().GetString("priority")
			description := ""

			if isInteractive {
				// Interactive mode
				task, err := interactive.CreateTaskInteractive()
				if err != nil {
					log.Fatalf("Interactive task creation failed: %v", err)
				}
				description = task.Description
				// Priority from interactive mode overrides flag
				if task.Priority != "" {
					priority = string(task.Priority)
				}
			} else {
				// Traditional CLI mode
				if len(args) < 1 {
					fmt.Println("❌ Description is required when not in interactive mode.")
					fmt.Println("💡 Use 'jbraincli task create \"My new task\"' or 'jbraincli task create -i'")
					return
				}
				description = strings.Join(args, " ")
			}

			// Get active project - require one to exist
			// TODO: Refactor this to use a config file instead of a direct DB call
			var project models.Project
			result := db.Where("is_active = ? AND deleted_at IS NULL", true).First(&project)
			if result.Error != nil {
				fmt.Println("❌ No active project found")
				fmt.Println("💡 Use 'jbraincli project use <name>' to create or set an active project")
				return
			}

			// Create JSON payload for the new task
			payload := map[string]string{
				"project_id":  project.ID.String(),
				"description": description,
				"priority":    strings.ToUpper(priority),
			}
			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("Failed to create JSON payload: %v", err)
			}

			// Send request to the API
			resp, err := http.Post("http://localhost:8080/v1/tasks", "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Fatalf("Failed to send request to API: %v", err)
			}
			defer resp.Body.Close()

			// Check response
			if resp.StatusCode != http.StatusCreated {
				var apiError map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&apiError); err == nil {
					log.Fatalf("API returned an error: %s (Status: %s)", apiError["error"], resp.Status)
				}
				log.Fatalf("API returned a non-201 status code: %s", resp.Status)
			}

			// Decode the created task and print info
			var createdTask models.Task
			if err := json.NewDecoder(resp.Body).Decode(&createdTask); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			fmt.Printf("🔄 Created task via API: %s\n", createdTask.Description)
			fmt.Printf("✅ Task ID: %s\n", createdTask.ID)
		},
	}
	
	// Add interactive flag and priority flag
	cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode for task creation")
	cmd.Flags().StringP("priority", "p", "M", "Set task priority (L, M, H)")
	
	// If not in interactive mode, description is required
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		isInteractive, _ := cmd.Flags().GetBool("interactive")
		if !isInteractive && len(args) < 1 {
			return fmt.Errorf("requires a description when not in interactive mode")
		}
		return nil
	}
	
	return cmd
}

// task list
func newTaskListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List tasks from the API",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			status, _ := cmd.Flags().GetString("status")

			// Fetch tasks from the API
			// TODO: Add support for filtering by status and project via API query params
			resp, err := http.Get("http://localhost:8080/v1/tasks")
			if err != nil {
				log.Fatalf("Failed to fetch tasks from API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			var tasks []models.Task
			if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			if len(tasks) == 0 {
				fmt.Println("📋 No tasks found.")
				fmt.Println("💡 Create one with 'jbraincli task create <description>'")
				return
			}

			// Display beautiful task list
			// For now, we are listing from all projects, so projectName is empty and allProjects is true.
			displayTaskList(tasks, "", true, status)
		},
	}

	cmd.Flags().StringP("status", "s", "", "Filter by status (TODO, IN_PROGRESS, IN_REVIEW, COMPLETED) - (API support pending)")

	return cmd
}

// task start
func newTaskStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [id]",
		Short: "Start working on a task via API",
		Args:  cobra.ExactArgs(1), // Require exactly one argument: the task ID
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			// Create the JSON payload
			payload := map[string]string{"status": string(models.TaskStatusInProgress)}
			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("Failed to create JSON payload: %v", err)
			}

			// Create the PUT request
			url := fmt.Sprintf("http://localhost:8080/v1/tasks/%s/status", taskID)
			req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Fatalf("Failed to create HTTP request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalf("Failed to send request to API: %v", err)
			}
			defer resp.Body.Close()

			// Check the response
			if resp.StatusCode != http.StatusOK {
				var apiError map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&apiError); err == nil {
					log.Fatalf("API returned an error: %s (Status: %s)", apiError["error"], resp.Status)
				}
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			var updatedTask models.Task
			if err := json.NewDecoder(resp.Body).Decode(&updatedTask); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			fmt.Printf("▶️ Started task: %s\n", updatedTask.Description)
			fmt.Println("✅ Task status updated to IN_PROGRESS!")
		},
	}

	// TODO: Re-implement interactive mode by fetching TODO tasks from the API
	// cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode for task selection")

	return cmd
}

// task done
func newTaskDoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done [id]",
		Short: "Mark task as completed via API",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			// Create the JSON payload
			payload := map[string]string{"status": string(models.TaskStatusCompleted)}
			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("Failed to create JSON payload: %v", err)
			}

			// Create the PUT request
			url := fmt.Sprintf("http://localhost:8080/v1/tasks/%s/status", taskID)
			req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Fatalf("Failed to create HTTP request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalf("Failed to send request to API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				var apiError map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&apiError); err == nil {
					log.Fatalf("API returned an error: %s", apiError["error"])
				}
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			var updatedTask models.Task
			if err := json.NewDecoder(resp.Body).Decode(&updatedTask); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			fmt.Printf("✅ Completed task: %s\n", updatedTask.Description)
		},
	}

	// TODO: Re-implement interactive mode
	return cmd
}

// task info
func newTaskInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [id]",
		Short: "Get detailed information about a task from the API",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			url := fmt.Sprintf("http://localhost:8080/v1/tasks/%s", taskID)

			resp, err := http.Get(url)
			if err != nil {
				log.Fatalf("Failed to fetch task info from API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			var task models.Task
			if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			// Display detailed task information
			fmt.Printf("## Task Details: %s\n\n", task.ID)
			fmt.Printf("**Description:** %s\n", task.Description)
			fmt.Printf("**Status:** %s %s\n", getStatusIconForTask(string(task.Status)), task.Status)
			fmt.Printf("**Priority:** %s %s\n", getPriorityIconForTask(string(task.Priority)), task.Priority)
			fmt.Printf("**Progress:** %s %d%%\n", getProgressBar(task.Progress, 20), task.Progress)
			fmt.Printf("**Created:** %s\n", task.CreatedAt.Format("2006-01-02 15:04"))
			if task.CompletedAt != nil {
				fmt.Printf("**Completed:** %s\n", task.CompletedAt.Format("2006-01-02 15:04"))
			}
			fmt.Println("\n**Annotations:**")
			if len(task.Annotations) > 0 {
				for _, an := range task.Annotations {
					fmt.Printf("- %s (%s)\n", an.Content, an.CreatedAt.Format("2006-01-02 15:04"))
				}
			} else {
				fmt.Println("  No annotations yet.")
			}
		},
	}
	return cmd
}

// task progress
func newTaskProgressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress [id] [percentage]",
		Short: "Update the progress of a task via API",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			progress, err := strconv.Atoi(args[1])
			if err != nil || progress < 0 || progress > 100 {
				log.Fatalf("Invalid progress value. Please provide a number between 0 and 100.")
			}

			payload := map[string]interface{}{"progress": progress}
			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("Failed to create JSON payload: %v", err)
			}

			url := fmt.Sprintf("http://localhost:8080/v1/tasks/%s", taskID)
			req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Fatalf("Failed to create HTTP request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalf("Failed to send request to API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			fmt.Println("✅ Task progress updated successfully.")
		},
	}
	return cmd
}

func getTaskByIDPrefix(db *gorm.DB, idPrefix string) (*models.Task, error) {
	var task models.Task
	if err := db.Where("id::text LIKE ?", idPrefix+"%").First(&task).Error; err != nil {
		return nil, fmt.Errorf("task with ID prefix '%s' not found", idPrefix)
	}
	return &task, nil
}

// displayTaskList shows tasks in a beautiful, responsive format
func displayTaskList(tasks []models.Task, projectName string, allProjects bool, statusFilter string) {
	// Import terminal width detection
	var width int = 80 // default width
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		width = w
	}

	// Header with project info
	if allProjects {
		if statusFilter != "" {
			fmt.Printf("📋 %s Tasks from All Projects (%d)\n", strings.ToUpper(statusFilter), len(tasks))
		} else {
			fmt.Printf("📋 All Tasks from All Projects (%d)\n", len(tasks))
		}
	} else {
		if statusFilter != "" {
			fmt.Printf("📋 %s Tasks - %s (%d)\n", strings.ToUpper(statusFilter), projectName, len(tasks))
		} else {
			fmt.Printf("📋 Tasks - %s (%d)\n", projectName, len(tasks))
		}
	}

	// Generate unique short IDs (reuse from kanban)
	uniqueIDs := generateUniqueShortIDsForTasks(tasks)

	// Responsive design
	if width < 100 {
		// Compact view for narrow terminals
		displayTaskListCompact(tasks, uniqueIDs, allProjects)
	} else {
		// Full table view for wide terminals
		displayTaskListTable(tasks, uniqueIDs, allProjects, width)
	}
}

// displayTaskListCompact shows tasks in compact format
func displayTaskListCompact(tasks []models.Task, uniqueIDs map[string]string, allProjects bool) {
	fmt.Println()
	for i, task := range tasks {
		// Priority and status icons
		priorityIcon := getPriorityIconForTask(task.Priority)
		statusIcon := getStatusIconForTask(task.Status)
		
		// Progress indicator
		progressBar := getProgressBar(task.Progress, 8)
		
		fmt.Printf("%s %s %s %s\n", 
			priorityIcon, 
			statusIcon, 
			uniqueIDs[task.ID.String()], 
			task.Description)
		
		if allProjects && task.Project != nil {
			fmt.Printf("   🏢 %s", task.Project.Name)
		}
		
		if task.Progress > 0 {
			fmt.Printf("   %s %d%%", progressBar, task.Progress)
		}
		
		fmt.Println()
		
		// Add separator between tasks (except last)
		if i < len(tasks)-1 {
			fmt.Println("   ─────────────────────────────────────")
		}
	}
}

// displayTaskListTable shows tasks in full table format  
func displayTaskListTable(tasks []models.Task, uniqueIDs map[string]string, allProjects bool, termWidth int) {
	// Calculate dynamic column widths
	idWidth := 12
	priorityWidth := 4
	statusWidth := 12
	progressWidth := 12
	projectWidth := 0
	if allProjects {
		projectWidth = 20
	}
	
	// Remaining width for description
	usedWidth := idWidth + priorityWidth + statusWidth + progressWidth + projectWidth + 8 // borders and spaces
	descWidth := termWidth - usedWidth
	if descWidth < 30 {
		descWidth = 30
	}

	// Table header
	fmt.Println()
	if allProjects {
		fmt.Printf("┌─%-*s─┬─%-*s─┬─%-*s─┬─%-*s─┬─%-*s─┬─%-*s─┐\n", 
			idWidth, strings.Repeat("─", idWidth),
			priorityWidth, strings.Repeat("─", priorityWidth),
			statusWidth, strings.Repeat("─", statusWidth),
			progressWidth, strings.Repeat("─", progressWidth),
			projectWidth, strings.Repeat("─", projectWidth),
			descWidth, strings.Repeat("─", descWidth))
		
		fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
			idWidth, "ID",
			priorityWidth, "PRI",
			statusWidth, "STATUS",
			progressWidth, "PROGRESS",
			projectWidth, "PROJECT",
			descWidth, "DESCRIPTION")
	} else {
		fmt.Printf("┌─%-*s─┬─%-*s─┬─%-*s─┬─%-*s─┬─%-*s─┐\n", 
			idWidth, strings.Repeat("─", idWidth),
			priorityWidth, strings.Repeat("─", priorityWidth),
			statusWidth, strings.Repeat("─", statusWidth),
			progressWidth, strings.Repeat("─", progressWidth),
			descWidth, strings.Repeat("─", descWidth))
		
		fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
			idWidth, "ID",
			priorityWidth, "PRI", 
			statusWidth, "STATUS",
			progressWidth, "PROGRESS",
			descWidth, "DESCRIPTION")
	}

	// Separator
	if allProjects {
		fmt.Printf("├─%-*s─┼─%-*s─┼─%-*s─┼─%-*s─┼─%-*s─┼─%-*s─┤\n",
			idWidth, strings.Repeat("─", idWidth),
			priorityWidth, strings.Repeat("─", priorityWidth),
			statusWidth, strings.Repeat("─", statusWidth),
			progressWidth, strings.Repeat("─", progressWidth),
			projectWidth, strings.Repeat("─", projectWidth),
			descWidth, strings.Repeat("─", descWidth))
	} else {
		fmt.Printf("├─%-*s─┼─%-*s─┼─%-*s─┼─%-*s─┼─%-*s─┤\n",
			idWidth, strings.Repeat("─", idWidth),
			priorityWidth, strings.Repeat("─", priorityWidth),
			statusWidth, strings.Repeat("─", statusWidth),
			progressWidth, strings.Repeat("─", progressWidth),
			descWidth, strings.Repeat("─", descWidth))
	}

	// Task rows
	for _, task := range tasks {
		priorityIcon := getPriorityIconForTask(task.Priority)
		statusIcon := getStatusIconForTask(task.Status)
		progressBar := getProgressBar(task.Progress, 10)
		
		shortID := uniqueIDs[task.ID.String()]
		description := truncateString(task.Description, descWidth)
		
		if allProjects {
			projectName := ""
			if task.Project != nil {
				projectName = truncateString(task.Project.Name, projectWidth)
			}
			
			fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
				idWidth, shortID,
				priorityWidth, priorityIcon,
				statusWidth, statusIcon,
				progressWidth, progressBar,
				projectWidth, projectName,
				descWidth, description)
		} else {
			fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
				idWidth, shortID,
				priorityWidth, priorityIcon,
				statusWidth, statusIcon,
				progressWidth, progressBar,
				descWidth, description)
		}
	}

	// Table footer
	if allProjects {
		fmt.Printf("└─%-*s─┴─%-*s─┴─%-*s─┴─%-*s─┴─%-*s─┴─%-*s─┘\n",
			idWidth, strings.Repeat("─", idWidth),
			priorityWidth, strings.Repeat("─", priorityWidth),
			statusWidth, strings.Repeat("─", statusWidth),
			progressWidth, strings.Repeat("─", progressWidth),
			projectWidth, strings.Repeat("─", projectWidth),
			descWidth, strings.Repeat("─", descWidth))
	} else {
		fmt.Printf("└─%-*s─┴─%-*s─┴─%-*s─┴─%-*s─┴─%-*s─┘\n",
			idWidth, strings.Repeat("─", idWidth),
			priorityWidth, strings.Repeat("─", priorityWidth),
			statusWidth, strings.Repeat("─", statusWidth),
			progressWidth, strings.Repeat("─", progressWidth),
			descWidth, strings.Repeat("─", descWidth))
	}
}

// Helper functions for task list display
func generateUniqueShortIDsForTasks(tasks []models.Task) map[string]string {
	uniqueIDs := make(map[string]string)
	usedShortIDs := make(map[string][]string)
	
	// First pass: try 8-character IDs
	for _, task := range tasks {
		fullID := task.ID.String()
		shortID := fullID[:8]
		usedShortIDs[shortID] = append(usedShortIDs[shortID], fullID)
	}
	
	// Second pass: resolve collisions
	for shortID, fullIDs := range usedShortIDs {
		if len(fullIDs) == 1 {
			uniqueIDs[fullIDs[0]] = shortID
		} else {
			for _, fullID := range fullIDs {
				uniqueLen := 8
				for uniqueLen < len(fullID) {
					candidate := fullID[:uniqueLen]
					isUnique := true
					for _, otherID := range fullIDs {
						if otherID != fullID && len(otherID) > uniqueLen && otherID[:uniqueLen] == candidate {
							isUnique = false
							break
						}
					}
					if isUnique {
						break
					}
					uniqueLen++
				}
				uniqueIDs[fullID] = fullID[:uniqueLen]
			}
		}
	}
	
	return uniqueIDs
}

func getPriorityIconForTask(priority string) string {
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

func getStatusIconForTask(status string) string {
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

func getProgressBar(progress int, width int) string {
	if progress == 0 {
		return strings.Repeat("░", width)
	}
	if progress == 100 {
		return "✅ 100%"
	}
	
	filled := (progress * width) / 100
	bar := strings.Repeat("▓", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("%s %d%%", bar, progress)
}

func newTaskDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [id]",
		Short:   "Delete a task via API",
		Aliases: []string{"rm", "del"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			if !askForConfirmation("Are you sure you want to delete this task?") {
				fmt.Println("🚫 Delete operation cancelled.")
				return
			}

			// Create the DELETE request
			url := fmt.Sprintf("http://localhost:8080/v1/tasks/%s", taskID)
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				log.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalf("Failed to send request to API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			fmt.Println("✅ Task successfully deleted.")
		},
	}
	return cmd
}

func newTaskModifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify [id]",
		Short: "Modify a task's description or priority via API",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			newDesc, _ := cmd.Flags().GetString("description")
			newPriority, _ := cmd.Flags().GetString("priority")

			if newDesc == "" && newPriority == "" {
				fmt.Println("Please provide a new description (--description) or priority (--priority).")
				return
			}

			payload := make(map[string]interface{})
			if newDesc != "" {
				payload["description"] = newDesc
			}
			if newPriority != "" {
				payload["priority"] = newPriority
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("Failed to create JSON payload: %v", err)
			}

			url := fmt.Sprintf("http://localhost:8080/v1/tasks/%s", taskID)
			req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Fatalf("Failed to create HTTP request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalf("Failed to send request to API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			fmt.Println("✅ Task modified successfully.")
		},
	}
	cmd.Flags().StringP("description", "d", "", "New description for the task")
	cmd.Flags().StringP("priority", "p", "", "New priority for the task (L, M, H)")
	return cmd
}