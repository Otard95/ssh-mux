package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ssh-mux",
	Short: "SSH ControlMaster lifecycle manager",
	Long: `ssh-mux manages SSH ControlMaster connections, making FIDO2/YubiKey
SSH keys practical for tools that run many SSH operations.
Authenticate once, reuse everywhere.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
