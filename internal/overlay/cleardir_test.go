package overlay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearDir_RemovesAllContents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "a")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "b")

	if err := ClearDir(dir); err != nil {
		t.Fatalf("ClearDir() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ClearDir() left %d entries, want 0", len(entries))
	}
}

func TestClearDir_PreservesRootDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "a")

	if err := ClearDir(dir); err != nil {
		t.Fatalf("ClearDir() error = %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("root dir should still exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("root should be a directory")
	}
}

func TestClearDir_NoErrorOnEmpty(t *testing.T) {
	dir := t.TempDir()

	if err := ClearDir(dir); err != nil {
		t.Fatalf("ClearDir() on empty dir error = %v", err)
	}
}

func TestClearDir_NoErrorOnMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	if err := ClearDir(missing); err != nil {
		t.Fatalf("ClearDir() on missing dir error = %v", err)
	}
}
