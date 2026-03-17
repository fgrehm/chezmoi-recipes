package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgrehm/chezmoi-recipes/internal/overlay"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

// compiledHome returns the compiled-home path for a repo root.
func compiledHome(repoRoot string) string {
	return filepath.Join(repoRoot, "compiled-home")
}

func TestRunOverlay_NoArgsLoadsAllRecipes(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig not overlaid")
	}
	if _, err := os.Stat(filepath.Join(ch, "dot_vimrc")); err != nil {
		t.Error("dot_vimrc not overlaid")
	}
}

func TestRunOverlay_NoArgsFiltersRecipesByRecipeignore(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})
	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		"dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	if err := os.WriteFile(ignoreFile, []byte("alacritty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should be overlaid")
	}
	if _, err := os.Stat(filepath.Join(ch, "dot_config/alacritty/alacritty.toml")); err == nil {
		t.Error("alacritty should NOT be overlaid (filtered by .recipeignore)")
	}
}

func TestRunOverlay_NamedArgsOverlayOnlySpecifiedRecipes(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "git config\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"git"}, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should be overlaid")
	}
	if _, err := os.Stat(filepath.Join(ch, "dot_vimrc")); err == nil {
		t.Error("dot_vimrc should NOT be overlaid (not specified)")
	}
}

func TestRunOverlay_NamedArgsOverrideRecipeignore(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		"dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	if err := os.WriteFile(ignoreFile, []byte("alacritty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"alacritty"}, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_config/alacritty/alacritty.toml")); err != nil {
		t.Error("alacritty should be overlaid even though it's in .recipeignore (explicit name overrides)")
	}
}

func TestRunOverlay_DoesNotInvokeChezmoi(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "chezmoi apply") || strings.Contains(output, "Running chezmoi") {
		t.Errorf("overlay should not reference chezmoi invocation, got: %s", output)
	}
}

func TestRunOverlay_ConflictOnSecondRecipe_StopsNoStateSaved(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "aaa", map[string]string{
		"dot_shared": "from aaa\n",
	})
	setupTestRecipe(t, recipesDir, "bbb", map[string]string{
		"dot_shared": "from bbb\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if _, ok := err.(*overlay.ConflictError); !ok {
		t.Errorf("expected *overlay.ConflictError, got %T: %v", err, err)
	}

	if _, err := os.Stat(stateFile); err == nil {
		t.Error("state file should not be created when overlay fails")
	}
}

func TestRunOverlay_DryRunMultipleRecipes(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err == nil {
		t.Error("dot_gitconfig should not exist in dry-run mode")
	}
	if _, err := os.Stat(filepath.Join(ch, "dot_vimrc")); err == nil {
		t.Error("dot_vimrc should not exist in dry-run mode")
	}

	if _, err := os.Stat(stateFile); err == nil {
		t.Error("state file should not be created during dry-run")
	}

	output := buf.String()
	if !strings.Contains(output, "dry-run") {
		t.Errorf("output should mention dry-run: %s", output)
	}
}

func TestRunOverlay_RecipesOverlaidInAlphabeticalOrder(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "zzz", map[string]string{
		"dot_zzz": "zzz\n",
	})
	setupTestRecipe(t, recipesDir, "aaa", map[string]string{
		"dot_aaa": "aaa\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"zzz", "aaa"}, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	aaaIdx := strings.Index(output, "aaa")
	zzzIdx := strings.Index(output, "zzz")
	if aaaIdx < 0 || zzzIdx < 0 {
		t.Fatalf("expected both recipes in output: %s", output)
	}
	if aaaIdx > zzzIdx {
		t.Errorf("aaa should appear before zzz in output (alphabetical order): %s", output)
	}
}

func TestRunOverlay_NoRecipesToOverlay_ReturnsNil(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("expected nil error for empty recipes dir, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No recipes found") {
		t.Errorf("expected 'No recipes found' message, got: %s", output)
	}
}

func TestRunOverlay_AllRecipesFiltered_ReturnsNil(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	if err := os.WriteFile(ignoreFile, []byte("git\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("expected nil error when all recipes filtered, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No recipes to overlay") {
		t.Errorf("expected 'No recipes to overlay' message, got: %s", output)
	}
}

func TestRunOverlay_StateRecordsAllRecipes(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	stateStr := string(data)
	if !strings.Contains(stateStr, "git") {
		t.Error("state should contain git recipe")
	}
	if !strings.Contains(stateStr, "vim") {
		t.Error("state should contain vim recipe")
	}
}

func TestRunOverlay_NamedRecipeNotFound(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"nonexistent"}, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err == nil {
		t.Fatal("expected error for nonexistent recipe")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestRunOverlay_QuietSuppressesOutput(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, true, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got: %s", buf.String())
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should be overlaid even in quiet mode")
	}
}

func TestRunOverlay_IdempotentOverlay(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})

	var buf1 bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf1); err != nil {
		t.Fatalf("first runOverlay() error = %v", err)
	}

	var buf2 bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf2); err != nil {
		t.Fatalf("second runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	data, err := os.ReadFile(filepath.Join(ch, "dot_gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "defaultBranch") {
		t.Error("file content incorrect after idempotent overlay")
	}
}

// --- Section: Output formatting ---

func TestRunOverlay_SingleRecipeOutput_NoPrefix(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":         "content\n",
		"dot_config/git/ignore": "*.swp\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"git"}, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "[1/1]") {
		t.Errorf("single recipe should not have [N/M] prefix: %s", output)
	}
	if !strings.Contains(output, "git") {
		t.Errorf("output should contain recipe name: %s", output)
	}
	if !strings.Contains(output, "dot_gitconfig") {
		t.Errorf("output should contain file name: %s", output)
	}
}

func TestRunOverlay_MultiRecipeOutput_WithPrefix(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[1/2]") {
		t.Errorf("multi-recipe output should have [1/2] prefix: %s", output)
	}
	if !strings.Contains(output, "[2/2]") {
		t.Errorf("multi-recipe output should have [2/2] prefix: %s", output)
	}
}

func TestRunOverlay_SummaryLineHasCorrectCounts(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":         "content\n",
		"dot_config/git/ignore": "*.swp\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 recipes") {
		t.Errorf("summary should mention recipe count: %s", output)
	}
	if !strings.Contains(output, "3 added") {
		t.Errorf("summary should mention added count: %s", output)
	}
}

func TestRunOverlay_NoChangesPerRecipe(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	recipeDir := filepath.Join(recipesDir, "empty")
	if err := os.MkdirAll(filepath.Join(recipeDir, "chezmoi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recipeDir, "README.md"), []byte("# empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("empty recipe should show 'no changes': %s", output)
	}
}

func TestRunOverlay_DryRunOutput_NoSummary(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Overlaid") {
		t.Errorf("dry-run should not have summary line: %s", output)
	}
}

func TestRunOverlay_QuietModeNoStdoutOutput(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, true, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no stdout output, got: %s", buf.String())
	}
}

func TestRunOverlay_NoRecipesFound_EmptyDir(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No recipes found") {
		t.Errorf("expected 'No recipes found' message: %s", output)
	}
}

// --- Section: compiled-home rebuild ---

func TestRunOverlay_CompiledHomeRebuiltFromScratch(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})

	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Fatal("dot_gitconfig should exist after first overlay")
	}

	// Delete the recipe directory.
	if err := os.RemoveAll(filepath.Join(recipesDir, "git")); err != nil {
		t.Fatal(err)
	}

	// Second overlay: compiled-home is cleared and rebuilt without git.
	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err == nil {
		t.Error("dot_gitconfig should not exist after recipe was removed")
	}

	loaded, err := state.Load(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Recipes["git"]; ok {
		t.Error("git should not be in state after recipe was removed")
	}
}

func TestRunOverlay_CompiledHomeRebuilt_RecipeIgnored(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		"dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Add alacritty to .recipeignore.
	if err := os.WriteFile(filepath.Join(recipesDir, ".recipeignore"), []byte("alacritty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_config/alacritty/alacritty.toml")); err == nil {
		t.Error("alacritty config should not exist after being ignored")
	}
}

func TestRunOverlay_CompiledHomeRebuilt_RecipeShrunk(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":         "[init]\n    defaultBranch = main\n",
		"dot_config/git/ignore": "*.swp\n",
	})

	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Remove one file from the recipe.
	os.Remove(filepath.Join(recipesDir, "git", "chezmoi", "dot_config/git/ignore"))

	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should still exist")
	}
	if _, err := os.Stat(filepath.Join(ch, "dot_config/git/ignore")); err == nil {
		t.Error("dot_config/git/ignore should not exist after being removed from recipe")
	}
}

// --- Section: home/ integration ---

func TestRunOverlay_CopiesHomeFirst(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestHomeFile(t, repoRoot, "dot_bashrc", "# bashrc\n")
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[user]\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)

	// home/ file should be in compiled-home.
	data, err := os.ReadFile(filepath.Join(ch, "dot_bashrc"))
	if err != nil {
		t.Fatal("home/ file dot_bashrc should be in compiled-home")
	}
	if string(data) != "# bashrc\n" {
		t.Errorf("dot_bashrc content = %q, want %q", string(data), "# bashrc\n")
	}

	// recipe file should also be in compiled-home.
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("recipe file dot_gitconfig should be in compiled-home")
	}
}

func TestRunOverlay_HomeRecipeConflict(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Same file in both home/ and recipe.
	setupTestHomeFile(t, repoRoot, "dot_gitconfig", "home version\n")
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "recipe version\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err == nil {
		t.Fatal("expected home/recipe conflict error")
	}

	hce, ok := err.(*overlay.HomeConflictError)
	if !ok {
		t.Fatalf("expected *overlay.HomeConflictError, got %T: %v", err, err)
	}
	if hce.RelPath != "dot_gitconfig" {
		t.Errorf("RelPath = %q, want %q", hce.RelPath, "dot_gitconfig")
	}
}

func TestRunOverlay_HomePlusRecipes(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestHomeFile(t, repoRoot, "dot_bashrc", "# bashrc\n")
	setupTestHomeFile(t, repoRoot, "dot_config/starship.toml", "[character]\n")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[user]\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	for _, f := range []string{"dot_bashrc", "dot_config/starship.toml", "dot_gitconfig", "dot_vimrc"} {
		if _, err := os.Stat(filepath.Join(ch, f)); err != nil {
			t.Errorf("%s should exist in compiled-home", f)
		}
	}
}

func TestRunOverlay_NoHomeDir_StillWorks(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Remove the home/ directory created by setupTestRepo.
	os.RemoveAll(filepath.Join(repoRoot, "home"))

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[user]\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err != nil {
		t.Error("recipe file should be in compiled-home even without home/ directory")
	}
}

func TestRunOverlay_DryRunSkipsClearAndCopy(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestHomeFile(t, repoRoot, "dot_bashrc", "# bashrc\n")
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	// compiled-home should not have been populated in dry-run.
	if _, err := os.Stat(filepath.Join(ch, "dot_bashrc")); err == nil {
		t.Error("home/ files should not be copied in dry-run mode")
	}
	if _, err := os.Stat(filepath.Join(ch, "dot_gitconfig")); err == nil {
		t.Error("recipe files should not be written in dry-run mode")
	}
}

// --- Section: Per-recipe .chezmoiignore ---

func TestRunOverlay_ChezmoiignoreMerged(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "{{ if .isContainer }}\nprivate_dot_config/alacritty/\n{{ end }}\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	data, err := os.ReadFile(filepath.Join(ch, ".chezmoiignore"))
	if err != nil {
		t.Fatalf(".chezmoiignore not written: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "scripts/") {
		t.Error("merged .chezmoiignore should contain scripts/")
	}
	if !strings.Contains(content, "# Recipe: alacritty") {
		t.Error("merged .chezmoiignore should contain recipe section")
	}
	if !strings.Contains(content, "private_dot_config/alacritty/") {
		t.Error("merged .chezmoiignore should contain recipe entries")
	}
}

func TestRunOverlay_ChezmoiignoreMultipleRecipesMerged(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "{{ if .isContainer }}\nprivate_dot_config/alacritty/\n{{ end }}\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})
	setupTestRecipe(t, recipesDir, "cartage", map[string]string{
		".chezmoiignore":                                  "{{ if .isContainer }}\nprivate_dot_config/systemd/\n{{ end }}\n",
		"private_dot_config/systemd/user/cartage.service": "service\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	data, err := os.ReadFile(filepath.Join(ch, ".chezmoiignore"))
	if err != nil {
		t.Fatalf(".chezmoiignore not written: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# Recipe: alacritty") {
		t.Error("should contain alacritty section")
	}
	if !strings.Contains(content, "# Recipe: cartage") {
		t.Error("should contain cartage section")
	}

	aIdx := strings.Index(content, "# Recipe: alacritty")
	cIdx := strings.Index(content, "# Recipe: cartage")
	if aIdx > cIdx {
		t.Error("alacritty should appear before cartage")
	}
}

func TestRunOverlay_ChezmoiignoreDryRunDoesNotWrite(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "private_dot_config/alacritty/\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	if _, err := os.Stat(filepath.Join(ch, ".chezmoiignore")); err == nil {
		t.Error(".chezmoiignore should not be written in dry-run mode")
	}

	output := buf.String()
	if !strings.Contains(output, ".chezmoiignore would be updated") {
		t.Errorf("dry-run should mention .chezmoiignore update: %s", output)
	}
}

func TestRunOverlay_ChezmoiignoreNotInState(t *testing.T) {
	setTestEnv(t)

	_, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "private_dot_config/alacritty/\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	store, err := state.Load(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	for name, rs := range store.Recipes {
		for _, f := range rs.Files {
			if f == ".chezmoiignore" {
				t.Errorf(".chezmoiignore should not be tracked in state (found in recipe %q)", name)
			}
		}
	}
}

func TestRunOverlay_ChezmoiignoreAlwaysHasScripts(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	ch := compiledHome(repoRoot)
	data, err := os.ReadFile(filepath.Join(ch, ".chezmoiignore"))
	if err != nil {
		t.Fatalf(".chezmoiignore not written: %v", err)
	}
	if !strings.Contains(string(data), "scripts/") {
		t.Error("scripts/ should always be in .chezmoiignore")
	}
}

func TestRunOverlay_NamedOverlaySkipsChezmoiignoreMerge(t *testing.T) {
	setTestEnv(t)

	repoRoot, recipesDir := setupTestRepo(t)
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		".chezmoiignore": "dot_gitconfig_extra\n",
		"dot_gitconfig":  "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"git"}, false, false, recipesDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("named overlay error = %v", err)
	}

	// Named overlay should not merge recipe .chezmoiignore entries.
	// DeploySharedScripts creates a minimal .chezmoiignore with scripts/,
	// but recipe-specific entries should not be merged.
	ch := compiledHome(repoRoot)
	data, err := os.ReadFile(filepath.Join(ch, ".chezmoiignore"))
	if err != nil {
		t.Fatalf(".chezmoiignore not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "scripts/") {
		t.Error("should still have scripts/ from DeploySharedScripts")
	}
	if strings.Contains(content, "# Recipe: git") {
		t.Error("named overlay should not merge recipe .chezmoiignore sections")
	}
	if strings.Contains(content, "dot_gitconfig_extra") {
		t.Error("named overlay should not include recipe ignore entries")
	}
}
