package cmd

import (
	"context"

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
	sourceDir, err := paths.DefaultSourceDir()
	if err != nil {
		sourceDir = ""
	}
	rootCmd.PersistentFlags().String("source-dir", sourceDir, "path to chezmoi source directory managed by chezmoi-recipes")
}

// recipesDir returns the value of the --recipes-dir persistent flag.
func recipesDir() string {
	dir, _ := rootCmd.PersistentFlags().GetString("recipes-dir")
	return dir
}

// sourceDir returns the value of the --source-dir persistent flag.
func sourceDir() string {
	dir, _ := rootCmd.PersistentFlags().GetString("source-dir")
	return dir
}

func Execute() error {
	return rootCmd.Execute()
}

// ExecuteContext runs the root command with the given context, allowing
// callers to propagate cancellation (e.g. from SIGINT).
func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
