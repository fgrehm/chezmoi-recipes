package cmd

import (
	"bufio"
	"context"
	"errors"
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
		return runInitCmd(cmd.Context(), recipesDir(), force, cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite .chezmoi.toml.tmpl if it already exists")
	rootCmd.AddCommand(initCmd)
}

func runInitCmd(ctx context.Context, recDir string, force bool, r io.Reader, w io.Writer) error {
	absRecDir, err := filepath.Abs(recDir)
	if err != nil {
		absRecDir = recDir
	}
	repoRoot := filepath.Dir(absRecDir)

	repoURL, detectErr := detectRepoURL(ctx, repoRoot)
	if repoURL == "" {
		if detectErr != nil {
			fmt.Fprintf(w, "Could not detect repository URL: %s\n", detectErr)
		} else {
			fmt.Fprint(w, "No git remote 'origin' configured.\n")
		}
		fmt.Fprint(w, "Repository URL for install.sh (leave empty to skip): ")
		scanner := bufio.NewScanner(r)
		if scanner.Scan() {
			repoURL = strings.TrimSpace(scanner.Text())
		} else if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading repository URL from stdin: %w", err)
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

// detectRepoURL returns the origin remote URL, or ("", err) if unavailable.
// err carries the underlying git error message for display to the user.
func detectRepoURL(ctx context.Context, repoRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("%s", stderr)
			}
			return "", fmt.Errorf("git remote get-url origin: %w", err)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
