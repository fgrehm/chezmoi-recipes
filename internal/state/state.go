// Package state tracks which recipes have been applied and which files in the
// chezmoi source directory each recipe owns. State is persisted as JSON at the
// XDG data path for chezmoi-recipes.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RecipeState tracks the state of a single applied recipe.
type RecipeState struct {
	AppliedAt time.Time `json:"applied_at"`
	Files     []string  `json:"files"`
}

// Store holds the state of all applied recipes.
type Store struct {
	Recipes map[string]*RecipeState `json:"recipes"`
}

// Load reads the state file from disk. Returns an empty store if the file does not exist.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{Recipes: make(map[string]*RecipeState)}, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	if s.Recipes == nil {
		s.Recipes = make(map[string]*RecipeState)
	}
	return &s, nil
}

// Save writes the store to disk as JSON, creating parent directories as needed.
// The write is atomic: data is written to a temporary file in the same directory
// and then renamed into place, preventing a partial write from corrupting state.
func (s *Store) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return fmt.Errorf("creating temp state file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing state file: %w", err)
	}
	return nil
}

// AllFiles returns a map of every tracked file path to the recipe that owns it.
func (s *Store) AllFiles() map[string]string {
	files := make(map[string]string)
	for name, rs := range s.Recipes {
		for _, f := range rs.Files {
			files[f] = name
		}
	}
	return files
}

// RecordRecipe upserts the state for a recipe with the given files and current timestamp.
func (s *Store) RecordRecipe(name string, files []string) {
	s.Recipes[name] = &RecipeState{
		AppliedAt: time.Now(),
		Files:     files,
	}
}
