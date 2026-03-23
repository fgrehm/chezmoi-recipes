package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fgrehm/chezmoi-recipes/internal/setup"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize chezmoi-recipes (set up .chezmoiroot, config template, and recipes directory)",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return runInitCmd(recipesDir(), force, cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite .chezmoi.toml.tmpl if it already exists")
	rootCmd.AddCommand(initCmd)
}

func runInitCmd(recDir string, force bool, r io.Reader, w io.Writer) error {
	absRecDir, err := filepath.Abs(recDir)
	if err != nil {
		absRecDir = recDir
	}
	repoRoot := filepath.Dir(absRecDir)

	repoURL := detectRepoURL(repoRoot)
	if repoURL == "" {
		fmt.Fprint(w, "Git remote 'origin' not found.\n")
		fmt.Fprint(w, "Repository URL for install.sh (leave empty to skip): ")
		scanner := bufio.NewScanner(r)
		if scanner.Scan() {
			repoURL = strings.TrimSpace(scanner.Text())
		}
	}

	result, err := setup.RunInit(repoRoot, absRecDir, repoURL, force)
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
	if repoURL != "" {
		fmt.Fprintf(w, "  Install:       %s\n", filepath.Join(repoRoot, "install.sh"))
	} else {
		fmt.Fprintln(w, "  Install:       skipped (no repo URL provided)")
	}
	fmt.Fprintf(w, "\nNext step: run 'chezmoi init --source \"%s\"' to configure user data (name, email).\n", repoRoot)
	fmt.Fprintln(w, "chezmoi will prompt for values defined in .chezmoi.toml.tmpl.")

	return nil
}

// detectRepoURL returns the origin remote URL, or "" if unavailable.
func detectRepoURL(repoRoot string) string {
	cmd := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
