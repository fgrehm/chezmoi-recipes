package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/fgrehm/chezmoi-recipes/internal/paths"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show applied recipes and their files",
	RunE: func(cmd *cobra.Command, args []string) error {
		stateFile, err := paths.DefaultStateFile()
		if err != nil {
			return fmt.Errorf("resolving state file: %w", err)
		}
		return runStatus(cmd.Context(), stateFile, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(_ context.Context, stateFile string, w io.Writer) error {
	store, err := state.Load(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if len(store.Recipes) == 0 {
		fmt.Fprintln(w, "No recipes applied.")
		return nil
	}

	names := make([]string, 0, len(store.Recipes))
	for name := range store.Recipes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		rs := store.Recipes[name]
		fmt.Fprintf(w, "%s (applied %s)\n", name, rs.AppliedAt.Format("2006-01-02T15:04:05Z07:00"))
		for _, f := range rs.Files {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}

	return nil
}
