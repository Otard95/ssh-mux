package cmd

import (
	"fmt"

	"github.com/Otard95/ssh-mux/mux"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [destination]",
	Short: "Show active ControlMaster connections",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dest := ""
		if len(args) > 0 {
			dest = args[0]
		}
		if err := mux.Status(dest); err != nil {
			return fmt.Errorf("status failed: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
