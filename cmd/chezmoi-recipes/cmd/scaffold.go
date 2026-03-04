package cmd

import (
	"context"
	"io"
	"os"

	"github.com/fgrehm/chezmoi-recipes/internal/scaffold"
	"github.com/spf13/cobra"
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <recipe-name>",
	Short: "Generate a new recipe directory with annotated example files",
	Long: `Generate a new recipe directory under the recipes directory.

The scaffolded files demonstrate chezmoi conventions (naming prefixes,
script ordering, template syntax, per-recipe .chezmoiignore, shell
integration) and serve as a starting point for new recipes.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScaffold(cmd.Context(), args[0], recipesDir(), os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(scaffoldCmd)
}

func runScaffold(_ context.Context, name, recDir string, w io.Writer) error {
	return scaffold.Run(recDir, name, w)
}
