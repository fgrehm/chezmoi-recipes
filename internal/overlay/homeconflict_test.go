package overlay

import (
	"path/filepath"
	"testing"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
)

func TestDetectHomeRecipeConflicts_NoConflict(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_bashrc"), "bashrc")

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "[user]",
	})

	if err := DetectHomeRecipeConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no conflict, got: %v", err)
	}
}

func TestDetectHomeRecipeConflicts_SingleConflict(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_gitconfig"), "home version")

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "recipe version",
	})

	err := DetectHomeRecipeConflicts(homeDir, []*recipe.Recipe{r})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	hce, ok := err.(*HomeConflictError)
	if !ok {
		t.Fatalf("expected *HomeConflictError, got %T: %v", err, err)
	}
	if hce.RelPath != "dot_gitconfig" {
		t.Errorf("RelPath = %q, want %q", hce.RelPath, "dot_gitconfig")
	}
	if hce.Recipe != "git" {
		t.Errorf("Recipe = %q, want %q", hce.Recipe, "git")
	}
}

func TestDetectHomeRecipeConflicts_NestedConflict(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_config", "nvim", "init.lua"), "home")

	r := setupRecipe(t, "neovim", map[string]string{
		filepath.Join("dot_config", "nvim", "init.lua"): "recipe",
	})

	err := DetectHomeRecipeConflicts(homeDir, []*recipe.Recipe{r})
	if err == nil {
		t.Fatal("expected conflict error for nested file, got nil")
	}

	hce, ok := err.(*HomeConflictError)
	if !ok {
		t.Fatalf("expected *HomeConflictError, got %T: %v", err, err)
	}
	if hce.RelPath != filepath.Join("dot_config", "nvim", "init.lua") {
		t.Errorf("RelPath = %q, want nested path", hce.RelPath)
	}
}

func TestDetectHomeRecipeConflicts_NoHomeDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "[user]",
	})

	if err := DetectHomeRecipeConflicts(missing, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no error for missing home dir, got: %v", err)
	}
}

func TestDetectHomeRecipeConflicts_RecipeWithoutChezmoi(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_bashrc"), "bashrc")

	// Recipe with no chezmoi/ dir
	r := &recipe.Recipe{Name: "docs-only", Dir: t.TempDir(), HasChezmoi: false}

	if err := DetectHomeRecipeConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no error for recipe without chezmoi, got: %v", err)
	}
}

func TestDetectHomeRecipeConflicts_SkipsChezmoiignore(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, ".chezmoiignore"), "some ignore")

	r := setupRecipe(t, "git", map[string]string{
		".chezmoiignore": "recipe ignore",
		"dot_gitconfig":  "[user]",
	})

	// .chezmoiignore exists in both home/ and recipe, but should not be
	// treated as a conflict because it's handled separately by merge logic.
	if err := DetectHomeRecipeConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no conflict for .chezmoiignore, got: %v", err)
	}
}
