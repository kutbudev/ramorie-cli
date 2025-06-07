package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/internal/api"
)

// NewAnnotateCmd creates the annotate command, fully API-driven.
func NewAnnotateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotate [task-id] [content]",
		Short: "Add an annotation to a task",
		Long:  "Add progress notes, decisions, or technical details to a task.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]
			content := args[1]

			client := api.NewClient()
			annotation, err := client.CreateAnnotation(taskID, content)
			if err != nil {
				fmt.Printf("Error creating annotation: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("📝 Annotation added successfully!\n")
			fmt.Printf("Task ID: %s\n", annotation.TaskID.String())
			fmt.Printf("Content: %s\n", annotation.Content)
			fmt.Printf("Created: %s\n", annotation.CreatedAt.Format("2006-01-02 15:04:05"))
		},
	}

	return cmd
}

// NewTaskAnnotationsCmd creates the task-annotations command
func NewTaskAnnotationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task-annotations [task-id]",
		Short: "List all annotations for a task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			taskID := args[0]

			client := api.NewClient()
			annotations, err := client.ListAnnotations(taskID)
			if err != nil {
				fmt.Printf("Error listing annotations: %v\n", err)
				os.Exit(1)
			}

			if len(annotations) == 0 {
				fmt.Printf("No annotations found for task %s\n", taskID)
				return
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

			// Show full content for each annotation
			fmt.Printf("\n📋 Full annotations:\n")
			for i, annotation := range annotations {
				fmt.Printf("\n%d. [%s] %s\n", i+1, annotation.CreatedAt.Format("2006-01-02 15:04"), annotation.Content)
			}
		},
	}

	return cmd
}