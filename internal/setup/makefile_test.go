package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureMakefile_CreatesFile(t *testing.T) {
	projectDir := t.TempDir()
	absRecipesDir := filepath.Join(projectDir, "recipes")

	created, err := EnsureMakefile(projectDir, absRecipesDir)
	if err != nil {
		t.Fatalf("EnsureMakefile() error = %v", err)
	}
	if !created {
		t.Error("expected created=true when Makefile does not exist")
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	content := string(data)

	// SHELL_FILES uses relative path to recipes dir.
	if !strings.Contains(content, "find recipes") {
		t.Errorf("expected 'find recipes' in SHELL_FILES, got:\n%s", content)
	}

	// All required targets are present.
	for _, target := range []string{"shell-fmt", "shell-fmt-check", "shell-lint", "check", "help"} {
		if !strings.Contains(content, target+":") {
			t.Errorf("expected target %q in Makefile", target)
		}
	}

	// Recipe lines use tabs (required by make).
	for _, cmd := range []string{"\tshfmt", "\tshellcheck", "\t@grep"} {
		if !strings.Contains(content, cmd) {
			t.Errorf("expected tab-indented command %q in Makefile", cmd)
		}
	}
}

func TestEnsureMakefile_SkipsIfExists(t *testing.T) {
	projectDir := t.TempDir()
	makefilePath := filepath.Join(projectDir, "Makefile")
	existing := "# existing Makefile content\n"
	if err := os.WriteFile(makefilePath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := EnsureMakefile(projectDir, filepath.Join(projectDir, "recipes"))
	if err != nil {
		t.Fatalf("EnsureMakefile() error = %v", err)
	}
	if created {
		t.Error("expected created=false when Makefile already exists")
	}

	// Existing content must be preserved.
	data, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Errorf("Makefile was modified: got %q, want %q", string(data), existing)
	}
}

func TestEnsureMakefile_RelativePath(t *testing.T) {
	projectDir := t.TempDir()
	absRecipesDir := filepath.Join(projectDir, "my-recipes")

	if _, err := EnsureMakefile(projectDir, absRecipesDir); err != nil {
		t.Fatalf("EnsureMakefile() error = %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, "Makefile"))
	if !strings.Contains(string(data), "find my-recipes") {
		t.Errorf("expected 'find my-recipes' for non-default recipes dir, got:\n%s", data)
	}
}
