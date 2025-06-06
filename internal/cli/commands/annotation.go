package commands

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/models"
	"github.com/terzigolu/josepshbrain-go/repository"
)

// NewAnnotationCmd creates the annotation command
func NewAnnotationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "annotate",
		Short:   "Add annotation to a task",
		Example: `jbraincli annotate a2e35246 "This is important to remember"`,
		Args:    cobra.ExactArgs(2),
		RunE:    createAnnotation,
	}

	return cmd
}

// NewTaskAnnotationsCmd creates the task-annotations command
func NewTaskAnnotationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task-annotations",
		Short:   "List annotations for a task",
		Example: `jbraincli task-annotations a2e35246`,
		Args:    cobra.ExactArgs(1),
		RunE:    listTaskAnnotations,
	}

	return cmd
}

func createAnnotation(cmd *cobra.Command, args []string) error {
	db := repository.GetDB()

	taskIDStr := args[0]
	content := args[1]

	// Find the task by UUID prefix or full UUID
	var task models.Task
	result := db.DB.Where("id LIKE ?", taskIDStr+"%").First(&task)
	if result.Error != nil {
		return fmt.Errorf("task not found: %v", result.Error)
	}

	// Create annotation
	annotation := models.Annotation{
		ID:        uuid.New(),
		TaskID:    uuid.MustParse(task.ID),
		Content:   content,
		CreatedAt: time.Now(),
	}

	result = db.DB.Create(&annotation)
	if result.Error != nil {
		return fmt.Errorf("failed to create annotation: %v", result.Error)
	}

	fmt.Printf("✅ Annotation added to task: %s\n", task.Description)
	fmt.Printf("   📝 %s\n", content)

	return nil
}

func listTaskAnnotations(cmd *cobra.Command, args []string) error {
	db := repository.GetDB()

	taskIDStr := args[0]

	// Find the task by UUID prefix or full UUID
	var task models.Task
	result := db.DB.Where("id LIKE ?", taskIDStr+"%").First(&task)
	if result.Error != nil {
		return fmt.Errorf("task not found: %v", result.Error)
	}

	// Get annotations for this task
	var annotations []models.Annotation
	result = db.DB.Where("task_id = ?", uuid.MustParse(task.ID)).Order("created_at DESC").Find(&annotations)
	if result.Error != nil {
		return fmt.Errorf("failed to get annotations: %v", result.Error)
	}

	if len(annotations) == 0 {
		fmt.Printf("📝 No annotations found for task: %s\n", task.Description)
		return nil
	}

	fmt.Printf("📝 Annotations for task: %s\n\n", task.Description)
	for i, annotation := range annotations {
		fmt.Printf("%d. %s\n", i+1, annotation.Content)
		fmt.Printf("   🕒 %s\n\n", annotation.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
} 