package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_CreatesChezmoiroot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".chezmoiroot"))
	if err != nil {
		t.Fatalf(".chezmoiroot not found: %v", err)
	}
	if string(data) != "compiled-home\n" {
		t.Errorf(".chezmoiroot = %q, want %q", string(data), "compiled-home\n")
	}
}

func TestRunInit_CreatesHomeDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(repoRoot, "home"))
	if err != nil {
		t.Fatalf("home/ not found: %v", err)
	}
	if !info.IsDir() {
		t.Error("home/ should be a directory")
	}
}

func TestRunInit_WritesConfigToHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, "home", ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatalf(".chezmoi.toml.tmpl not in home/: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `[hooks.read-source-state.pre]`) {
		t.Error("config should have read-source-state.pre hook")
	}
	if !strings.Contains(content, `{{ .chezmoi.workingTree }}/recipes`) {
		t.Error("config should use {{ .chezmoi.workingTree }}/recipes for portable paths")
	}
}

func TestRunInit_AddsCompiledHomeToGitignore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	// Write an existing .gitignore.
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte("*.swp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf(".gitignore not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "compiled-home/") {
		t.Error(".gitignore should contain compiled-home/")
	}
	if !strings.Contains(content, "*.swp") {
		t.Error("existing .gitignore content should be preserved")
	}
}

func TestRunInit_GitignoreIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := RunInit(repoRoot, recipesDir, true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(string(data), "compiled-home/")
	if count != 1 {
		t.Errorf("compiled-home/ appears %d times in .gitignore, want 1", count)
	}
}

func TestRunInit_ConfigTemplateNoApplyPre(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, "home", ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, `[hooks.apply.pre]`) {
		t.Error("config should NOT have [hooks.apply.pre] section")
	}
	if !strings.Contains(content, "sourceDir") {
		t.Error("config should set sourceDir")
	}
}

func TestRunInit_ConfigTemplateHasGuardHooks(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, "home", ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	guardedCommands := []string{"add", "edit", "re-add", "merge", "chattr", "import", "forget", "destroy"}
	for _, cmd := range guardedCommands {
		section := "[hooks." + cmd + ".pre]"
		if !strings.Contains(content, section) {
			t.Errorf("config should have guard hook section %q", section)
		}
	}
}

func TestRunInit_CompiledHomePopulated(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}

	// compiled-home/ should contain .chezmoi.toml.tmpl (copied from home/).
	if _, err := os.Stat(filepath.Join(repoRoot, "compiled-home", ".chezmoi.toml.tmpl")); err != nil {
		t.Error("compiled-home/ should contain .chezmoi.toml.tmpl after init")
	}
}

func TestRunInit_CreatesRecipesDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(recipesDir)
	if err != nil {
		t.Fatalf("recipes dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("recipes should be a directory")
	}
}

func TestRunInit_SkipExistingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}

	result, err := RunInit(repoRoot, recipesDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.ConfigSkipped {
		t.Error("expected ConfigSkipped=true on second run")
	}
}

func TestRunInit_ForceOverwriteConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatal(err)
	}

	// Write custom content to verify it gets overwritten.
	configPath := filepath.Join(repoRoot, "home", ".chezmoi.toml.tmpl")
	if err := os.WriteFile(configPath, []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := RunInit(repoRoot, recipesDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.ConfigSkipped {
		t.Error("expected ConfigSkipped=false with force=true")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "custom" {
		t.Error("config should have been overwritten with force=true")
	}
}

func TestRunInit_CreatesEditorConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".editorconfig"))
	if err != nil {
		t.Fatalf(".editorconfig not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "indent_size = 2") {
		t.Error(".editorconfig should set indent_size = 2 for shell files")
	}
}

func TestRunInit_CreatesShellcheckRC(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".shellcheckrc"))
	if err != nil {
		t.Fatalf(".shellcheckrc not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "SC1091") {
		t.Error(".shellcheckrc should disable SC1091")
	}
}

func TestRunInit_CreatesReadme(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "README.md")); err != nil {
		t.Error("README.md should be created by init")
	}
}

func TestRunInit_DoesNotOverwriteExistingEditorConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	existing := "root = true\n"
	if err := os.WriteFile(filepath.Join(repoRoot, ".editorconfig"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".editorconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Error("existing .editorconfig should not be overwritten")
	}
}

func TestRunInit_DoesNotOverwriteExistingShellcheckRC(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	existing := "disable=SC1234\n"
	if err := os.WriteFile(filepath.Join(repoRoot, ".shellcheckrc"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".shellcheckrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Error("existing .shellcheckrc should not be overwritten")
	}
}

func TestRunInit_DoesNotOverwriteExistingReadme(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	repoRoot := t.TempDir()
	recipesDir := filepath.Join(repoRoot, "recipes")

	existing := "# My custom readme\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RunInit(repoRoot, recipesDir, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Error("existing README.md should not be overwritten")
	}
}

func TestWriteChezmoiConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	homeDir := t.TempDir()
	repoRoot := "/home/user/dotfiles"
	recipesDir := "/home/user/dotfiles/recipes"

	skipped, err := WriteChezmoiConfig(homeDir, repoRoot, recipesDir, false)
	if err != nil {
		t.Fatalf("WriteChezmoiConfig() error = %v", err)
	}
	if skipped {
		t.Error("expected skipped=false for new file")
	}

	data, err := os.ReadFile(filepath.Join(homeDir, ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatalf("reading .chezmoi.toml.tmpl: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "sourceDir") {
		t.Error("config should set sourceDir")
	}
	if strings.Contains(content, `[hooks.apply.pre]`) {
		t.Error("config should NOT have [hooks.apply.pre] section")
	}
	if !strings.Contains(content, `[hooks.read-source-state.pre]`) {
		t.Error("missing [hooks.read-source-state.pre] section")
	}
	if !strings.Contains(content, `{{ .chezmoi.workingTree }}/recipes`) {
		t.Error("config should use {{ .chezmoi.workingTree }}/recipes, not absolute path")
	}
	if strings.Contains(content, repoRoot) {
		t.Error("config should NOT contain absolute repo root path")
	}
	if !strings.Contains(content, `[data]`) {
		t.Error("missing [data] section")
	}
	if !strings.Contains(content, `promptStringOnce . "name"`) {
		t.Error("missing promptStringOnce for name")
	}
	if !strings.Contains(content, `$isContainer`) {
		t.Error("missing isContainer detection")
	}
}

func TestWriteChezmoiConfig_SkipExisting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	homeDir := t.TempDir()

	if _, err := WriteChezmoiConfig(homeDir, "/old", "/old/recipes", false); err != nil {
		t.Fatal(err)
	}

	skipped, err := WriteChezmoiConfig(homeDir, "/new", "/new/recipes", false)
	if err != nil {
		t.Fatal(err)
	}
	if !skipped {
		t.Error("expected skipped=true when file exists")
	}
}

func TestWriteChezmoiConfig_ForceOverwrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	homeDir := t.TempDir()

	if _, err := WriteChezmoiConfig(homeDir, "/old", "/old/recipes", false); err != nil {
		t.Fatal(err)
	}

	skipped, err := WriteChezmoiConfig(homeDir, "/new", "/new/recipes", true)
	if err != nil {
		t.Fatal(err)
	}
	if skipped {
		t.Error("expected skipped=false with force=true")
	}

	data, err := os.ReadFile(filepath.Join(homeDir, ".chezmoi.toml.tmpl"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Both old and new use relative path "recipes", so content should use
	// {{ .chezmoi.workingTree }}/recipes (not absolute paths).
	if !strings.Contains(content, `{{ .chezmoi.workingTree }}/recipes`) {
		t.Error("config should use {{ .chezmoi.workingTree }}/recipes")
	}
}
