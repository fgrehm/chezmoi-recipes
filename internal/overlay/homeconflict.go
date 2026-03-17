package overlay

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
)

// HomeConflictError indicates a file exists in both home/ and a recipe's chezmoi/ directory.
type HomeConflictError struct {
	RelPath string
	Recipe  string
}

func (e *HomeConflictError) Error() string {
	return fmt.Sprintf(
		"conflict: %q exists in both home/ and recipe %q\n"+
			"  hint: each file must belong to either home/ or exactly one recipe",
		e.RelPath, e.Recipe,
	)
}

// DetectHomeRecipeConflicts checks whether any file in homeDir also exists
// in a recipe's chezmoi/ directory. Returns a *HomeConflictError on the first
// collision found. Returns nil if homeDir does not exist.
// .chezmoiignore files are excluded from conflict detection (handled separately).
func DetectHomeRecipeConflicts(homeDir string, recipes []*recipe.Recipe) error {
	homePaths, err := collectRelPaths(homeDir)
	if err != nil {
		return err
	}
	if len(homePaths) == 0 {
		return nil
	}

	for _, r := range recipes {
		if !r.HasChezmoi {
			continue
		}
		chezmoiDir := filepath.Join(r.Dir, "chezmoi")
		recipePaths, err := collectRelPaths(chezmoiDir)
		if err != nil {
			return fmt.Errorf("scanning recipe %q: %w", r.Name, err)
		}
		for rp := range recipePaths {
			if homePaths[rp] {
				return &HomeConflictError{RelPath: rp, Recipe: r.Name}
			}
		}
	}
	return nil
}

// collectRelPaths walks a directory and returns a set of relative file paths.
// Returns nil if the directory does not exist.
// .chezmoiignore files are excluded.
func collectRelPaths(dir string) (map[string]bool, error) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	paths := make(map[string]bool)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath == ".chezmoiignore" {
			return nil
		}
		paths[relPath] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", dir, err)
	}
	return paths, nil
}
