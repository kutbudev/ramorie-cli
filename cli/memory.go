package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewMemoryCommand memory komutunu oluşturur
func NewMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Hafıza yönetimi komutları",
		Long: `Hafıza öğelerini oluşturma, listeleme ve yönetme komutları.
		
Örnekler:
  jbraincli memory add "Önemli bilgi parçası"
  jbraincli memory list
  jbraincli memory search "anahtar kelime"`,
	}

	cmd.AddCommand(newMemoryAddCommand())
	cmd.AddCommand(newMemoryListCommand())
	cmd.AddCommand(newMemorySearchCommand())

	return cmd
}

func newMemoryAddCommand() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "add <content>",
		Short: "Yeni hafıza öğesi ekle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content := args[0]
			// TODO: Memory service çağrısı
			fmt.Printf("Adding memory: %s\n", content)
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Etiketler")
	return cmd
}

func newMemoryListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Hafıza öğelerini listele",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Memory service çağrısı
			fmt.Println("Listing memories...")
			return nil
		},
	}

	return cmd
}

func newMemorySearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Hafıza öğelerinde ara",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			// TODO: Memory service çağrısı
			fmt.Printf("Searching memories: %s\n", query)
			return nil
		},
	}

	return cmd
} 