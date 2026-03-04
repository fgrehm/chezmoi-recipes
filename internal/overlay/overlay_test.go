package overlay

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

// setupRecipe creates a recipe directory with the given files under chezmoi/.
// Files is a map of relative path to content.
func setupRecipe(t *testing.T, name string, files map[string]string) *recipe.Recipe {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)

	// Create README.md (required for recipe discovery)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# "+name), 0o644); err != nil {
		t.Fatal(err)
	}

	chezmoiDir := filepath.Join(dir, "chezmoi")
	for relPath, content := range files {
		fullPath := filepath.Join(chezmoiDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return &recipe.Recipe{
		Name:       name,
		Dir:        dir,
		HasChezmoi: true,
	}
}

func TestPlan_CleanOverlay(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig":      "[init]\n    defaultBranch = main\n",
		"dot_config/git/ignore": ".DS_Store\n",
	})
	sourceDir := t.TempDir()
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	result, err := Plan(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(result.Added) != 2 {
		t.Errorf("got %d added files, want 2", len(result.Added))
	}
	if len(result.Updated) != 0 {
		t.Errorf("got %d updated files, want 0", len(result.Updated))
	}
}

func TestPlan_ReapplySameRecipe(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})
	sourceDir := t.TempDir()

	// Create existing file in source dir
	if err := os.WriteFile(filepath.Join(sourceDir, "dot_gitconfig"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	// State says git owns this file
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}
	store.RecordRecipe("git", []string{"dot_gitconfig"})

	result, err := Plan(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(result.Added) != 0 {
		t.Errorf("got %d added files, want 0", len(result.Added))
	}
	if len(result.Updated) != 1 {
		t.Errorf("got %d updated files, want 1", len(result.Updated))
	}
}

func TestPlan_ConflictWithOtherRecipe(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "vim", map[string]string{
		"dot_gitconfig": "conflict",
	})
	sourceDir := t.TempDir()

	// File exists and is owned by another recipe
	if err := os.WriteFile(filepath.Join(sourceDir, "dot_gitconfig"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}
	store.RecordRecipe("git", []string{"dot_gitconfig"})

	_, err := Plan(context.Background(), r, sourceDir, store)
	if err == nil {
		t.Fatal("Plan() should fail with conflict error")
	}

	conflictErr, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if conflictErr.ExistingOwner != "git" {
		t.Errorf("ExistingOwner = %q, want %q", conflictErr.ExistingOwner, "git")
	}
	if conflictErr.Recipe != "vim" {
		t.Errorf("Recipe = %q, want %q", conflictErr.Recipe, "vim")
	}
}

func TestPlan_ConflictWithUntrackedFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "new",
	})
	sourceDir := t.TempDir()

	// File exists but not in state (untracked)
	if err := os.WriteFile(filepath.Join(sourceDir, "dot_gitconfig"), []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	_, err := Plan(context.Background(), r, sourceDir, store)
	if err == nil {
		t.Fatal("Plan() should fail with conflict error for untracked file")
	}

	conflictErr, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if conflictErr.ExistingOwner != "" {
		t.Errorf("ExistingOwner = %q, want empty string", conflictErr.ExistingOwner)
	}
}

func TestPlan_NochezmoiDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	dir := t.TempDir()
	r := &recipe.Recipe{Name: "broken", Dir: dir, HasChezmoi: false}
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	_, err := Plan(context.Background(), r, t.TempDir(), store)
	if err == nil {
		t.Fatal("Plan() should fail when chezmoi dir is missing")
	}
}

func TestExecute_CopiesFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig":      "[init]\n    defaultBranch = main\n",
		"dot_config/git/ignore": ".DS_Store\n",
	})
	sourceDir := t.TempDir()
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	result, err := Execute(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Added) != 2 {
		t.Errorf("got %d added files, want 2", len(result.Added))
	}

	// Verify files exist on disk
	for _, relPath := range result.Added {
		destPath := filepath.Join(sourceDir, relPath)
		if _, err := os.Stat(destPath); err != nil {
			t.Errorf("file %q not found in source dir: %v", relPath, err)
		}
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(sourceDir, "dot_gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[init]\n    defaultBranch = main\n" {
		t.Errorf("content mismatch: got %q", string(data))
	}
}

func TestExecute_NestedDirectories(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		".chezmoiscripts/run_once_install-git.sh": "#!/bin/bash\nsudo apt install -y git\n",
		"dot_config/git/ignore":                   ".DS_Store\n",
	})
	sourceDir := t.TempDir()
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	result, err := Execute(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Added) != 2 {
		t.Errorf("got %d added files, want 2", len(result.Added))
	}

	// Verify nested dirs were created
	if _, err := os.Stat(filepath.Join(sourceDir, ".chezmoiscripts", "run_once_install-git.sh")); err != nil {
		t.Errorf("nested script not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "dot_config", "git", "ignore")); err != nil {
		t.Errorf("nested config not found: %v", err)
	}
}

func TestExecute_PreservesPermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	dir := filepath.Join(t.TempDir(), "perms-recipe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# perms"), 0o644); err != nil {
		t.Fatal(err)
	}

	chezmoiDir := filepath.Join(dir, "chezmoi")
	scriptDir := filepath.Join(chezmoiDir, ".chezmoiscripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write executable script
	if err := os.WriteFile(filepath.Join(scriptDir, "run_once_install.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := &recipe.Recipe{Name: "perms", Dir: dir, HasChezmoi: true}
	sourceDir := t.TempDir()
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	_, err := Execute(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	destPath := filepath.Join(sourceDir, ".chezmoiscripts", "run_once_install.sh")
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}

	mode := info.Mode()
	if mode&0o111 == 0 {
		t.Errorf("expected executable permissions, got %v", mode)
	}
}

func TestExecute_ReapplySameRecipe(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "new content",
	})
	sourceDir := t.TempDir()

	// Create existing file
	if err := os.WriteFile(filepath.Join(sourceDir, "dot_gitconfig"), []byte("old content"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}
	store.RecordRecipe("git", []string{"dot_gitconfig"})

	result, err := Execute(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Errorf("got %d updated, want 1", len(result.Updated))
	}

	// Verify content was updated
	data, err := os.ReadFile(filepath.Join(sourceDir, "dot_gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("file not updated, got %q", string(data))
	}
}

func TestPlan_SkipsChezmoiignore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "alacritty", map[string]string{
		".chezmoiignore":                              "{{ if .isContainer }}\nprivate_dot_config/alacritty/\n{{ end }}\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})
	sourceDir := t.TempDir()
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	result, err := Plan(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// .chezmoiignore should NOT appear in results.
	for _, f := range result.Added {
		if f == ".chezmoiignore" {
			t.Error(".chezmoiignore should be excluded from Plan results")
		}
	}
	// The other file should be present.
	if len(result.Added) != 1 {
		t.Errorf("got %d added files, want 1: %v", len(result.Added), result.Added)
	}
}

func TestExecute_DoesNotCopyChezmoiignore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "alacritty", map[string]string{
		".chezmoiignore":                              "private_dot_config/alacritty/\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})
	sourceDir := t.TempDir()
	store := &state.Store{Recipes: make(map[string]*state.RecipeState)}

	_, err := Execute(context.Background(), r, sourceDir, store)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// .chezmoiignore should NOT be copied to source dir.
	if _, err := os.Stat(filepath.Join(sourceDir, ".chezmoiignore")); err == nil {
		t.Error(".chezmoiignore should not be copied to source dir by Execute")
	}

	// The config file should be copied.
	if _, err := os.Stat(filepath.Join(sourceDir, "private_dot_config/alacritty/alacritty.toml")); err != nil {
		t.Error("alacritty.toml should be copied")
	}
}

func TestReadIgnoreEntries_ReturnsContentWhenPresent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	want := "{{ if .isContainer }}\nprivate_dot_config/alacritty/\n{{ end }}\n"
	r := setupRecipe(t, "alacritty", map[string]string{
		".chezmoiignore":                              want,
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	got, err := ReadIgnoreEntries(r)
	if err != nil {
		t.Fatalf("ReadIgnoreEntries() error = %v", err)
	}
	if got != want {
		t.Errorf("ReadIgnoreEntries() = %q, want %q", got, want)
	}
}

func TestReadIgnoreEntries_ReturnsEmptyWhenMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	r := setupRecipe(t, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})

	got, err := ReadIgnoreEntries(r)
	if err != nil {
		t.Fatalf("ReadIgnoreEntries() error = %v", err)
	}
	if got != "" {
		t.Errorf("ReadIgnoreEntries() = %q, want empty string", got)
	}
}

func TestConflictError_UntrackedFile(t *testing.T) {
	err := &ConflictError{
		RelPath: "dot_gitconfig",
		Recipe:  "vim",
	}

	msg := err.Error()
	if !strings.Contains(msg, "dot_gitconfig") {
		t.Errorf("error should mention the file path: %s", msg)
	}
	if !strings.Contains(msg, "untracked") {
		t.Errorf("error should mention 'untracked': %s", msg)
	}
	if !strings.Contains(msg, "hint:") {
		t.Errorf("error should include a hint: %s", msg)
	}
}

func TestConflictError_OwnedByOtherRecipe(t *testing.T) {
	err := &ConflictError{
		RelPath:       "dot_gitconfig",
		ExistingOwner: "git",
		Recipe:        "vim",
	}

	msg := err.Error()
	if !strings.Contains(msg, "dot_gitconfig") {
		t.Errorf("error should mention the file path: %s", msg)
	}
	if !strings.Contains(msg, "git") {
		t.Errorf("error should mention the owning recipe: %s", msg)
	}
	if !strings.Contains(msg, "vim") {
		t.Errorf("error should mention the conflicting recipe: %s", msg)
	}
}
