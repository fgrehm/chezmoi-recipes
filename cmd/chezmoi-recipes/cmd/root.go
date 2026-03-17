package cmd

import (
	"context"
	"path/filepath"

	"github.com/fgrehm/chezmoi-recipes/internal/paths"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "chezmoi-recipes",
	Short:         "A recipe layer for chezmoi",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().String("recipes-dir", "./recipes", "path to recipes directory")
}

// recipesDir returns the value of the --recipes-dir persistent flag.
func recipesDir() string {
	dir, _ := rootCmd.PersistentFlags().GetString("recipes-dir")
	return dir
}

// repoRoot returns the repo root derived from the recipes directory.
// The recipes directory is expected to be a direct child of the repo root.
func repoRoot() string {
	abs, err := filepath.Abs(recipesDir())
	if err != nil {
		return filepath.Dir(recipesDir())
	}
	return filepath.Dir(abs)
}

// compiledHomeDir returns the compiled-home path derived from the recipes directory.
func compiledHomeDir() string {
	return paths.CompiledHomeDir(repoRoot())
}

func Execute() error {
	return rootCmd.Execute()
}

// ExecuteContext runs the root command with the given context, allowing
// callers to propagate cancellation (e.g. from SIGINT).
func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
