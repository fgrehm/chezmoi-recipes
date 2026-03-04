package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgrehm/chezmoi-recipes/internal/overlay"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

func TestRunOverlay_NoArgsLoadsAllRecipes(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// Both recipes should be overlaid.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig not overlaid")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "dot_vimrc")); err != nil {
		t.Error("dot_vimrc not overlaid")
	}
}

func TestRunOverlay_NoArgsFiltersRecipesByRecipeignore(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})
	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		"dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	// Write .recipeignore to skip alacritty.
	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	if err := os.WriteFile(ignoreFile, []byte("alacritty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// git should be overlaid, alacritty should not.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should be overlaid")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "dot_config/alacritty/alacritty.toml")); err == nil {
		t.Error("alacritty should NOT be overlaid (filtered by .recipeignore)")
	}
}

func TestRunOverlay_NamedArgsOverlayOnlySpecifiedRecipes(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "git config\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"git"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should be overlaid")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "dot_vimrc")); err == nil {
		t.Error("dot_vimrc should NOT be overlaid (not specified)")
	}
}

func TestRunOverlay_NamedArgsOverrideRecipeignore(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		"dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	// Ignore alacritty.
	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	if err := os.WriteFile(ignoreFile, []byte("alacritty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	// Explicitly name alacritty -- should override ignore.
	err := runOverlay(context.Background(), []string{"alacritty"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(srcDir, "dot_config/alacritty/alacritty.toml")); err != nil {
		t.Error("alacritty should be overlaid even though it's in .recipeignore (explicit name overrides)")
	}
}

func TestRunOverlay_DoesNotInvokeChezmoi(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	// runOverlay has no chezmoi.Runner parameter at all.
	// This test verifies the signature: if it compiled, chezmoi is not invoked.
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// The output should NOT mention "chezmoi apply" or "Running chezmoi".
	output := buf.String()
	if strings.Contains(output, "chezmoi apply") || strings.Contains(output, "Running chezmoi") {
		t.Errorf("overlay should not reference chezmoi invocation, got: %s", output)
	}
}

func TestRunOverlay_ConflictOnSecondRecipe_StopsNoStateSaved(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Both recipes add the same file -- conflict on second recipe.
	setupTestRecipe(t, recipesDir, "aaa", map[string]string{
		"dot_shared": "from aaa\n",
	})
	setupTestRecipe(t, recipesDir, "bbb", map[string]string{
		"dot_shared": "from bbb\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if _, ok := err.(*overlay.ConflictError); !ok {
		t.Errorf("expected *overlay.ConflictError, got %T: %v", err, err)
	}

	// State should NOT be saved.
	if _, err := os.Stat(stateFile); err == nil {
		t.Error("state file should not be created when overlay fails")
	}
}

func TestRunOverlay_DryRunMultipleRecipes(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// Files should NOT be written in dry-run.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err == nil {
		t.Error("dot_gitconfig should not exist in dry-run mode")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "dot_vimrc")); err == nil {
		t.Error("dot_vimrc should not exist in dry-run mode")
	}

	// State should NOT be saved.
	if _, err := os.Stat(stateFile); err == nil {
		t.Error("state file should not be created during dry-run")
	}

	// Output should mention dry-run.
	output := buf.String()
	if !strings.Contains(output, "dry-run") {
		t.Errorf("output should mention dry-run: %s", output)
	}
}

func TestRunOverlay_RecipesOverlaidInAlphabeticalOrder(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Create recipes in non-alphabetical order but pass them reversed.
	setupTestRecipe(t, recipesDir, "zzz", map[string]string{
		"dot_zzz": "zzz\n",
	})
	setupTestRecipe(t, recipesDir, "aaa", map[string]string{
		"dot_aaa": "aaa\n",
	})

	var buf bytes.Buffer
	// Pass names in reverse order.
	err := runOverlay(context.Background(), []string{"zzz", "aaa"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// Both should be overlaid.
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Empty recipes dir -- no recipes.
	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	// Ignore the only recipe.
	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	if err := os.WriteFile(ignoreFile, []byte("git\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// State file should exist.
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"nonexistent"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err == nil {
		t.Fatal("expected error for nonexistent recipe")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestRunOverlay_QuietSuppressesOutput(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, true, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got: %s", buf.String())
	}

	// But files should still be overlaid.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should be overlaid even in quiet mode")
	}
}

func TestRunOverlay_IdempotentOverlay(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})

	// Run overlay twice.
	var buf1 bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf1); err != nil {
		t.Fatalf("first runOverlay() error = %v", err)
	}

	var buf2 bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf2); err != nil {
		t.Fatalf("second runOverlay() error = %v", err)
	}

	// File should still exist with correct content.
	data, err := os.ReadFile(filepath.Join(srcDir, "dot_gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "defaultBranch") {
		t.Error("file content incorrect after idempotent overlay")
	}
}

// --- Section 3: Output formatting ---

func TestRunOverlay_SingleRecipeOutput_NoPrefix(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":        "content\n",
		"dot_config/git/ignore": "*.swp\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), []string{"git"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	// Single recipe should NOT have [N/M] prefix.
	if strings.Contains(output, "[1/1]") {
		t.Errorf("single recipe should not have [N/M] prefix: %s", output)
	}
	// Should have the recipe name and file names.
	if !strings.Contains(output, "git") {
		t.Errorf("output should contain recipe name: %s", output)
	}
	if !strings.Contains(output, "dot_gitconfig") {
		t.Errorf("output should contain file name: %s", output)
	}
}

func TestRunOverlay_MultiRecipeOutput_WithPrefix(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":        "content\n",
		"dot_config/git/ignore": "*.swp\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	// Should mention "2 recipes" and "3 added" in summary.
	if !strings.Contains(output, "2 recipes") {
		t.Errorf("summary should mention recipe count: %s", output)
	}
	if !strings.Contains(output, "3 added") {
		t.Errorf("summary should mention added count: %s", output)
	}
}

func TestRunOverlay_NoChangesPerRecipe(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Recipe with no chezmoi dir (README only, HasChezmoi=false).
	// Actually, recipes without chezmoi/ are skipped by LoadAll.
	// Instead, use a recipe with an empty chezmoi/ dir.
	recipeDir := filepath.Join(recipesDir, "empty")
	if err := os.MkdirAll(filepath.Join(recipeDir, "chezmoi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recipeDir, "README.md"), []byte("# empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also add a real recipe so we get multi-recipe output.
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	output := buf.String()
	// Dry-run should NOT have summary line "Overlaid N recipes".
	if strings.Contains(output, "Overlaid") {
		t.Errorf("dry-run should not have summary line: %s", output)
	}
}

func TestRunOverlay_QuietModeNoStdoutOutput(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, true, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no stdout output, got: %s", buf.String())
	}
}

func TestRunOverlay_NoRecipesFound_EmptyDir(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No recipes found") {
		t.Errorf("expected 'No recipes found' message: %s", output)
	}
}

// --- Section: Stale file cleanup ---

func TestRunOverlay_StaleCleanup_RecipeDeleted(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// First overlay: apply git recipe.
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "[init]\n    defaultBranch = main\n",
	})

	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Fatal("dot_gitconfig should exist after first overlay")
	}

	// Delete the recipe directory.
	if err := os.RemoveAll(filepath.Join(recipesDir, "git")); err != nil {
		t.Fatal(err)
	}

	// Second overlay: git recipe is gone.
	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	// Stale file should be removed.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); !errors.Is(err, os.ErrNotExist) {
		t.Error("dot_gitconfig should be removed as stale")
	}

	// State should not contain git.
	loaded, err := state.Load(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Recipes["git"]; ok {
		t.Error("git should be removed from state")
	}
}

func TestRunOverlay_StaleCleanup_RecipeIgnored(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		"dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	// First overlay: apply alacritty.
	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Add alacritty to .recipeignore.
	if err := os.WriteFile(filepath.Join(recipesDir, ".recipeignore"), []byte("alacritty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second overlay: alacritty is ignored.
	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	// Stale file should be removed.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_config/alacritty/alacritty.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Error("alacritty config should be removed as stale")
	}

	// Empty parent dirs should be cleaned up.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_config/alacritty")); !errors.Is(err, os.ErrNotExist) {
		t.Error("empty dot_config/alacritty/ should be cleaned up")
	}
}

func TestRunOverlay_StaleCleanup_RecipeShrunk(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// First overlay: git has 2 files.
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":         "[init]\n    defaultBranch = main\n",
		"dot_config/git/ignore": "*.swp\n",
	})

	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Remove one file from the recipe.
	os.Remove(filepath.Join(recipesDir, "git", "chezmoi", "dot_config/git/ignore"))

	// Second overlay: git now has 1 file.
	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	// Remaining file should still exist.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should still exist")
	}

	// Removed file should be cleaned up.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_config/git/ignore")); !errors.Is(err, os.ErrNotExist) {
		t.Error("dot_config/git/ignore should be removed as stale")
	}
}

func TestRunOverlay_StaleCleanup_DryRunDoesNotDelete(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	// First overlay (non-dry-run to create state).
	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Delete recipe.
	os.RemoveAll(filepath.Join(recipesDir, "git"))

	// Dry-run overlay.
	buf.Reset()
	if err := runOverlay(context.Background(), nil, true, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("dry-run overlay error = %v", err)
	}

	// File should still exist (dry-run).
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should still exist in dry-run mode")
	}

	// Output should mention stale files.
	output := buf.String()
	if !strings.Contains(output, "Would remove") {
		t.Errorf("dry-run should mention stale files: %s", output)
	}
	if !strings.Contains(output, "dot_gitconfig") {
		t.Errorf("dry-run should list stale file: %s", output)
	}
}

func TestRunOverlay_StaleCleanup_NamedArgsSkipCleanup(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})
	setupTestRecipe(t, recipesDir, "vim", map[string]string{
		"dot_vimrc": "set number\n",
	})

	// First overlay: apply both.
	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Delete git recipe.
	os.RemoveAll(filepath.Join(recipesDir, "git"))

	// Named overlay for vim only -- should NOT clean git's stale files.
	buf.Reset()
	if err := runOverlay(context.Background(), []string{"vim"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("named overlay error = %v", err)
	}

	// Git's file should still be there.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); err != nil {
		t.Error("dot_gitconfig should NOT be cleaned up during named overlay")
	}
}

func TestRunOverlay_StaleCleanup_QuietSuppressesOutput(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	// First overlay.
	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	// Delete recipe.
	os.RemoveAll(filepath.Join(recipesDir, "git"))

	// Quiet overlay.
	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, true, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("quiet overlay error = %v", err)
	}

	// No output in quiet mode.
	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got: %s", buf.String())
	}

	// But file should still be cleaned up.
	if _, err := os.Stat(filepath.Join(srcDir, "dot_gitconfig")); !errors.Is(err, os.ErrNotExist) {
		t.Error("stale file should be removed even in quiet mode")
	}
}

// --- Section: Per-recipe .chezmoiignore ---

func TestRunOverlay_ChezmoiignoreMerged(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "{{ if .isContainer }}\nprivate_dot_config/alacritty/\n{{ end }}\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(srcDir, ".chezmoiignore"))
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "{{ if .isContainer }}\nprivate_dot_config/alacritty/\n{{ end }}\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})
	setupTestRecipe(t, recipesDir, "cartage", map[string]string{
		".chezmoiignore":                                      "{{ if .isContainer }}\nprivate_dot_config/systemd/\n{{ end }}\n",
		"private_dot_config/systemd/user/cartage.service": "service\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(srcDir, ".chezmoiignore"))
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

	// Verify alphabetical order.
	aIdx := strings.Index(content, "# Recipe: alacritty")
	cIdx := strings.Index(content, "# Recipe: cartage")
	if aIdx > cIdx {
		t.Error("alacritty should appear before cartage")
	}
}

func TestRunOverlay_ChezmoiignoreDryRunDoesNotWrite(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "private_dot_config/alacritty/\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, true, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// .chezmoiignore should NOT be written in dry-run mode.
	if _, err := os.Stat(filepath.Join(srcDir, ".chezmoiignore")); err == nil {
		t.Error(".chezmoiignore should not be written in dry-run mode")
	}

	// Output should mention the update.
	output := buf.String()
	if !strings.Contains(output, ".chezmoiignore would be updated") {
		t.Errorf("dry-run should mention .chezmoiignore update: %s", output)
	}
}

func TestRunOverlay_ChezmoiignoreNotInState(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "private_dot_config/alacritty/\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	// .chezmoiignore should not appear in state.
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

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// Recipe without .chezmoiignore.
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig": "content\n",
	})

	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("runOverlay() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(srcDir, ".chezmoiignore"))
	if err != nil {
		t.Fatalf(".chezmoiignore not written: %v", err)
	}
	if !strings.Contains(string(data), "scripts/") {
		t.Error("scripts/ should always be in .chezmoiignore")
	}
}

func TestRunOverlay_NamedOverlayDoesNotRegenerateChezmoiignore(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "alacritty", map[string]string{
		".chezmoiignore":                              "private_dot_config/alacritty/\n",
		"private_dot_config/alacritty/alacritty.toml": "font_size = 12\n",
	})
	setupTestRecipe(t, recipesDir, "git", map[string]string{
		".chezmoiignore": "dot_gitconfig_extra\n",
		"dot_gitconfig":  "content\n",
	})

	// Full overlay first.
	var buf bytes.Buffer
	err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("full overlay error = %v", err)
	}

	// Read the merged .chezmoiignore.
	before, err := os.ReadFile(filepath.Join(srcDir, ".chezmoiignore"))
	if err != nil {
		t.Fatal(err)
	}

	// Named overlay for git only.
	buf.Reset()
	err = runOverlay(context.Background(), []string{"git"}, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf)
	if err != nil {
		t.Fatalf("named overlay error = %v", err)
	}

	// .chezmoiignore should be unchanged.
	after, err := os.ReadFile(filepath.Join(srcDir, ".chezmoiignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("named overlay should not regenerate .chezmoiignore")
	}
}

func TestRunOverlay_StaleCleanup_OutputShowsRemovedFiles(t *testing.T) {
	setTestEnv(t)

	recipesDir := t.TempDir()
	srcDir := t.TempDir()
	stateFile := filepath.Join(t.TempDir(), "state.json")

	setupTestRecipe(t, recipesDir, "git", map[string]string{
		"dot_gitconfig":         "content\n",
		"dot_config/git/ignore": "*.swp\n",
	})

	var buf bytes.Buffer
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("first overlay error = %v", err)
	}

	os.RemoveAll(filepath.Join(recipesDir, "git"))

	buf.Reset()
	if err := runOverlay(context.Background(), nil, false, false, recipesDir, srcDir, stateFile, chezmoiConfigFile(t), &buf); err != nil {
		t.Fatalf("second overlay error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Removed 2 stale files") {
		t.Errorf("output should mention removed stale files count: %s", output)
	}
	if !strings.Contains(output, "dot_gitconfig") {
		t.Errorf("output should list removed file: %s", output)
	}
}
