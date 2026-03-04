package recipe

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// LoadAll discovers all recipes in the given base directory.
// A subdirectory is a recipe if it contains a README.md.
// Returns recipes sorted by name.
func LoadAll(recipesDir string) ([]*Recipe, error) {
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return nil, fmt.Errorf("reading recipes directory: %w", err)
	}

	var recipes []*Recipe
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(recipesDir, entry.Name())
		r, err := Load(dir)
		if err != nil {
			return nil, fmt.Errorf("loading recipe %q: %w", entry.Name(), err)
		}
		if r != nil {
			recipes = append(recipes, r)
		}
	}

	slices.SortFunc(recipes, func(a, b *Recipe) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return recipes, nil
}

// Load discovers a single recipe from the given directory.
// Returns nil if the directory does not contain a README.md (not a recipe).
func Load(dir string) (*Recipe, error) {
	readmePath := filepath.Join(dir, "README.md")
	info, err := os.Stat(readmePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("checking README.md: %w", err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("recipe %q has an empty README.md", filepath.Base(dir))
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving recipe directory: %w", err)
	}

	chezmoiDir := filepath.Join(absDir, "chezmoi")
	hasChezmoi := false
	emptyChezmoi := false
	if cInfo, err := os.Stat(chezmoiDir); err == nil && cInfo.IsDir() {
		hasChezmoi = true
		emptyChezmoi = isDirEmpty(chezmoiDir)
	}

	return &Recipe{
		Name:         filepath.Base(absDir),
		Dir:          absDir,
		HasChezmoi:   hasChezmoi,
		EmptyChezmoi: emptyChezmoi,
	}, nil
}

// isDirEmpty reports whether a directory contains no files (recursively).
func isDirEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		if !e.IsDir() {
			return false
		}
		if !isDirEmpty(filepath.Join(dir, e.Name())) {
			return false
		}
	}
	return true
}
