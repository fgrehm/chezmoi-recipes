package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeREADME(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkChezmoiDir(t *testing.T, recipeDir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(recipeDir, "chezmoi"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_WithReadmeAndChezmoi(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myrecipe")
	writeREADME(t, dir, "# My Recipe\n")
	mkChezmoiDir(t, dir)

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected recipe, got nil")
	}
	if r.Name != "myrecipe" {
		t.Errorf("Name = %q, want %q", r.Name, "myrecipe")
	}
	if !r.HasChezmoi {
		t.Error("HasChezmoi = false, want true")
	}
	if r.Dir == "" {
		t.Error("Dir should be set")
	}
}

func TestLoad_WithReadmeNoChezmoi(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "minimal")
	writeREADME(t, dir, "# Minimal\n")

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected recipe, got nil")
	}
	if r.Name != "minimal" {
		t.Errorf("Name = %q, want %q", r.Name, "minimal")
	}
	if r.HasChezmoi {
		t.Error("HasChezmoi = true, want false")
	}
}

func TestLoad_NoReadme(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "notarecipe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != nil {
		t.Fatalf("expected nil for directory without README.md, got %+v", r)
	}
}

func TestLoad_NameFromDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "neovim")
	writeREADME(t, dir, "# Neovim\n")

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "neovim" {
		t.Errorf("Name = %q, want %q", r.Name, "neovim")
	}
}

func TestLoadAll_MultipleRecipes(t *testing.T) {
	base := t.TempDir()

	writeREADME(t, filepath.Join(base, "bravo"), "# Bravo\n")
	writeREADME(t, filepath.Join(base, "alpha"), "# Alpha\n")
	mkChezmoiDir(t, filepath.Join(base, "alpha"))

	recipes, err := LoadAll(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recipes) != 2 {
		t.Fatalf("got %d recipes, want 2", len(recipes))
	}
	if recipes[0].Name != "alpha" {
		t.Errorf("first recipe = %q, want %q", recipes[0].Name, "alpha")
	}
	if recipes[1].Name != "bravo" {
		t.Errorf("second recipe = %q, want %q", recipes[1].Name, "bravo")
	}
}

func TestLoadAll_EmptyDirectory(t *testing.T) {
	base := t.TempDir()

	recipes, err := LoadAll(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("got %d recipes, want 0", len(recipes))
	}
}

func TestLoadAll_SkipsFilesAndNonRecipeDirs(t *testing.T) {
	base := t.TempDir()

	// Regular file at root - should be ignored
	if err := os.WriteFile(filepath.Join(base, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Directory without README.md - should be ignored
	if err := os.MkdirAll(filepath.Join(base, "notarecipe"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Actual recipe
	writeREADME(t, filepath.Join(base, "only"), "# Only\n")

	recipes, err := LoadAll(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recipes) != 1 {
		t.Fatalf("got %d recipes, want 1", len(recipes))
	}
	if recipes[0].Name != "only" {
		t.Errorf("recipe name = %q, want %q", recipes[0].Name, "only")
	}
}

func TestLoad_EmptyReadme(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "badrecipe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a 0-byte README.md.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() should fail for empty README.md")
	}
	if !strings.Contains(err.Error(), "empty README.md") {
		t.Errorf("error should mention 'empty README.md': %v", err)
	}
}

func TestLoad_EmptyChezmoiDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "emptychez")
	writeREADME(t, dir, "# Empty chezmoi\n")
	mkChezmoiDir(t, dir)

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if r == nil {
		t.Fatal("expected recipe, got nil")
	}
	if !r.HasChezmoi {
		t.Error("HasChezmoi = false, want true")
	}
	if !r.EmptyChezmoi {
		t.Error("EmptyChezmoi = false, want true")
	}
}

func TestLoad_NonEmptyChezmoiDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "withfiles")
	writeREADME(t, dir, "# With files\n")
	mkChezmoiDir(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "chezmoi", "dot_gitconfig"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if r.EmptyChezmoi {
		t.Error("EmptyChezmoi = true, want false")
	}
}
