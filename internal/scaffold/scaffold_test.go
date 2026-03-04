package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_CreatesAllFiles(t *testing.T) {
	recipesDir := t.TempDir()
	var buf bytes.Buffer

	if err := Run(recipesDir, "mytool", &buf); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantFiles := []string{
		"README.md",
		"chezmoi/.chezmoiscripts/run_once_install-mytool.sh.tmpl",
		"chezmoi/.chezmoiignore",
		"chezmoi/private_dot_config/mytool/config.toml.tmpl",
		"chezmoi/dot_shellrc.d/mytool.sh",
	}

	for _, relPath := range wantFiles {
		fullPath := filepath.Join(recipesDir, "mytool", relPath)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("expected file %s to exist: %v", relPath, err)
		}
	}
}

func TestRun_NameSubstitution(t *testing.T) {
	recipesDir := t.TempDir()
	var buf bytes.Buffer

	if err := Run(recipesDir, "fzf", &buf); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	recipeDir := filepath.Join(recipesDir, "fzf")

	// Check install script filename contains recipe name.
	scriptPath := filepath.Join(recipeDir, "chezmoi/.chezmoiscripts/run_once_install-fzf.sh.tmpl")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("install script not found at expected path: %v", err)
	}

	// Check file contents have no <name> placeholders remaining.
	filesToCheck := []string{
		"README.md",
		"chezmoi/.chezmoiscripts/run_once_install-fzf.sh.tmpl",
		"chezmoi/.chezmoiignore",
		"chezmoi/private_dot_config/fzf/config.toml.tmpl",
		"chezmoi/dot_shellrc.d/fzf.sh",
	}

	for _, relPath := range filesToCheck {
		data, err := os.ReadFile(filepath.Join(recipeDir, relPath))
		if err != nil {
			t.Fatalf("reading %s: %v", relPath, err)
		}
		content := string(data)
		if strings.Contains(content, "<name>") {
			t.Errorf("%s still contains <name> placeholder", relPath)
		}
		if strings.Contains(content, "<NAME>") {
			t.Errorf("%s still contains <NAME> placeholder", relPath)
		}
	}

	// Check the shell module uses uppercase name for env var.
	shellData, err := os.ReadFile(filepath.Join(recipeDir, "chezmoi/dot_shellrc.d/fzf.sh"))
	if err != nil {
		t.Fatalf("reading shell module: %v", err)
	}
	if !strings.Contains(string(shellData), "FZF_HOME") {
		t.Error("shell module should contain FZF_HOME (uppercased name)")
	}
}

func TestRun_HyphenatedName(t *testing.T) {
	recipesDir := t.TempDir()
	var buf bytes.Buffer

	if err := Run(recipesDir, "my-tool", &buf); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Hyphens should become underscores in the uppercase variant.
	shellData, err := os.ReadFile(filepath.Join(recipesDir, "my-tool/chezmoi/dot_shellrc.d/my-tool.sh"))
	if err != nil {
		t.Fatalf("reading shell module: %v", err)
	}
	if !strings.Contains(string(shellData), "MY_TOOL_HOME") {
		t.Error("shell module should contain MY_TOOL_HOME (hyphens replaced with underscores)")
	}
}

func TestRun_AlreadyExists(t *testing.T) {
	recipesDir := t.TempDir()

	// Pre-create the recipe directory.
	if err := os.MkdirAll(filepath.Join(recipesDir, "mytool"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := Run(recipesDir, "mytool", &buf)
	if err == nil {
		t.Fatal("expected error for existing recipe")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

func TestRun_InvalidName(t *testing.T) {
	tests := []struct {
		label   string
		name    string
		wantErr string
	}{
		{"empty", "", "empty"},
		{"dot", ".", "invalid"},
		{"dotdot", "..", "invalid"},
		{"slash", "a/b", "path separators"},
		{"backslash", "a\\b", "path separators"},
		{"space", "a b", "whitespace"},
		{"tab", "a\tb", "whitespace"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			var buf bytes.Buffer
			err := Run(t.TempDir(), tc.name, &buf)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q should contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestRun_Output(t *testing.T) {
	recipesDir := t.TempDir()
	var buf bytes.Buffer

	if err := Run(recipesDir, "mytool", &buf); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, `"mytool"`) {
		t.Errorf("output should mention recipe name: %s", output)
	}
	if !strings.Contains(output, "README.md") {
		t.Errorf("output should list README.md: %s", output)
	}
	if !strings.Contains(output, "run_once_install-mytool.sh.tmpl") {
		t.Errorf("output should list install script: %s", output)
	}
	if !strings.Contains(output, "recipe-authoring.md") {
		t.Errorf("output should reference authoring guide: %s", output)
	}
}
