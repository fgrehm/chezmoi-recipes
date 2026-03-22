package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/fgrehm/chezmoi-recipes/internal/setup"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize chezmoi-recipes (set up .chezmoiroot, config template, and recipes directory)",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return runInitCmd(recipesDir(), force, cmd.OutOrStdout())
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite .chezmoi.toml.tmpl if it already exists")
	rootCmd.AddCommand(initCmd)
}

func runInitCmd(recDir string, force bool, w io.Writer) error {
	absRecDir, err := filepath.Abs(recDir)
	if err != nil {
		absRecDir = recDir
	}
	repoRoot := filepath.Dir(absRecDir)

	result, err := setup.RunInit(repoRoot, absRecDir, force)
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	makefileCreated, err := setup.EnsureMakefile(repoRoot, absRecDir)
	if err != nil {
		return fmt.Errorf("configuring Makefile: %w", err)
	}

	fmt.Fprintln(w, "\nchezmoi-recipes initialized.")
	fmt.Fprintf(w, "  Repo root:     %s\n", repoRoot)
	fmt.Fprintf(w, "  Home dir:      %s\n", filepath.Join(repoRoot, "home"))
	fmt.Fprintf(w, "  Recipes dir:   %s\n", absRecDir)
	if result.ConfigSkipped {
		fmt.Fprintln(w, "  Config:        .chezmoi.toml.tmpl already exists, skipped (use --force to overwrite)")
	}
	if makefileCreated {
		fmt.Fprintf(w, "  Makefile:      %s (shell-fmt, shell-fmt-check, shell-lint, check)\n", filepath.Join(repoRoot, "Makefile"))
	}
	fmt.Fprintf(w, "\nNext step: run 'chezmoi init --source \"%s\"' to configure user data (name, email).\n", repoRoot)
	fmt.Fprintln(w, "chezmoi will prompt for values defined in .chezmoi.toml.tmpl.")

	return nil
}
