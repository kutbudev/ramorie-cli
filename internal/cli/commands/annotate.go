package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/urfave/cli/v2"
)

// NewAnnotateCmd creates the 'annotate' command using urfave/cli.
func NewAnnotateCmd() *cli.Command {
	return &cli.Command{
		Name:      "annotate",
		Usage:     "Add an annotation to a task",
		ArgsUsage: "[task-id] [content]",
		Action: func(c *cli.Context) error {
			if c.NArg() != 2 {
				return fmt.Errorf("task ID and content are required")
			}
			taskID := c.Args().Get(0)
			content := c.Args().Get(1)

			client := api.NewClient()
			annotation, err := client.CreateAnnotation(taskID, content)
			if err != nil {
				return fmt.Errorf("error creating annotation: %w", err)
			}

			fmt.Printf("📝 Annotation added successfully!\n")
			fmt.Printf("Task ID: %s\n", annotation.TaskID.String())
			fmt.Printf("Content: %s\n", annotation.Content)
			fmt.Printf("Created: %s\n", annotation.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

// NewTaskAnnotationsCmd creates the 'task-annotations' command using urfave/cli.
func NewTaskAnnotationsCmd() *cli.Command {
	return &cli.Command{
		Name:      "task-annotations",
		Usage:     "List all annotations for a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()
			annotations, err := client.ListAnnotations(taskID)
			if err != nil {
				return fmt.Errorf("error listing annotations: %w", err)
			}

			if len(annotations) == 0 {
				fmt.Printf("No annotations found for task %s\n", taskID)
				return nil
			}

			fmt.Printf("📝 Annotations for task %s:\n\n", taskID[:8])

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tCONTENT\tCREATED")
			fmt.Fprintln(w, "--\t-------\t-------")

			for _, annotation := range annotations {
				shortID := annotation.ID.String()[:8]
				content := strings.ReplaceAll(annotation.Content, "\n", " ")
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					shortID,
					truncateString(content, 50),
					annotation.CreatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()

			fmt.Printf("\n📋 Full annotations:\n")
			for i, annotation := range annotations {
				fmt.Printf("\n%d. [%s] %s\n", i+1, annotation.CreatedAt.Format("2006-01-02 15:04"), annotation.Content)
			}
			return nil
		},
	}
}
