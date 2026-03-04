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
	Short: "Initialize chezmoi-recipes (set up config template, shared utilities, and recipes directory)",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return runInitCmd(sourceDir(), recipesDir(), force, cmd.OutOrStdout())
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite .chezmoi.toml.tmpl if it already exists")
	rootCmd.AddCommand(initCmd)
}

func runInitCmd(srcDir, recDir string, force bool, w io.Writer) error {
	result, err := setup.RunInit(srcDir, recDir, force)
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	absRecDir, err := filepath.Abs(recDir)
	if err != nil {
		absRecDir = recDir
	}
	projectDir := filepath.Dir(absRecDir)

	makefileCreated, err := setup.EnsureMakefile(projectDir, absRecDir)
	if err != nil {
		return fmt.Errorf("configuring Makefile: %w", err)
	}

	fmt.Fprintln(w, "\nchezmoi-recipes initialized.")
	fmt.Fprintf(w, "  Source dir:    %s\n", srcDir)
	fmt.Fprintf(w, "  Recipes dir:   %s\n", recDir)
	if result.ConfigSkipped {
		fmt.Fprintln(w, "  Config:        .chezmoi.toml.tmpl already exists, skipped (use --force to overwrite)")
	}
	if makefileCreated {
		fmt.Fprintf(w, "  Makefile:      %s (shell-fmt, shell-fmt-check, shell-lint, check)\n", filepath.Join(projectDir, "Makefile"))
	}
	fmt.Fprintln(w, "\nNext step: run 'chezmoi init' to configure user data (name, email).")
	fmt.Fprintln(w, "chezmoi will prompt for values defined in .chezmoi.toml.tmpl.")

	return nil
}
