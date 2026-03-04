package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NonexistentFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	s, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if s.Recipes == nil {
		t.Fatal("Load() returned nil Recipes map")
	}
	if len(s.Recipes) != 0 {
		t.Errorf("Load() returned %d recipes, want 0", len(s.Recipes))
	}
}

func TestSaveAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	path := filepath.Join(dir, "sub", "state.json")

	s := &Store{Recipes: make(map[string]*RecipeState)}
	s.RecordRecipe("git", []string{"dot_gitconfig", ".chezmoiscripts/run_once_install-git.sh"})

	if err := s.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	rs, ok := loaded.Recipes["git"]
	if !ok {
		t.Fatal("loaded store missing 'git' recipe")
	}
	if len(rs.Files) != 2 {
		t.Errorf("got %d files, want 2", len(rs.Files))
	}
	if rs.AppliedAt.IsZero() {
		t.Error("AppliedAt is zero")
	}
}


func TestRecordRecipe_Overwrites(t *testing.T) {
	s := &Store{Recipes: make(map[string]*RecipeState)}

	s.RecordRecipe("git", []string{"dot_gitconfig"})
	if len(s.Recipes["git"].Files) != 1 {
		t.Fatalf("initial record: got %d files, want 1", len(s.Recipes["git"].Files))
	}

	s.RecordRecipe("git", []string{"dot_gitconfig", "dot_config/git/ignore"})
	if len(s.Recipes["git"].Files) != 2 {
		t.Errorf("after overwrite: got %d files, want 2", len(s.Recipes["git"].Files))
	}
}

func TestAllFiles(t *testing.T) {
	s := &Store{Recipes: make(map[string]*RecipeState)}
	s.RecordRecipe("git", []string{"dot_gitconfig", "dot_config/git/ignore"})
	s.RecordRecipe("ripgrep", []string{".chezmoiscripts/run_once_install-ripgrep.sh"})

	got := s.AllFiles()
	want := map[string]string{
		"dot_gitconfig":                                  "git",
		"dot_config/git/ignore":                          "git",
		".chezmoiscripts/run_once_install-ripgrep.sh": "ripgrep",
	}

	if len(got) != len(want) {
		t.Fatalf("AllFiles() returned %d entries, want %d", len(got), len(want))
	}
	for path, recipe := range want {
		if got[path] != recipe {
			t.Errorf("AllFiles()[%q] = %q, want %q", path, got[path], recipe)
		}
	}
}

func TestAllFiles_EmptyStore(t *testing.T) {
	s := &Store{Recipes: make(map[string]*RecipeState)}
	got := s.AllFiles()
	if len(got) != 0 {
		t.Errorf("AllFiles() returned %d entries for empty store, want 0", len(got))
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should fail on invalid JSON")
	}
}
