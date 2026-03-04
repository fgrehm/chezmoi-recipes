package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// removeFileAndCleanDirs removes a file and then walks up the directory tree,
// removing empty parent directories until it reaches rootDir.
// Returns nil if the file does not exist (already gone).
func removeFileAndCleanDirs(fullPath, rootDir string) error {
	if err := os.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	dir := filepath.Dir(fullPath)
	for dir != rootDir && strings.HasPrefix(dir, rootDir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		// Error intentionally ignored: a concurrent write or permission issue
		// just means the directory stays around, which is harmless.
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
	return nil
}
