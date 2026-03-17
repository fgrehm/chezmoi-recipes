package overlay

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyTree recursively copies all files from src into dst.
// Directories are created as needed. File permissions are preserved.
// Returns the list of relative paths copied.
// Returns nil, nil if src does not exist.
func CopyTree(src, dst string) ([]string, error) {
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	var copied []string
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("creating directory for %q: %w", relPath, err)
		}

		if err := copyFile(path, destPath); err != nil {
			return fmt.Errorf("copying %q: %w", relPath, err)
		}

		copied = append(copied, relPath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("copying tree from %s to %s: %w", src, dst, err)
	}
	return copied, nil
}
