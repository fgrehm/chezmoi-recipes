package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fgrehm/chezmoi-recipes/internal/paths"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <recipe>",
	Short: "Remove a recipe's files from the source directory",
	Long: `Remove deletes the recipe's files from the chezmoi source directory and
removes its entry from state. It does not undo any system changes made
by the recipe's chezmoi scripts (installed packages, etc.).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stateFile, err := paths.DefaultStateFile()
		if err != nil {
			return fmt.Errorf("resolving state file: %w", err)
		}
		return runRemove(cmd.Context(), args[0], sourceDir(), stateFile, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(_ context.Context, name, srcDir, stateFile string, w io.Writer) error {
	store, err := state.Load(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	rs, ok := store.Recipes[name]
	if !ok {
		return fmt.Errorf("recipe %q is not applied", name)
	}

	var removed []string
	var missing []string

	for _, relPath := range rs.Files {
		fullPath := filepath.Join(srcDir, relPath)
		if _, err := os.Stat(fullPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, relPath)
				continue
			}
			return fmt.Errorf("checking %q: %w", relPath, err)
		}
		if err := removeFileAndCleanDirs(fullPath, srcDir); err != nil {
			return fmt.Errorf("removing %q: %w", relPath, err)
		}
		removed = append(removed, relPath)
	}

	if len(removed) > 0 {
		fmt.Fprintln(w, "Removed files:")
		for _, f := range removed {
			fmt.Fprintf(w, "  - %s\n", f)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintln(w, "\nAlready missing:")
		for _, f := range missing {
			fmt.Fprintf(w, "  ? %s\n", f)
		}
	}

	delete(store.Recipes, name)
	if err := store.Save(stateFile); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Fprintf(w, "\nRecipe %q removed.\n", name)
	fmt.Fprintf(w, "\nNote: if the recipe directory still exists, the next chezmoi command\n")
	fmt.Fprintf(w, "will re-overlay it via the read-source-state.pre hook. To prevent this,\n")
	fmt.Fprintf(w, "delete the recipe directory or add %q to .recipeignore.\n", name)
	return nil
}
