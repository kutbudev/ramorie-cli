package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewContextCommand context komutunu oluşturur
func NewContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Bağlam yönetimi komutları",
		Long:  `Bağlam oluşturma, listeleme ve yönetme komutları.`,
	}

	cmd.AddCommand(newContextCreateCommand())
	cmd.AddCommand(newContextListCommand())
	cmd.AddCommand(newContextUseCommand())

	return cmd
}

func newContextCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Yeni bağlam oluştur",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Creating context: %s\n", name)
			return nil
		},
	}

	return cmd
}

func newContextListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Bağlamları listele",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Listing contexts...")
			return nil
		},
	}

	return cmd
}

func newContextUseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <context-id>",
		Short: "Aktif bağlamı değiştir",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextID := args[0]
			fmt.Printf("Setting active context: %s\n", contextID)
			return nil
		},
	}

	return cmd
}
