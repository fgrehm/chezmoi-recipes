// Package overlay implements the recipe file overlay logic: planning what files
// would be copied from a recipe's chezmoi/ directory into the chezmoi source
// directory, and executing those copies.
package overlay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
)

// ConflictError indicates a file conflict between recipes or with an untracked file.
type ConflictError struct {
	RelPath       string
	ExistingOwner string // empty string means untracked (not owned by any recipe)
	Recipe        string
}

func (e *ConflictError) Error() string {
	if e.ExistingOwner == "" {
		return fmt.Sprintf(
			"conflict: %q already exists in source dir (untracked, not owned by any recipe)\n"+
				"  hint: remove the file from the source directory, or apply the recipe that owns it first",
			e.RelPath,
		)
	}
	return fmt.Sprintf("conflict: %q is owned by recipe %q, cannot be overwritten by recipe %q", e.RelPath, e.ExistingOwner, e.Recipe)
}

// Result describes what a Plan or Execute operation will do or did.
type Result struct {
	Added     []string
	Updated   []string
	Unchanged []string
}

// Files returns all affected file paths (added + updated).
func (r *Result) Files() []string {
	files := make([]string, 0, len(r.Added)+len(r.Updated))
	files = append(files, r.Added...)
	files = append(files, r.Updated...)
	return files
}

// AllFiles returns all file paths owned by this recipe (added + updated + unchanged).
func (r *Result) AllFiles() []string {
	files := make([]string, 0, len(r.Added)+len(r.Updated)+len(r.Unchanged))
	files = append(files, r.Added...)
	files = append(files, r.Updated...)
	files = append(files, r.Unchanged...)
	return files
}

// Plan walks the recipe's chezmoi/ directory and determines what files would be
// overlaid into the source directory. It checks for conflicts against the state
// and existing files on disk.
func Plan(_ context.Context, r *recipe.Recipe, sourceDir string, store *state.Store) (*Result, error) {
	chezmoiDir := filepath.Join(r.Dir, "chezmoi")
	info, err := os.Stat(chezmoiDir)
	if err != nil {
		return nil, fmt.Errorf("recipe %q has no chezmoi directory: %w", r.Name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("recipe %q: chezmoi path is not a directory", r.Name)
	}

	result := &Result{}
	owners := store.AllFiles()

	err = filepath.WalkDir(chezmoiDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(chezmoiDir, path)
		if err != nil {
			return err
		}

		// Skip .chezmoiignore; handled separately by MergeChezmoiIgnore.
		if relPath == ".chezmoiignore" {
			return nil
		}

		destPath := filepath.Join(sourceDir, relPath)
		_, statErr := os.Stat(destPath)
		fileExists := statErr == nil

		if fileExists {
			owner := owners[relPath]
			switch owner {
			case r.Name:
				changed, err := fileContentsDiffer(path, destPath)
				if err != nil {
					return fmt.Errorf("comparing %q: %w", relPath, err)
				}
				if changed {
					result.Updated = append(result.Updated, relPath)
				} else {
					result.Unchanged = append(result.Unchanged, relPath)
				}
			case "":
				return &ConflictError{RelPath: relPath, Recipe: r.Name}
			default:
				return &ConflictError{RelPath: relPath, ExistingOwner: owner, Recipe: r.Name}
			}
		} else {
			result.Added = append(result.Added, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// copyFile reads src and writes its contents to dest, preserving the source file's permissions.
func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, info.Mode())
}

// fileContentsDiffer returns true if the two files have different contents.
func fileContentsDiffer(a, b string) (bool, error) {
	dataA, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	dataB, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(dataA, dataB), nil
}

// ReadIgnoreEntries reads a recipe's chezmoi/.chezmoiignore file and returns
// its contents. Returns empty string if the file does not exist.
func ReadIgnoreEntries(r *recipe.Recipe) (string, error) {
	path := filepath.Join(r.Dir, "chezmoi", ".chezmoiignore")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("reading .chezmoiignore for recipe %q: %w", r.Name, err)
	}
	return string(data), nil
}

// Execute plans and then copies recipe files into the source directory.
// It creates directories as needed and preserves file permissions.
func Execute(ctx context.Context, r *recipe.Recipe, sourceDir string, store *state.Store) (*Result, error) {
	result, err := Plan(ctx, r, sourceDir, store)
	if err != nil {
		return nil, err
	}

	chezmoiDir := filepath.Join(r.Dir, "chezmoi")

	for _, relPath := range result.Files() {
		srcPath := filepath.Join(chezmoiDir, relPath)
		destPath := filepath.Join(sourceDir, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return nil, fmt.Errorf("creating directory for %q: %w", relPath, err)
		}

		if err := copyFile(srcPath, destPath); err != nil {
			return nil, fmt.Errorf("copying %q: %w", relPath, err)
		}
	}

	return result, nil
}
