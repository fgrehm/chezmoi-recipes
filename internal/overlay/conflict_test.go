package overlay

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
)

func TestDetectConflicts_NoConflict(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_bashrc"), "bashrc")

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "[user]",
	})

	if err := DetectConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no conflict, got: %v", err)
	}
}

func TestDetectConflicts_HomeRecipeSameFile(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_gitconfig"), "home version")

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "recipe version",
	})

	err := DetectConflicts(homeDir, []*recipe.Recipe{r})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if ce.TargetPath != ".gitconfig" {
		t.Errorf("TargetPath = %q, want %q", ce.TargetPath, ".gitconfig")
	}
	if len(ce.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ce.Entries))
	}
	if ce.Entries[0].Owner != "home" {
		t.Errorf("first entry owner = %q, want %q", ce.Entries[0].Owner, "home")
	}
	if ce.Entries[1].Owner != "git" {
		t.Errorf("second entry owner = %q, want %q", ce.Entries[1].Owner, "git")
	}
}

func TestDetectConflicts_HomeRecipeAttributeMismatch(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_config", "starship.toml"), "home")

	r := setupRecipe(t, "starship", map[string]string{
		filepath.Join("private_dot_config", "starship.toml"): "recipe",
	})

	err := DetectConflicts(homeDir, []*recipe.Recipe{r})
	if err == nil {
		t.Fatal("expected conflict error for attribute mismatch, got nil")
	}

	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	// Conflict is caught at the directory level (dot_config vs private_dot_config).
	if ce.TargetPath != ".config" {
		t.Errorf("TargetPath = %q, want %q", ce.TargetPath, ".config")
	}
}

func TestDetectConflicts_RecipeVsRecipeAttributeMismatch(t *testing.T) {
	homeDir := t.TempDir() // empty

	r1 := setupRecipe(t, "alacritty", map[string]string{
		filepath.Join("dot_config", "alacritty", "alacritty.toml"): "font_size = 12",
	})
	r2 := setupRecipe(t, "kitty", map[string]string{
		filepath.Join("private_dot_config", "kitty", "kitty.conf"): "font_size 12",
	})

	err := DetectConflicts(homeDir, []*recipe.Recipe{r1, r2})
	if err == nil {
		t.Fatal("expected conflict error for dir attribute mismatch between recipes, got nil")
	}

	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	// The directory itself should be the conflict point.
	if ce.TargetPath != ".config" {
		t.Errorf("TargetPath = %q, want %q", ce.TargetPath, ".config")
	}
	if ce.Entries[0].Owner != "alacritty" {
		t.Errorf("first entry owner = %q, want %q", ce.Entries[0].Owner, "alacritty")
	}
	if ce.Entries[1].Owner != "kitty" {
		t.Errorf("second entry owner = %q, want %q", ce.Entries[1].Owner, "kitty")
	}
}

func TestDetectConflicts_RecipeVsRecipeSameFile(t *testing.T) {
	homeDir := t.TempDir()

	r1 := setupRecipe(t, "zsh", map[string]string{
		"dot_zshrc": "zsh config",
	})
	r2 := setupRecipe(t, "shell", map[string]string{
		"dot_zshrc": "shell config",
	})

	err := DetectConflicts(homeDir, []*recipe.Recipe{r1, r2})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if ce.TargetPath != ".zshrc" {
		t.Errorf("TargetPath = %q, want %q", ce.TargetPath, ".zshrc")
	}
}

func TestDetectConflicts_NestedConflict(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_config", "nvim", "init.lua"), "home")

	r := setupRecipe(t, "neovim", map[string]string{
		filepath.Join("dot_config", "nvim", "init.lua"): "recipe",
	})

	err := DetectConflicts(homeDir, []*recipe.Recipe{r})
	if err == nil {
		t.Fatal("expected conflict error for nested file, got nil")
	}

	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	// Both use dot_config/nvim/init.lua, so the file itself is the conflict.
	if ce.TargetPath != filepath.Join(".config", "nvim", "init.lua") {
		t.Errorf("TargetPath = %q, want %q", ce.TargetPath, filepath.Join(".config", "nvim", "init.lua"))
	}
}

func TestDetectConflicts_NoHomeDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "[user]",
	})

	if err := DetectConflicts(missing, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no error for missing home dir, got: %v", err)
	}
}

func TestDetectConflicts_RecipeWithoutChezmoi(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, "dot_bashrc"), "bashrc")

	r := &recipe.Recipe{Name: "docs-only", Dir: t.TempDir(), HasChezmoi: false}

	if err := DetectConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no error for recipe without chezmoi, got: %v", err)
	}
}

func TestDetectConflicts_SkipsChezmoiignore(t *testing.T) {
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(homeDir, ".chezmoiignore"), "some ignore")

	r := setupRecipe(t, "git", map[string]string{
		".chezmoiignore": "recipe ignore",
		"dot_gitconfig":  "[user]",
	})

	if err := DetectConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no conflict for .chezmoiignore, got: %v", err)
	}
}

func TestDetectConflicts_SameRecipeOwnerNoConflict(t *testing.T) {
	// Two files from the same recipe under the same directory should not conflict.
	homeDir := t.TempDir()

	r := setupRecipe(t, "neovim", map[string]string{
		filepath.Join("private_dot_config", "nvim", "init.lua"):        "lua",
		filepath.Join("private_dot_config", "nvim", "lazy-lock.json"): "{}",
	})

	if err := DetectConflicts(homeDir, []*recipe.Recipe{r}); err != nil {
		t.Fatalf("expected no conflict within same recipe, got: %v", err)
	}
}

func TestDetectConflicts_DirectoryVsDirectoryMismatch(t *testing.T) {
	// Specifically test that directory entries (not just files) trigger conflicts
	// when they disagree on attributes.
	homeDir := t.TempDir()

	r1 := setupRecipe(t, "alacritty", map[string]string{
		filepath.Join("exact_dot_config", "alacritty", "alacritty.toml"): "config",
	})
	r2 := setupRecipe(t, "kitty", map[string]string{
		filepath.Join("private_dot_config", "kitty", "kitty.conf"): "config",
	})

	err := DetectConflicts(homeDir, []*recipe.Recipe{r1, r2})
	if err == nil {
		t.Fatal("expected conflict for exact_dot_config vs private_dot_config")
	}

	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if ce.TargetPath != ".config" {
		t.Errorf("TargetPath = %q, want %q", ce.TargetPath, ".config")
	}
}

func TestConflictError_SameSourcePath(t *testing.T) {
	err := &ConflictError{
		TargetPath: ".gitconfig",
		Entries: []SourceEntry{
			{Owner: "home", SourcePath: "dot_gitconfig"},
			{Owner: "git", SourcePath: "dot_gitconfig"},
		},
	}

	msg := err.Error()
	if !strings.Contains(msg, "dot_gitconfig") {
		t.Errorf("error should mention the file path: %s", msg)
	}
	if !strings.Contains(msg, "home") {
		t.Errorf("error should mention home: %s", msg)
	}
	if !strings.Contains(msg, "git") {
		t.Errorf("error should mention the recipe: %s", msg)
	}
}

func TestConflictError_AttributeMismatch(t *testing.T) {
	err := &ConflictError{
		TargetPath: ".config",
		Entries: []SourceEntry{
			{Owner: "alacritty", SourcePath: "dot_config"},
			{Owner: "kitty", SourcePath: "private_dot_config"},
		},
	}

	msg := err.Error()
	if !strings.Contains(msg, ".config") {
		t.Errorf("error should mention the target: %s", msg)
	}
	if !strings.Contains(msg, "mismatched") {
		t.Errorf("error should mention mismatch: %s", msg)
	}
	if !strings.Contains(msg, "dot_config") {
		t.Errorf("error should mention first source: %s", msg)
	}
	if !strings.Contains(msg, "private_dot_config") {
		t.Errorf("error should mention second source: %s", msg)
	}
}
