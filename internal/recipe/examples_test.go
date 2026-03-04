package recipe

import (
	"path/filepath"
	"runtime"
	"testing"
)

// projectRoot returns the absolute path to the project root by navigating
// up from this test file's location to find the directory containing go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// internal/recipe/examples_test.go -> project root is 3 levels up
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolving project root: %v", err)
	}
	return abs
}

func TestExampleRecipes_LoadAll(t *testing.T) {
	root := projectRoot(t)
	recipesDir := filepath.Join(root, "examples")

	recipes, err := LoadAll(recipesDir)
	if err != nil {
		t.Fatalf("failed to load example recipes: %v", err)
	}

	if len(recipes) < 2 {
		t.Fatalf("expected at least 2 example recipes, got %d", len(recipes))
	}

	byName := make(map[string]*Recipe)
	for _, r := range recipes {
		byName[r.Name] = r
	}

	// Verify git recipe
	git, ok := byName["git"]
	if !ok {
		t.Fatal("expected git recipe to exist")
	}
	if !git.HasChezmoi {
		t.Error("git recipe should have a chezmoi/ directory")
	}

	// Verify ripgrep recipe
	rg, ok := byName["ripgrep"]
	if !ok {
		t.Fatal("expected ripgrep recipe to exist")
	}
	if !rg.HasChezmoi {
		t.Error("ripgrep recipe should have a chezmoi/ directory")
	}
}
