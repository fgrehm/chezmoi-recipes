package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fgrehm/chezmoi-recipes/internal/paths"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

// setTestEnv overrides HOME, XDG_DATA_HOME, and XDG_CONFIG_HOME to
// prevent tests from touching host directories.
func setTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// setupTestRepo creates a repo-like directory structure with home/ and recipes/.
// Returns the repo root and recipes directory paths.
func setupTestRepo(t *testing.T) (repoRoot, recipesDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	recipesDir = filepath.Join(repoRoot, "recipes")
	homeDir := filepath.Join(repoRoot, "home")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return repoRoot, recipesDir
}

// setupTestRecipe creates a recipe directory with the given chezmoi files.
func setupTestRecipe(t *testing.T, recipesDir, name string, files map[string]string) {
	t.Helper()
	recipeDir := filepath.Join(recipesDir, name)
	if err := os.MkdirAll(recipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recipeDir, "README.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for relPath, content := range files {
		fullPath := filepath.Join(recipeDir, "chezmoi", relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// setupTestHomeFile creates a file in the repo's home/ directory.
func setupTestHomeFile(t *testing.T, repoRoot, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(repoRoot, "home", relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeState saves a state.Store to the given path.
func writeState(t *testing.T, stateFile string, store *state.Store) {
	t.Helper()
	if err := store.Save(stateFile); err != nil {
		t.Fatal(err)
	}
}

// chezmoiConfigFile returns paths.ChezmoiConfigFile(), fataling on error.
func chezmoiConfigFile(t *testing.T) string {
	t.Helper()
	path, err := paths.ChezmoiConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
