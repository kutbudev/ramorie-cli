package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTaskCommand task komutunu oluşturur
func NewTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Görev yönetimi komutları",
		Long: `Görevleri oluşturma, listeleme, güncelleme ve silme işlemleri için komutlar.

Örnekler:
  ramorie task create "Yeni özellik geliştir" --priority high
  ramorie task list --status todo
  ramorie task start <task-id>
  ramorie task complete <task-id>`,
	}

	// Alt komutları ekle
	cmd.AddCommand(newTaskCreateCommand())
	cmd.AddCommand(newTaskListCommand())
	cmd.AddCommand(newTaskShowCommand())
	cmd.AddCommand(newTaskUpdateCommand())
	cmd.AddCommand(newTaskStartCommand())
	cmd.AddCommand(newTaskCompleteCommand())
	cmd.AddCommand(newTaskDeleteCommand())

	return cmd
}

// newTaskCreateCommand yeni görev oluşturma komutu
func newTaskCreateCommand() *cobra.Command {
	var (
		projectID string
		contextID string
		priority  string
		tags      []string
		dueDate   string
	)

	cmd := &cobra.Command{
		Use:   "create <description>",
		Short: "Yeni görev oluştur",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := args[0]

			// TODO: Task service çağrısı
			fmt.Printf("Creating task: %s\n", description)
			fmt.Printf("Project ID: %s\n", projectID)
			fmt.Printf("Priority: %s\n", priority)
			fmt.Printf("Tags: %v\n", tags)

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Proje ID'si")
	cmd.Flags().StringVarP(&contextID, "context", "c", "", "Bağlam ID'si")
	cmd.Flags().StringVar(&priority, "priority", "medium", "Öncelik (low, medium, high)")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Etiketler (virgülle ayrılmış)")
	cmd.Flags().StringVar(&dueDate, "due", "", "Son tarih (YYYY-MM-DD)")

	return cmd
}

// newTaskListCommand görev listeleme komutu
func newTaskListCommand() *cobra.Command {
	var (
		projectID string
		status    string
		priority  string
		limit     int
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "Görevleri listele",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Task service çağrısı
			fmt.Printf("Listing tasks...\n")
			fmt.Printf("Project ID: %s\n", projectID)
			fmt.Printf("Status: %s\n", status)
			fmt.Printf("Priority: %s\n", priority)
			fmt.Printf("Limit: %d\n", limit)

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Proje ID'si ile filtrele")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Durum ile filtrele (todo, in_progress, in_review, completed)")
	cmd.Flags().StringVar(&priority, "priority", "", "Öncelik ile filtrele (low, medium, high)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Gösterilecek maksimum görev sayısı")

	return cmd
}

// newTaskShowCommand görev detaylarını gösterme komutu
func newTaskShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "Görev detaylarını göster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: Task service çağrısı
			fmt.Printf("Showing task: %s\n", taskID)

			return nil
		},
	}

	return cmd
}

// newTaskUpdateCommand görev güncelleme komutu
func newTaskUpdateCommand() *cobra.Command {
	var (
		description string
		priority    string
		progress    int
		status      string
	)

	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Görevi güncelle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: Task service çağrısı
			fmt.Printf("Updating task: %s\n", taskID)
			if description != "" {
				fmt.Printf("New description: %s\n", description)
			}
			if priority != "" {
				fmt.Printf("New priority: %s\n", priority)
			}
			if progress >= 0 {
				fmt.Printf("New progress: %d%%\n", progress)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Yeni açıklama")
	cmd.Flags().StringVar(&priority, "priority", "", "Yeni öncelik (low, medium, high)")
	cmd.Flags().IntVar(&progress, "progress", -1, "İlerleme yüzdesi (0-100)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Yeni durum (todo, in_progress, in_review, completed)")

	return cmd
}

// newTaskStartCommand görev başlatma komutu
func newTaskStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <task-id>",
		Short: "Görevi başlat (durumu IN_PROGRESS yapar)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: Task service çağrısı
			fmt.Printf("Starting task: %s\n", taskID)

			return nil
		},
	}

	return cmd
}

// newTaskCompleteCommand görev tamamlama komutu
func newTaskCompleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete <task-id>",
		Short: "Görevi tamamla (durumu COMPLETED yapar)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: Task service çağrısı
			fmt.Printf("Completing task: %s\n", taskID)

			return nil
		},
	}

	return cmd
}

// newTaskDeleteCommand görev silme komutu
func newTaskDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Görevi sil",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: Task service çağrısı
			fmt.Printf("Deleting task: %s\n", taskID)
			if force {
				fmt.Println("Force delete enabled")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Onay almadan sil")

	return cmd
}
