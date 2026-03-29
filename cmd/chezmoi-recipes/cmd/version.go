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
		w := cmd.OutOrStdout()
		_, err := fmt.Fprintf(w,
			"chezmoi-recipes version %s\n  commit: %s\n  built:  %s\n  go:     %s\n",
			version, commit, date, runtime.Version(),
		)
		return err
	},
}

func init() {
	rootCmd.Version = version
	rootCmd.AddCommand(versionCmd)
}
