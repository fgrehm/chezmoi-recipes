package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull latest changes from the recipes repository",
	Long: `Pull latest changes from the git repository containing the recipes
directory. Designed to be called via chezmoi's apply.pre hook so that
recipe updates are picked up on every chezmoi apply.

--on-error controls behavior when git pull fails (e.g. when offline):
  fail    exit non-zero (aborts chezmoi apply)
  warn    print a warning to stderr and continue
  ignore  continue silently`,
	RunE: func(cmd *cobra.Command, args []string) error {
		onError, _ := cmd.Flags().GetString("on-error")
		switch onError {
		case "fail", "warn", "ignore":
		default:
			return fmt.Errorf("--on-error must be fail, warn, or ignore; got %q", onError)
		}
		return runPull(cmd.Context(), recipesDir(), onError, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	pullCmd.Flags().String("on-error", "fail", "behavior on pull failure: fail, warn, or ignore")
	rootCmd.AddCommand(pullCmd)
}

func runPull(ctx context.Context, recipesDir, onError string, w, errW io.Writer) error {
	repoRoot, err := findGitRoot(recipesDir)
	if err != nil {
		return handlePullError(fmt.Errorf("finding git root for %s: %w", recipesDir, err), onError, errW)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "pull")
	cmd.Stdout = w
	cmd.Stderr = errW
	if err := cmd.Run(); err != nil {
		return handlePullError(fmt.Errorf("git pull in %s: %w", repoRoot, err), onError, errW)
	}
	return nil
}

// findGitRoot walks up from dir until it finds a directory containing .git.
func findGitRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	current := abs
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no git repository found at or above %s", abs)
		}
		current = parent
	}
}

// handlePullError handles a pull error according to --on-error.
func handlePullError(err error, onError string, errW io.Writer) error {
	switch onError {
	case "warn":
		fmt.Fprintf(errW, "warning: chezmoi-recipes pull: %v\n", err)
		return nil
	case "ignore":
		return nil
	default: // "fail"
		return err
	}
}
