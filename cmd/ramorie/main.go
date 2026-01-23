package main

import (
	"log"
	"os"

	"github.com/kutbudev/ramorie-cli/internal/cli/commands"
	"github.com/urfave/cli/v2"
)

// Version will be set during build with ldflags
var Version = "3.6.0"

func main() {
	app := &cli.App{
		Name:    "ramorie",
		Usage:   "AI-powered task and memory management CLI",
		Version: Version,
		Commands: []*cli.Command{
			// Core commands
			commands.NewSetupCommand(),
			commands.NewVaultCommand(), // Alias for vault operations (ramorie vault unlock)
			commands.NewTaskCommand(),
			commands.NewProjectCommand(),
			commands.NewMemoryCommand(),
			commands.NewRememberCommand(),

			// Reports & Views
			commands.NewReportsCommand(),
			commands.NewKanbanCmd(),

			// Relations
			commands.NewTaskMemoriesCommand(),
			commands.NewMemoryTasksCommand(),
			commands.NewLinkCommand(),

			// Annotations
			commands.NewAnnotateCmd(),
			commands.NewTaskAnnotationsCmd(),

			// Context & Focus
			commands.NewContextCommand(),
			commands.NewContextPackCommand(),
			commands.NewFocusCommand(), // NEW

			// Decisions (ADRs)
			commands.NewDecisionCommand(), // NEW

			// Organizations
			commands.NewOrganizationCommand(), // NEW

			// AI Features
			commands.NewAICommand(), // NEW

			// Planning
			commands.NewPlanCommand(), // Multi-agent AI planning

			// Subtasks
			commands.NewSubtaskCommand(),

			// Meta
			commands.NewOverviewCommand(),
			commands.NewMcpCommand(),
			commands.NewConfigCommand(),
			commands.NewGeminiKeyCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
