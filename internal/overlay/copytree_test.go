package overlay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTree_CopiesAllFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "a.txt"), "aaa")
	writeFile(t, filepath.Join(src, "b.txt"), "bbb")

	copied, err := CopyTree(src, dst)
	if err != nil {
		t.Fatalf("CopyTree() error = %v", err)
	}
	if len(copied) != 2 {
		t.Errorf("CopyTree() returned %d paths, want 2", len(copied))
	}

	assertFileContent(t, filepath.Join(dst, "a.txt"), "aaa")
	assertFileContent(t, filepath.Join(dst, "b.txt"), "bbb")
}

func TestCopyTree_CreatesNestedDirs(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "d1", "d2", "deep.txt"), "deep")

	_, err := CopyTree(src, dst)
	if err != nil {
		t.Fatalf("CopyTree() error = %v", err)
	}

	assertFileContent(t, filepath.Join(dst, "d1", "d2", "deep.txt"), "deep")
}

func TestCopyTree_PreservesPermissions(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	path := filepath.Join(src, "script.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := CopyTree(src, dst)
	if err != nil {
		t.Fatalf("CopyTree() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dst, "script.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("expected executable permissions, got %v", info.Mode())
	}
}

func TestCopyTree_EmptySource(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	copied, err := CopyTree(src, dst)
	if err != nil {
		t.Fatalf("CopyTree() error = %v", err)
	}
	if len(copied) != 0 {
		t.Errorf("CopyTree() returned %d paths, want 0", len(copied))
	}
}

func TestCopyTree_SourceDoesNotExist(t *testing.T) {
	dst := t.TempDir()

	copied, err := CopyTree(filepath.Join(t.TempDir(), "nope"), dst)
	if err != nil {
		t.Fatalf("CopyTree() error = %v, want nil for missing source", err)
	}
	if len(copied) != 0 {
		t.Errorf("CopyTree() returned %d paths, want 0", len(copied))
	}
}

func TestCopyTree_ReturnsRelativePaths(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "top.txt"), "t")
	writeFile(t, filepath.Join(src, "sub", "nested.txt"), "n")

	copied, err := CopyTree(src, dst)
	if err != nil {
		t.Fatalf("CopyTree() error = %v", err)
	}

	want := map[string]bool{"top.txt": true, filepath.Join("sub", "nested.txt"): true}
	for _, p := range copied {
		if !want[p] {
			t.Errorf("unexpected path %q in result", p)
		}
		delete(want, p)
	}
	for p := range want {
		t.Errorf("missing path %q in result", p)
	}
}

// helpers

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("%s content = %q, want %q", path, string(data), want)
	}
}
