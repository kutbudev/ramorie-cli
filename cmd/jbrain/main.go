package main

import (
	"log"
	"os"

	"github.com/terzigolu/josepshbrain-go/internal/cli/commands"
	"github.com/urfave/cli/v2"
)

// Version will be set during build with ldflags
var Version = "dev"

func main() {
	app := &cli.App{
		Name:    "ramorie",
		Usage:   "AI-powered task and memory management CLI",
		Version: Version,
		Commands: []*cli.Command{
			commands.NewSetupCommand(),
			commands.NewTaskCommand(),
			commands.NewProjectCommand(),
			commands.NewMemoryCommand(),
			commands.NewRememberCommand(), // Direct remember command
			commands.NewReportsCommand(),
			commands.NewTaskMemoriesCommand(),
			commands.NewMemoryTasksCommand(),
			commands.NewLinkCommand(),
			commands.NewKanbanCmd(),
			commands.NewAnnotateCmd(),
			commands.NewTaskAnnotationsCmd(),
			commands.NewContextCommand(),
			commands.NewSubtaskCommand(),
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
