package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/config"
	"github.com/terzigolu/josepshbrain-go/internal/cli/commands"
	"github.com/terzigolu/josepshbrain-go/repository"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := repository.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	var rootCmd = &cobra.Command{
		Use:   "jbraincli",
		Short: "🧠 JosephsBrain CLI - Go Edition",
		Long:  `AI-powered task and memory management CLI tool

Available commands:
  task      - Task management (create, list, start, done)
  project   - Project management (init, use, list)
  remember  - Memory management (add, search, recall)
  context   - Context management (create, use, list)
  kanban    - Kanban board view

Use 'jbraincli [command] --help' for command details`,
	}

	// Add modular commands with dependencies
	rootCmd.AddCommand(commands.NewTaskCmd(db.DB))
	rootCmd.AddCommand(commands.NewProjectCmd(db.DB))
	rootCmd.AddCommand(commands.NewMemoryCmd(db.DB))
	rootCmd.AddCommand(commands.NewKanbanCmd(db.DB))
	
	// Add root-level shortcuts for memory commands (like TS version)
	rootCmd.AddCommand(commands.NewRememberCmd(db.DB))
	rootCmd.AddCommand(commands.NewMemoriesCmd(db.DB))

	// Add annotation commands
	rootCmd.AddCommand(commands.NewAnnotationCmd())
	rootCmd.AddCommand(commands.NewTaskAnnotationsCmd())

	// Add root-level shortcuts for common project commands (like TS version)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "init [name]",
		Short: "Create a new project (shortcut for 'project init')",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			commands.NewProjectCmd(db.DB).Commands()[0].Run(cmd, args)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "use [name]",
		Short: "Set active project or show current (shortcut for 'project use')",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			commands.NewProjectCmd(db.DB).Commands()[1].Run(cmd, args)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "projects",
		Short: "List all projects (shortcut for 'project list')",
		Run: func(cmd *cobra.Command, args []string) {
			commands.NewProjectCmd(db.DB).Commands()[2].Run(cmd, args)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
} 