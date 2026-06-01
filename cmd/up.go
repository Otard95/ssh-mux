package cmd

import (
	"fmt"

	"github.com/Otard95/ssh-mux/mux"
	"github.com/spf13/cobra"
)

var upPort int

var upCmd = &cobra.Command{
	Use:   "up <destination>",
	Short: "Establish a ControlMaster connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dest := args[0]
		if err := mux.Up(dest, upPort); err != nil {
			return fmt.Errorf("up failed: %w", err)
		}
		return nil
	},
}

func init() {
	upCmd.Flags().IntVarP(&upPort, "port", "p", 22, "SSH port")
	rootCmd.AddCommand(upCmd)
}
