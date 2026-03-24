package overlay

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
)

// SourceEntry records which source contributed an entry and its raw source name.
type SourceEntry struct {
	Owner      string // "home" or recipe name
	SourcePath string // source-relative path, e.g. "private_dot_config/nvim/init.lua"
}

// ConflictError indicates two sources map to the same chezmoi target path.
// This covers both attribute mismatches (dot_config vs private_dot_config)
// and direct file collisions (same source name from home/ and a recipe).
type ConflictError struct {
	TargetPath string        // resolved target, e.g. ".config/nvim/init.lua"
	Entries    []SourceEntry // conflicting sources (2+)
}

func (e *ConflictError) Error() string {
	if len(e.Entries) == 2 && e.Entries[0].SourcePath == e.Entries[1].SourcePath {
		// Same source path, different owners (home vs recipe or recipe vs recipe).
		return fmt.Sprintf(
			"conflict: %q exists in both %s and %s\n"+
				"  hint: each file must belong to either home/ or exactly one recipe",
			e.Entries[0].SourcePath, e.Entries[0].Owner, e.Entries[1].Owner,
		)
	}
	// Different source paths mapping to the same target (attribute mismatch).
	var parts []string
	for _, entry := range e.Entries {
		parts = append(parts, fmt.Sprintf("%s (%s)", entry.SourcePath, entry.Owner))
	}
	return fmt.Sprintf(
		"conflict: target %q has mismatched source attributes:\n  %s\n"+
			"  hint: all sources must use the same chezmoi attribute prefixes for the same target path",
		e.TargetPath, strings.Join(parts, "\n  "),
	)
}

// DetectConflicts scans home/ and all recipe chezmoi/ directories for files
// and directories that resolve to the same target path from different sources.
// Returns the first *ConflictError found, or nil.
//
// This catches both direct file collisions (home/ and a recipe contributing
// the same source path) and attribute prefix mismatches (e.g. dot_config vs
// private_dot_config mapping to the same target).
func DetectConflicts(homeDir string, recipes []*recipe.Recipe) error {
	// Map from target path to the first source entry seen.
	seen := make(map[string]SourceEntry)

	// Scan home/ first.
	if err := scanSource(homeDir, "home", seen); err != nil {
		return err
	}

	// Scan each recipe's chezmoi/ directory.
	for _, r := range recipes {
		if !r.HasChezmoi {
			continue
		}
		chezmoiDir := filepath.Join(r.Dir, "chezmoi")
		if err := scanSource(chezmoiDir, r.Name, seen); err != nil {
			return err
		}
	}

	return nil
}

// scanSource walks root collecting all file and directory entries, resolves
// their target paths, and checks for conflicts against the seen map.
// seen is updated in place. Returns a *ConflictError on the first conflict.
func scanSource(root, owner string, seen map[string]SourceEntry) error {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip .chezmoiignore (handled separately by merge logic).
		if filepath.Base(relPath) == ".chezmoiignore" {
			return nil
		}

		targetPath := ParseTargetPath(relPath, d.IsDir())
		current := SourceEntry{Owner: owner, SourcePath: relPath}
		existing, found := seen[targetPath]

		if found {
			conflict := false
			if d.IsDir() {
				// Directories: multiple recipes can share a parent dir (e.g.
				// .config) as long as they agree on attributes (same source
				// name). Different source names = attribute conflict.
				//
				// This comparison uses filepath.Base because the seen map is
				// keyed by target path, and different recipes will have
				// different full relative paths (e.g. "dot_config/alacritty"
				// vs "dot_config/kitty"). We only care whether the directory
				// component that maps to this target uses the same chezmoi
				// source name. Walk ordering guarantees parent dirs are
				// visited before children, so a mismatch at any level is
				// caught before descending further.
				conflict = filepath.Base(existing.SourcePath) != filepath.Base(relPath)
			} else {
				// Files: same target from different owners always conflicts,
				// regardless of whether the source names match.
				conflict = existing.Owner != owner
			}
			if conflict {
				return &ConflictError{
					TargetPath: targetPath,
					Entries:    []SourceEntry{existing, current},
				}
			}
		} else {
			seen[targetPath] = current
		}

		return nil
	})
}
