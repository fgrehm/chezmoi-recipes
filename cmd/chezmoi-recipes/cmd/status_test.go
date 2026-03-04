package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

func TestRunStatus_NoRecipes(t *testing.T) {
	setTestEnv(t)

	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	if err := runStatus(t.Context(), stateFile, &buf); err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No recipes applied") {
		t.Errorf("expected 'No recipes applied' message, got: %s", output)
	}
}

func TestRunStatus_WithRecipes(t *testing.T) {
	setTestEnv(t)

	stateFile := filepath.Join(t.TempDir(), "state.json")

	appliedAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	store := &state.Store{
		Recipes: map[string]*state.RecipeState{
			"git": {
				AppliedAt: appliedAt,
				Files:     []string{"dot_gitconfig", "dot_config/git/ignore"},
			},
			"alacritty": {
				AppliedAt: appliedAt,
				Files:     []string{"dot_config/alacritty/alacritty.toml"},
			},
		},
	}
	writeState(t, stateFile, store)

	var buf bytes.Buffer
	if err := runStatus(t.Context(), stateFile, &buf); err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}

	output := buf.String()

	// Verify alphabetical ordering (alacritty before git).
	alacrittyIdx := strings.Index(output, "alacritty")
	gitIdx := strings.Index(output, "git")
	if alacrittyIdx == -1 || gitIdx == -1 {
		t.Fatalf("missing recipe names in output: %s", output)
	}
	if alacrittyIdx > gitIdx {
		t.Errorf("recipes not sorted alphabetically: %s", output)
	}

	// Verify timestamp format.
	if !strings.Contains(output, "2025-01-15T10:30:00Z") {
		t.Errorf("missing or wrong timestamp in output: %s", output)
	}

	// Verify files are listed.
	if !strings.Contains(output, "  dot_gitconfig") {
		t.Errorf("missing file listing: %s", output)
	}
	if !strings.Contains(output, "  dot_config/git/ignore") {
		t.Errorf("missing file listing: %s", output)
	}
	if !strings.Contains(output, "  dot_config/alacritty/alacritty.toml") {
		t.Errorf("missing file listing: %s", output)
	}
}
