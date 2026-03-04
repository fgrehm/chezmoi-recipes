package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteChezmoiConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()
	recipesDir := "/home/user/dotfiles/recipes"

	skipped, err := WriteChezmoiConfig(sourceDir, recipesDir, false)
	if err != nil {
		t.Fatalf("WriteChezmoiConfig() error = %v", err)
	}
	if skipped {
		t.Error("expected skipped=false for new file")
	}

	data, err := os.ReadFile(filepath.Join(sourceDir, ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatalf("reading .chezmoi.toml.tmpl: %v", err)
	}

	content := string(data)

	// sourceDir is set to the chezmoi-recipes source directory.
	if !strings.Contains(content, fmt.Sprintf("sourceDir = %q", sourceDir)) {
		t.Error("missing or incorrect sourceDir setting")
	}

	// Hook config is present with correct recipes dir.
	if !strings.Contains(content, `[hooks.read-source-state.pre]`) {
		t.Error("missing hook config section")
	}
	if !strings.Contains(content, recipesDir) {
		t.Errorf("missing recipes dir %q in template", recipesDir)
	}

	// Data section uses chezmoi template functions.
	if !strings.Contains(content, `[data]`) {
		t.Error("missing [data] section")
	}
	if !strings.Contains(content, fmt.Sprintf("recipesDir = %q", recipesDir)) {
		t.Error("missing recipesDir in [data] section")
	}
	if !strings.Contains(content, `promptStringOnce . "name"`) {
		t.Error("missing promptStringOnce for name")
	}
	if !strings.Contains(content, `promptStringOnce . "email"`) {
		t.Error("missing promptStringOnce for email")
	}
	if !strings.Contains(content, `$isContainer`) {
		t.Error("missing isContainer detection")
	}
	if !strings.Contains(content, `$isDebian`) {
		t.Error("missing isDebian detection")
	}
	if !strings.Contains(content, `$hasNvidiaGPU`) {
		t.Error("missing hasNvidiaGPU detection")
	}
}

func TestWriteChezmoiConfig_SkipExisting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()

	if _, err := WriteChezmoiConfig(sourceDir, "/old/recipes", false); err != nil {
		t.Fatal(err)
	}

	skipped, err := WriteChezmoiConfig(sourceDir, "/new/recipes", false)
	if err != nil {
		t.Fatal(err)
	}
	if !skipped {
		t.Error("expected skipped=true when file exists")
	}

	data, err := os.ReadFile(filepath.Join(sourceDir, ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "/old/recipes") {
		t.Error("original content should be preserved")
	}
	if strings.Contains(content, "/new/recipes") {
		t.Error("file should not have been overwritten")
	}
}

func TestWriteChezmoiConfig_ForceOverwrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()

	if _, err := WriteChezmoiConfig(sourceDir, "/old/recipes", false); err != nil {
		t.Fatal(err)
	}

	skipped, err := WriteChezmoiConfig(sourceDir, "/new/recipes", true)
	if err != nil {
		t.Fatal(err)
	}
	if skipped {
		t.Error("expected skipped=false with force=true")
	}

	data, err := os.ReadFile(filepath.Join(sourceDir, ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, "/old/recipes") {
		t.Error("old recipes dir should be overwritten")
	}
	if !strings.Contains(content, "/new/recipes") {
		t.Errorf("expected new recipes dir, got:\n%s", content)
	}
}

func TestRunInit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	tmp := t.TempDir()
	sourceDir := filepath.Join(tmp, "source")
	recipesDir := filepath.Join(tmp, "recipes")

	result, err := RunInit(sourceDir, recipesDir, false)
	if err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}
	if result.ConfigSkipped {
		t.Error("expected ConfigSkipped=false on first run")
	}

	// Verify source dir was created.
	if _, err := os.Stat(sourceDir); err != nil {
		t.Errorf("source dir not created: %v", err)
	}

	// Verify .chezmoi.toml.tmpl exists.
	if _, err := os.Stat(filepath.Join(sourceDir, ".chezmoi.toml.tmpl")); err != nil {
		t.Errorf(".chezmoi.toml.tmpl not found: %v", err)
	}

	// Verify shared scripts deployed.
	if _, err := os.Stat(filepath.Join(sourceDir, "scripts", "ui.bash")); err != nil {
		t.Errorf("scripts/ui.bash not found: %v", err)
	}

	// Verify recipes dir was created.
	if _, err := os.Stat(recipesDir); err != nil {
		t.Errorf("recipes dir not created: %v", err)
	}

	// Second run should skip config.
	result, err = RunInit(sourceDir, recipesDir, false)
	if err != nil {
		t.Fatalf("RunInit() second run error = %v", err)
	}
	if !result.ConfigSkipped {
		t.Error("expected ConfigSkipped=true on second run")
	}
}
