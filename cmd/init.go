package cmd

import (
	"fmt"

	"github.com/Otard95/ssh-mux/mux"
	"github.com/spf13/cobra"
)

var (
	forceInit  bool
	noEditInit bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "One-time setup — configure SSH for multiplexing",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := mux.Init(forceInit, noEditInit); err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		fmt.Println("ssh-mux initialized successfully")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&forceInit, "force", false, "Overwrite existing mux.conf")
	initCmd.Flags().BoolVar(&noEditInit, "no-edit", false, "Don't modify ~/.ssh/config (for externally managed configs)")
	rootCmd.AddCommand(initCmd)
}
