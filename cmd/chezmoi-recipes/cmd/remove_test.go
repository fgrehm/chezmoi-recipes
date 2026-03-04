package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

func TestRunRemove_Success(t *testing.T) {
	setTestEnv(t)

	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Create files in source dir.
	for _, relPath := range []string{"dot_gitconfig", "dot_config/git/ignore"} {
		fullPath := filepath.Join(srcDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Write state with the recipe.
	store := &state.Store{
		Recipes: map[string]*state.RecipeState{
			"git": {
				AppliedAt: time.Now(),
				Files:     []string{"dot_gitconfig", "dot_config/git/ignore"},
			},
		},
	}
	writeState(t, stateFile, store)

	var buf bytes.Buffer
	err := runRemove(t.Context(), "git", srcDir, stateFile, &buf)
	if err != nil {
		t.Fatalf("runRemove() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "- dot_gitconfig") {
		t.Errorf("output missing removed file: %s", output)
	}
	if !strings.Contains(output, `removed`) {
		t.Errorf("output missing success message: %s", output)
	}
	if !strings.Contains(output, ".recipeignore") {
		t.Errorf("output should mention .recipeignore: %s", output)
	}

	// Verify files were deleted.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); !errors.Is(err, os.ErrNotExist) {
		t.Error("dot_gitconfig should be deleted")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "dot_config/git/ignore")); !errors.Is(err, os.ErrNotExist) {
		t.Error("dot_config/git/ignore should be deleted")
	}

	// Verify empty parent dirs were cleaned up.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_config", "git")); !errors.Is(err, os.ErrNotExist) {
		t.Error("empty dot_config/git/ directory should be cleaned up")
	}

	// Verify state was updated.
	loaded, err := state.Load(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Recipes["git"]; ok {
		t.Error("git recipe should be removed from state")
	}
}

func TestRunRemove_NotApplied(t *testing.T) {
	setTestEnv(t)

	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	err := runRemove(t.Context(), "nonexistent", t.TempDir(), stateFile, &buf)
	if err == nil {
		t.Fatal("runRemove() should fail for unapplied recipe")
	}
	if !strings.Contains(err.Error(), "not applied") {
		t.Errorf("error should mention 'not applied': %v", err)
	}
}

func TestRunRemove_MissingFiles(t *testing.T) {
	setTestEnv(t)

	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// State references files that don't exist on disk.
	store := &state.Store{
		Recipes: map[string]*state.RecipeState{
			"git": {
				AppliedAt: time.Now(),
				Files:     []string{"dot_gitconfig"},
			},
		},
	}
	writeState(t, stateFile, store)

	var buf bytes.Buffer
	err := runRemove(t.Context(), "git", srcDir, stateFile, &buf)
	if err != nil {
		t.Fatalf("runRemove() should succeed even with missing files: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Already missing") {
		t.Errorf("output should mention missing files: %s", output)
	}
	if !strings.Contains(output, "? dot_gitconfig") {
		t.Errorf("output should list the missing file: %s", output)
	}

	// State should still be cleaned up.
	loaded, err := state.Load(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Recipes["git"]; ok {
		t.Error("git recipe should be removed from state")
	}
}

func TestRunRemove_PreservesOtherRecipes(t *testing.T) {
	setTestEnv(t)

	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Create git file.
	if err := os.WriteFile(filepath.Join(srcDir, "dot_gitconfig"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := &state.Store{
		Recipes: map[string]*state.RecipeState{
			"git": {
				AppliedAt: time.Now(),
				Files:     []string{"dot_gitconfig"},
			},
			"ripgrep": {
				AppliedAt: time.Now(),
				Files:     []string{".chezmoiscripts/run_once_install-ripgrep.sh"},
			},
		},
	}
	writeState(t, stateFile, store)

	var buf bytes.Buffer
	err := runRemove(t.Context(), "git", srcDir, stateFile, &buf)
	if err != nil {
		t.Fatalf("runRemove() error = %v", err)
	}

	loaded, err := state.Load(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Recipes["git"]; ok {
		t.Error("git should be removed")
	}
	if _, ok := loaded.Recipes["ripgrep"]; !ok {
		t.Error("ripgrep should be preserved")
	}
}
