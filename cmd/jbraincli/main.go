package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/internal/cli/commands"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "jbraincli",
		Short: "Joseph's Brain CLI - A tool for managing your tasks and knowledge.",
		Long: `jbraincli is a powerful command-line interface to interact with Joseph's Brain,
a system designed to help you manage your projects, tasks, and memories efficiently.`,
	}

	// Add all commands to the root command
	rootCmd.AddCommand(commands.NewProjectCmd())     // API-driven
	rootCmd.AddCommand(commands.NewTaskCmd())        // API-driven
	rootCmd.AddCommand(commands.NewMemoryCmd())      // API-driven
	rootCmd.AddCommand(commands.NewKanbanCmd())      // API-driven
	rootCmd.AddCommand(commands.NewAnnotateCmd())    // API-driven
	rootCmd.AddCommand(commands.NewTaskAnnotationsCmd()) // API-driven
	rootCmd.AddCommand(commands.NewRememberCmd())    // API-driven shortcut

	// TODO: Re-enable these commands after implementing them
	// rootCmd.AddCommand(commands.NewVersionCmd())
	// rootCmd.AddCommand(commands.NewMCPCmd(cfg))

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints the error, so we just need to exit.
		os.Exit(1)
	}
} 