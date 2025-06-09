package main

import (
	"log"
	"os"

	"github.com/terzigolu/josepshbrain-go/internal/cli/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "jbraincli",
		Usage: "A CLI for interacting with the JosephsBrain API",
		Commands: []*cli.Command{
			commands.NewSetupCommand(),
			commands.NewTaskCommand(),
			commands.NewProjectCommand(),
			commands.NewMemoryCommand(),
			commands.NewRememberCommand(), // Direct remember command
			commands.NewKanbanCmd(),
			commands.NewAnnotateCmd(),
			commands.NewTaskAnnotationsCmd(),
			commands.NewContextCommand(),
			commands.NewConfigCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
} 