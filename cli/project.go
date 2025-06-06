package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewProjectCommand project komutunu oluşturur
func NewProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Proje yönetimi komutları",
		Long: `Projeleri oluşturma, listeleme, güncelleme ve silme işlemleri için komutlar.
		
Örnekler:
  jbraincli project create "My New Project" --description "Proje açıklaması"
  jbraincli project list
  jbraincli project use <project-id>
  jbraincli project show <project-id>`,
	}

	// Alt komutları ekle
	cmd.AddCommand(newProjectCreateCommand())
	cmd.AddCommand(newProjectListCommand())
	cmd.AddCommand(newProjectShowCommand())
	cmd.AddCommand(newProjectUseCommand())
	cmd.AddCommand(newProjectDeleteCommand())

	return cmd
}

func newProjectCreateCommand() *cobra.Command {
	var (
		description string
		path        string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Yeni proje oluştur",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// TODO: Project service çağrısı
			fmt.Printf("Creating project: %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Proje açıklaması")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Proje dosya yolu")

	return cmd
}

func newProjectListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "Projeleri listele",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Project service çağrısı
			fmt.Println("Listing projects...")
			return nil
		},
	}

	return cmd
}

func newProjectShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <project-id>",
		Short: "Proje detaylarını göster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			// TODO: Project service çağrısı
			fmt.Printf("Showing project: %s\n", projectID)
			return nil
		},
	}

	return cmd
}

func newProjectUseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <project-id>",
		Short: "Aktif projeyi değiştir",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			// TODO: Project service çağrısı
			fmt.Printf("Setting active project: %s\n", projectID)
			return nil
		},
	}

	return cmd
}

func newProjectDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Projeyi sil",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			// TODO: Project service çağrısı
			fmt.Printf("Deleting project: %s\n", projectID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Onay almadan sil")

	return cmd
} 