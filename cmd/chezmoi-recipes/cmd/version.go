package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "chezmoi-recipes version %s\n", version)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", commit)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  built:  %s\n", date)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  go:     %s\n", runtime.Version())
		return nil
	},
}

func init() {
	rootCmd.Version = version
	rootCmd.AddCommand(versionCmd)
}
