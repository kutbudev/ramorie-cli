package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
	"gorm.io/gorm"
)

// NewProjectCmd creates the project command with all subcommands
func NewProjectCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Project management commands",
		Long:  "Create, list, and manage projects",
	}

	// Add subcommands
	cmd.AddCommand(newProjectInitCmd(db))
	cmd.AddCommand(newProjectUseCmd(db))
	cmd.AddCommand(newProjectListCmd(db))
	cmd.AddCommand(newProjectSelectCmd(db))

	return cmd
}

// project init - create new project
func newProjectInitCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "init [name]",
		Short:   "Create a new project",
		Aliases: []string{"create"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]

			// Check if project already exists
			var existingProject models.Project
			result := db.Where("name = ? AND deleted_at IS NULL", projectName).First(&existingProject)
			if result.Error == nil {
				fmt.Printf("❌ Project '%s' already exists\n", projectName)
				return
			}

			// Deactivate all other projects first
			if err := db.Model(&models.Project{}).Where("is_active = ? AND deleted_at IS NULL", true).Update("is_active", false).Error; err != nil {
				log.Fatalf("Failed to deactivate existing projects: %v", err)
			}

			// Create new project
			project := models.Project{
				Name:        projectName,
				Description: stringPtr(fmt.Sprintf("Project: %s", projectName)),
				IsActive:    true,
			}

			if err := db.Create(&project).Error; err != nil {
				log.Fatalf("Failed to create project: %v", err)
			}

			fmt.Printf("✨ Created and activated project: %s\n", projectName)
			fmt.Printf("📋 Project ID: %s\n", project.ID.String())
		},
	}
}

// project use - set active project or show current
func newProjectUseCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "use [name]",
		Short: "Set active project or show current active project",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Show current active project
				var activeProject models.Project
				result := db.Where("is_active = ? AND deleted_at IS NULL", true).First(&activeProject)
				if result.Error != nil {
					fmt.Println("❌ No active project found")
					fmt.Println("💡 Use 'jbraincli project init <n>' to create a project")
					fmt.Println("💡 Or use 'jbraincli project select' for interactive selection")
					return
				}
				fmt.Printf("📋 Active project: %s\n", activeProject.Name)
				if activeProject.Description != nil {
					fmt.Printf("📝 Description: %s\n", *activeProject.Description)
				}
				fmt.Printf("🆔 ID: %s\n", activeProject.ID.String())
				return
			}

			projectName := args[0]

			// Find the project
			var project models.Project
			result := db.Where("name = ? AND deleted_at IS NULL", projectName).First(&project)
			if result.Error != nil {
				fmt.Printf("❌ Project '%s' not found\n", projectName)
				fmt.Println("💡 Use 'jbraincli project list' to see available projects")
				fmt.Println("💡 Or use 'jbraincli project select' for interactive selection")
				return
			}

			// Deactivate all projects first
			if err := db.Model(&models.Project{}).Where("deleted_at IS NULL").Update("is_active", false).Error; err != nil {
				log.Fatalf("Failed to deactivate projects: %v", err)
			}

			// Activate the selected project
			project.IsActive = true
			if err := db.Save(&project).Error; err != nil {
				log.Fatalf("Failed to activate project: %v", err)
			}

			fmt.Printf("✅ Activated project: %s\n", projectName)
		},
	}
}

// project select - interactive project selection
func newProjectSelectCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "select",
		Short:   "Interactively select and activate a project",
		Aliases: []string{"choose"},
		Run: func(cmd *cobra.Command, args []string) {
			selectProjectInteractively(db)
		},
	}
}

// project list - list all projects
func newProjectListCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all projects",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			interactive, _ := cmd.Flags().GetBool("interactive")
			if interactive {
				selectProjectInteractively(db)
				return
			}
			
			var projects []models.Project
			if err := db.Where("deleted_at IS NULL").Find(&projects).Error; err != nil {
				log.Fatalf("Failed to fetch projects: %v", err)
			}

			if len(projects) == 0 {
				fmt.Println("📋 No projects found. Create one with 'jbraincli project init <n>'")
				return
			}

			fmt.Println("📋 Project List:")
			fmt.Println("┌─────────────────────────────────────────┬───────────────────────────┬──────────┬────────────────────────┐")
			fmt.Println("│ ID                                      │ Name                      │ Active   │ Description            │")
			fmt.Println("├─────────────────────────────────────────┼───────────────────────────┼──────────┼────────────────────────┤")

			for _, project := range projects {
				activeStatus := "❌"
				if project.IsActive {
					activeStatus = "✅"
				}
				
				description := ""
				if project.Description != nil {
					description = truncateString(*project.Description, 20)
				}

				fmt.Printf("│ %-39s │ %-25s │ %-8s │ %-22s │\n",
					project.ID.String()[:8]+"...",
					truncateString(project.Name, 25),
					activeStatus,
					description)
			}
			fmt.Println("└─────────────────────────────────────────┴───────────────────────────┴──────────┴────────────────────────┘")
			fmt.Println()
			fmt.Println("💡 Use 'jbraincli project select' for interactive selection")
		},
	}
	
	cmd.Flags().BoolP("interactive", "i", false, "Interactive project selection")
	return cmd
}

// selectProjectInteractively shows an interactive menu for project selection
func selectProjectInteractively(db *gorm.DB) {
	var projects []models.Project
	if err := db.Where("deleted_at IS NULL").Find(&projects).Error; err != nil {
		log.Fatalf("Failed to fetch projects: %v", err)
	}

	if len(projects) == 0 {
		fmt.Println("📋 No projects found. Create one with 'jbraincli project init <n>'")
		return
	}

	fmt.Println("🎯 Select a project to activate:")
	fmt.Println()
	
	// Show numbered list
	for i, project := range projects {
		activeStatus := ""
		if project.IsActive {
			activeStatus = " (current)"
		}
		
		fmt.Printf("  %d) %s%s\n", i+1, project.Name, activeStatus)
		if project.Description != nil && *project.Description != "" {
			fmt.Printf("     📝 %s\n", *project.Description)
		}
	}
	
	fmt.Println()
	fmt.Printf("Enter number (1-%d) or 'q' to quit: ", len(projects))
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("❌ Error reading input: %v\n", err)
		return
	}
	
	input = strings.TrimSpace(input)
	
	if input == "q" || input == "quit" {
		fmt.Println("👋 Selection cancelled")
		return
	}
	
	// Parse number
	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(projects) {
		fmt.Printf("❌ Invalid selection. Please enter a number between 1 and %d\n", len(projects))
		return
	}
	
	selectedProject := projects[num-1]
	
	// Deactivate all projects first
	if err := db.Model(&models.Project{}).Where("deleted_at IS NULL").Update("is_active", false).Error; err != nil {
		log.Fatalf("Failed to deactivate projects: %v", err)
	}

	// Activate the selected project
	selectedProject.IsActive = true
	if err := db.Save(&selectedProject).Error; err != nil {
		log.Fatalf("Failed to activate project: %v", err)
	}

	fmt.Printf("✅ Activated project: %s\n", selectedProject.Name)
	fmt.Printf("🆔 ID: %s\n", selectedProject.ID.String())
}

 