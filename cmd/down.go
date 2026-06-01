package cmd

import (
	"fmt"

	"github.com/Otard95/ssh-mux/mux"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down [destination]",
	Short: "Tear down ControlMaster connections",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dest := ""
		if len(args) > 0 {
			dest = args[0]
		}
		if err := mux.Down(dest); err != nil {
			return fmt.Errorf("down failed: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
