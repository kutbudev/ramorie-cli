package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
	"gorm.io/gorm"
)

// NewMemoryCmd creates the memory command with all subcommands
func NewMemoryCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Memory management commands",
		Long:  "Create, list, and manage memories",
	}

	// Add subcommands
	cmd.AddCommand(newMemoryAddCmd(db))
	cmd.AddCommand(newMemoryListCmd(db))
	cmd.AddCommand(newMemoryRecallCmd(db))
	cmd.AddCommand(newMemoryForgetCmd(db))

	return cmd
}

// NewRememberCmd creates the remember command (shortcut for memory add)
func NewRememberCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "remember [text]",
		Short: "Store a new memory",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			createMemory(db, strings.Join(args, " "))
		},
	}
}

// NewMemoriesCmd creates the memories command (shortcut for memory list)
func NewMemoriesCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "memories",
		Short:   "List all memories",
		Aliases: []string{"recall"},
		Run: func(cmd *cobra.Command, args []string) {
			all, _ := cmd.Flags().GetBool("all")
			listMemories(db, "", all)
		},
	}
	
	cmd.Flags().BoolP("all", "a", false, "Show memories from all projects")
	return cmd
}

// memory add - create new memory
func newMemoryAddCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "add [text]",
		Short:   "Store a new memory",
		Aliases: []string{"create"},
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			createMemory(db, strings.Join(args, " "))
		},
	}
}

// memory list - list all memories
func newMemoryListCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all memories",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			search, _ := cmd.Flags().GetString("search")
			all, _ := cmd.Flags().GetBool("all")
			listMemories(db, search, all)
		},
	}
	
	cmd.Flags().StringP("search", "s", "", "Search term to filter memories")
	cmd.Flags().BoolP("all", "a", false, "Show memories from all projects")
	return cmd
}

// memory recall - search memories
func newMemoryRecallCmd(db *gorm.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall [search_term]",
		Short: "Search and recall memories",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			searchTerm := strings.Join(args, " ")
			all, _ := cmd.Flags().GetBool("all")
			listMemories(db, searchTerm, all)
		},
	}
	
	cmd.Flags().BoolP("all", "a", false, "Show memories from all projects")
	return cmd
}

// memory forget - delete memory
func newMemoryForgetCmd(db *gorm.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "forget [memory_id]",
		Short: "Delete a memory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			memoryID := args[0]
			forgetMemory(db, memoryID)
		},
	}
}

// Helper functions

func createMemory(db *gorm.DB, text string) {
	// Get active project
	var activeProject models.Project
	result := db.Where("is_active = ? AND deleted_at IS NULL", true).First(&activeProject)
	if result.Error != nil {
		fmt.Println("❌ No active project found")
		fmt.Println("💡 Use 'jbraincli project init <n>' to create a project")
		return
	}

	// Create new memory in memory_items table (based on schema)
	memoryItem := models.MemoryItem{
		ProjectID: &activeProject.ID,
		Content:   text,
	}

	if err := db.Create(&memoryItem).Error; err != nil {
		log.Fatalf("Failed to create memory: %v", err)
	}

	fmt.Printf("🧠 Memory stored successfully!\n")
	fmt.Printf("📋 Memory ID: %s\n", memoryItem.ID.String()[:8]+"...")
	fmt.Printf("📝 Content: %s\n", truncateString(text, 100))
	fmt.Printf("📁 Project: %s\n", activeProject.Name)
}

func listMemories(db *gorm.DB, searchTerm string, showAll bool) {
	var activeProject models.Project
	var projectName string
	
	if !showAll {
		// Get active project
		result := db.Where("is_active = ? AND deleted_at IS NULL", true).First(&activeProject)
		if result.Error != nil {
			fmt.Println("❌ No active project found")
			fmt.Println("💡 Use --all flag to see all memories or set an active project")
			return
		}
		projectName = activeProject.Name
	}

	// Build query for memory_items table
	query := db.Where("deleted_at IS NULL")
	
	if !showAll {
		query = query.Where("project_id = ?", activeProject.ID)
	}
	
	if searchTerm != "" {
		query = query.Where("content ILIKE ?", "%"+searchTerm+"%")
	}

	var memories []models.MemoryItem
	if err := query.Order("created_at DESC").Find(&memories).Error; err != nil {
		log.Fatalf("Failed to fetch memories: %v", err)
	}

	if len(memories) == 0 {
		if searchTerm != "" {
			fmt.Printf("🔍 No memories found matching '%s'\n", searchTerm)
		} else {
			if showAll {
				fmt.Println("🧠 No memories found. Create one with 'jbraincli remember <text>'")
			} else {
				fmt.Printf("🧠 No memories found for project '%s'. Create one with 'jbraincli remember <text>'\n", projectName)
			}
		}
		return
	}

	if searchTerm != "" {
		fmt.Printf("🔍 Search results for '%s' (%d found):\n", searchTerm, len(memories))
	} else {
		if showAll {
			fmt.Printf("🧠 All memories (%d total):\n", len(memories))
		} else {
			fmt.Printf("🧠 Memories for project '%s' (%d total):\n", projectName, len(memories))
		}
	}
	
	fmt.Println("┌─────────────────────────────────────────┬────────────────────────┬─────────────────────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ ID                                      │ Created                │ Content                                                                                         │")
	fmt.Println("├─────────────────────────────────────────┼────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────┤")

	for _, memory := range memories {
		fmt.Printf("│ %-39s │ %-22s │ %-103s │\n",
			memory.ID.String()[:8]+"...",
			memory.CreatedAt.Format("2006-01-02 15:04"),
			truncateString(memory.Content, 103))
	}
	fmt.Println("└─────────────────────────────────────────┴────────────────────────┴─────────────────────────────────────────────────────────────────────────────────────────────────┘")
}

func forgetMemory(db *gorm.DB, memoryID string) {
	// Find memory by ID (partial match) in memory_items table
	var memory models.MemoryItem
	result := db.Where("id::text LIKE ? AND deleted_at IS NULL", memoryID+"%").First(&memory)
	if result.Error != nil {
		fmt.Printf("❌ Memory with ID '%s' not found\n", memoryID)
		return
	}

	// Soft delete the memory
	if err := db.Model(&memory).Update("deleted_at", time.Now()).Error; err != nil {
		log.Fatalf("Failed to delete memory: %v", err)
	}

	fmt.Printf("🗑️  Memory deleted successfully!\n")
	fmt.Printf("📋 Memory ID: %s\n", memory.ID.String()[:8]+"...")
	fmt.Printf("📝 Content: %s\n", truncateString(memory.Content, 100))
} 