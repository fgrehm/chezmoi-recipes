package overlay

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ClearDir removes all contents of dir but not dir itself.
// Returns nil if dir does not exist or is already empty.
func ClearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("removing %s: %w", entry.Name(), err)
		}
	}
	return nil
}
