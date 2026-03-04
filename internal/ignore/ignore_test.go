package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeChezmoiConfig writes a chezmoi.toml with the given [data] content.
func writeChezmoiConfig(t *testing.T, dir, dataContent string) string {
	t.Helper()
	configFile := filepath.Join(dir, "chezmoi.toml")
	content := "[data]\n" + dataContent
	writeFile(t, configFile, content)
	return configFile
}

func TestLoad_NoIgnoreFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty set, got %v", got)
	}
}

func TestLoad_EmptyIgnoreFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"), "")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty set, got %v", got)
	}
}

func TestLoad_PlainNames(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"), "git\nalacritty\n")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %v", got)
	}
	if !got["git"] {
		t.Error("expected git to be ignored")
	}
	if !got["alacritty"] {
		t.Error("expected alacritty to be ignored")
	}
}

func TestLoad_CommentsAndBlankLines(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	content := "# This is a comment\n\ngit\n\n# Another comment\nalacritty\n"
	writeFile(t, filepath.Join(recipesDir, ".recipeignore"), content)

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %v", got)
	}
	if !got["git"] || !got["alacritty"] {
		t.Errorf("expected git and alacritty, got %v", got)
	}
}

func TestLoad_TemplateConditional_True(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configDir := t.TempDir()
	configFile := writeChezmoiConfig(t, configDir, "    isContainer = true\n")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"),
		"{{ if .isContainer }}\nalacritty\n{{ end }}\n")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got["alacritty"] {
		t.Errorf("expected alacritty to be ignored when isContainer=true, got %v", got)
	}
}

func TestLoad_TemplateConditional_False(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configDir := t.TempDir()
	configFile := writeChezmoiConfig(t, configDir, "    isContainer = false\n")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"),
		"{{ if .isContainer }}\nalacritty\n{{ end }}\n")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got["alacritty"] {
		t.Errorf("expected alacritty NOT to be ignored when isContainer=false, got %v", got)
	}
}

func TestLoad_NoChezmoiConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	// Template references a variable but no chezmoi config exists.
	// With missingkey=zero, .isContainer evaluates to false (zero value).
	writeFile(t, filepath.Join(recipesDir, ".recipeignore"),
		"example\n{{ if .isContainer }}\nalacritty\n{{ end }}\n")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got["example"] {
		t.Error("expected example to be ignored")
	}
	if got["alacritty"] {
		t.Error("expected alacritty NOT to be ignored (isContainer is zero-valued)")
	}
}

func TestLoad_InvalidTemplate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"), "{{ if }}")

	_, err := Load(recipesDir, configFile)
	if err == nil {
		t.Fatal("expected error for invalid template syntax")
	}
}

func TestLoad_WhitespaceTrimmed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "chezmoi.toml")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"), "  git  \n\t alacritty \t\n")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got["git"] {
		t.Error("expected git (with whitespace trimmed)")
	}
	if !got["alacritty"] {
		t.Error("expected alacritty (with whitespace trimmed)")
	}
}

func TestLoad_ConfigWithNoDataSection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	recipesDir := t.TempDir()
	configDir := t.TempDir()
	// Config exists but has no [data] section.
	configFile := filepath.Join(configDir, "chezmoi.toml")
	writeFile(t, configFile, "[hooks.read-source-state.pre]\n    command = \"chezmoi-recipes\"\n")

	writeFile(t, filepath.Join(recipesDir, ".recipeignore"),
		"example\n{{ if .isContainer }}\nalacritty\n{{ end }}\n")

	got, err := Load(recipesDir, configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got["example"] {
		t.Error("expected example to be ignored")
	}
	if got["alacritty"] {
		t.Error("expected alacritty NOT to be ignored (no data section)")
	}
}
