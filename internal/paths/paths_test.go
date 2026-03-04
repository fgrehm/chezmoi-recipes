package paths

import (
	"path/filepath"
	"testing"
)

func TestDefaultSourceDir_WithXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	got, err := DefaultSourceDir()
	if err != nil {
		t.Fatalf("DefaultSourceDir() error = %v", err)
	}
	want := "/tmp/xdg-data/chezmoi-recipes/source"
	if got != want {
		t.Errorf("DefaultSourceDir() = %q, want %q", got, want)
	}
}

func TestDefaultSourceDir_FallbackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/fakehome")

	got, err := DefaultSourceDir()
	if err != nil {
		t.Fatalf("DefaultSourceDir() error = %v", err)
	}
	want := filepath.Join("/tmp/fakehome", ".local", "share", "chezmoi-recipes", "source")
	if got != want {
		t.Errorf("DefaultSourceDir() = %q, want %q", got, want)
	}
}

func TestDefaultStateFile_WithXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	got, err := DefaultStateFile()
	if err != nil {
		t.Fatalf("DefaultStateFile() error = %v", err)
	}
	want := "/tmp/xdg-data/chezmoi-recipes/state.json"
	if got != want {
		t.Errorf("DefaultStateFile() = %q, want %q", got, want)
	}
}

func TestDefaultStateFile_FallbackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/fakehome")

	got, err := DefaultStateFile()
	if err != nil {
		t.Fatalf("DefaultStateFile() error = %v", err)
	}
	want := filepath.Join("/tmp/fakehome", ".local", "share", "chezmoi-recipes", "state.json")
	if got != want {
		t.Errorf("DefaultStateFile() = %q, want %q", got, want)
	}
}
