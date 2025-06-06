package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/config"
	"github.com/terzigolu/josepshbrain-go/database"
	"github.com/terzigolu/josepshbrain-go/internal/cli/commands"
)

func main() {
	// Load application configuration from config.yml
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load application configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.Initialize(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	var rootCmd = &cobra.Command{
		Use:   "jbraincli",
		Short: "Joseph's Brain CLI - A tool for managing your tasks and knowledge.",
		Long: `jbraincli is a powerful command-line interface to interact with Joseph's Brain,
a system designed to help you manage your projects, tasks, and memories efficiently.`,
		Run: func(cmd *cobra.Command, args []string) {
			// When no command is given, check for an active project.
			// If active, show kanban. If not, show help.
			cliCfg, err := config.LoadCliConfig()
			if err == nil && cliCfg.ActiveProjectID != "" {
				fmt.Println("🎯 Active project found. Showing Kanban board by default.")
				// We need a way to find the project name from ID via API if we want to display it
				// For now, just run the command
				kanbanCmd := commands.NewKanbanCmd(db)
				kanbanCmd.Run(cmd, args)
			} else {
				fmt.Println("ℹ️ No active project. Use 'jbraincli project use <name>' to select one.")
				fmt.Println("   Or 'jbraincli project init <name>' to create a new one.")
				fmt.Println()
				cmd.Help()
			}
		},
	}

	// Add all commands to the root command
	rootCmd.AddCommand(commands.NewVersionCmd())
	rootCmd.AddCommand(commands.NewMCPCmd(cfg))
	rootCmd.AddCommand(commands.NewProjectCmd())      // API-driven, no 'db'
	rootCmd.AddCommand(commands.NewTaskCmd(db))        // Still uses 'db'
	rootCmd.AddCommand(commands.NewMemoryCmd(db))      // Still uses 'db'
	rootCmd.AddCommand(commands.NewKanbanCmd(db))      // Still uses 'db'
	rootCmd.AddCommand(commands.NewAnnotationCmd(db))  // Still uses 'db'

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints the error, so we just need to exit.
		os.Exit(1)
	}
} 