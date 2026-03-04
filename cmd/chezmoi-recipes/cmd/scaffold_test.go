package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunScaffold_CreatesRecipe(t *testing.T) {
	setTestEnv(t)
	recipesDir := t.TempDir()
	var buf bytes.Buffer

	if err := runScaffold(t.Context(), "mytool", recipesDir, &buf); err != nil {
		t.Fatalf("runScaffold() error = %v", err)
	}

	// Verify recipe directory was created with README and chezmoi dir.
	recipeDir := filepath.Join(recipesDir, "mytool")
	if _, err := os.Stat(filepath.Join(recipeDir, "README.md")); err != nil {
		t.Error("README.md should exist")
	}
	if _, err := os.Stat(filepath.Join(recipeDir, "chezmoi/.chezmoiscripts/run_once_install-mytool.sh.tmpl")); err != nil {
		t.Error("install script should exist")
	}
}

func TestRunScaffold_ExistingRecipeErrors(t *testing.T) {
	setTestEnv(t)
	recipesDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(recipesDir, "mytool"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runScaffold(t.Context(), "mytool", recipesDir, &buf)
	if err == nil {
		t.Fatal("expected error for existing recipe")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

func TestRunScaffold_RespectsRecipesDir(t *testing.T) {
	setTestEnv(t)
	customDir := filepath.Join(t.TempDir(), "custom-recipes")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runScaffold(t.Context(), "neovim", customDir, &buf); err != nil {
		t.Fatalf("runScaffold() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(customDir, "neovim", "README.md")); err != nil {
		t.Error("recipe should be created under custom recipes dir")
	}
}
