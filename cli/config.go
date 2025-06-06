package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewConfigCommand config komutunu oluşturur
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Konfigürasyon yönetimi komutları",
		Long: `Uygulama konfigürasyonunu görüntüleme ve yönetme komutları.`,
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigSetCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Mevcut konfigürasyonu göster",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Showing configuration...")
			return nil
		},
	}

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Konfigürasyon değeri ayarla",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			fmt.Printf("Setting config: %s = %s\n", key, value)
			return nil
		},
	}

	return cmd
} 