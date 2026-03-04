// Package scaffold generates new recipe directory scaffolding with annotated
// example files demonstrating chezmoi conventions.
package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// scaffoldFile describes a file to generate inside the recipe directory.
type scaffoldFile struct {
	// relPath is relative to the recipe directory. May contain <name>.
	relPath string
	content string
}

func files() []scaffoldFile {
	return []scaffoldFile{
		{"README.md", readmeTmpl},
		{"chezmoi/.chezmoiscripts/run_once_install-<name>.sh.tmpl", installScriptTmpl},
		{"chezmoi/.chezmoiignore", chezmoiIgnoreTmpl},
		{"chezmoi/private_dot_config/<name>/config.toml.tmpl", configTmpl},
		{"chezmoi/dot_shellrc.d/<name>.sh", shellModuleTmpl},
	}
}

// Run creates a new recipe directory with scaffolded files that demonstrate
// chezmoi conventions. The recipe name is interpolated into file paths and
// contents.
func Run(recipesDir, name string, w io.Writer) error {
	if err := validateName(name); err != nil {
		return err
	}

	recipeDir := filepath.Join(recipesDir, name)
	if _, err := os.Stat(recipeDir); err == nil {
		return fmt.Errorf("recipe %q already exists in %s", name, recipesDir)
	}

	upperName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	scaffoldFiles := files()
	relPaths := make([]string, len(scaffoldFiles))
	for i, f := range scaffoldFiles {
		relPath := strings.ReplaceAll(f.relPath, "<name>", name)
		relPaths[i] = relPath
		fullPath := filepath.Join(recipeDir, relPath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", relPath, err)
		}

		content := strings.ReplaceAll(f.content, "<name>", name)
		content = strings.ReplaceAll(content, "<NAME>", upperName)

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", relPath, err)
		}
	}

	_, _ = fmt.Fprintf(w, "Scaffolded recipe %q in %s/\n\n", name, filepath.Join(recipesDir, name))
	for _, relPath := range relPaths {
		_, _ = fmt.Fprintf(w, "  %s\n", relPath)
	}
	_, _ = fmt.Fprintf(w, "\nSee docs/recipe-authoring.md for the full authoring guide.\n")

	return nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("recipe name must not be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("recipe name %q is invalid", name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("recipe name %q must not contain path separators", name)
	}
	for _, r := range name {
		if unicode.IsSpace(r) {
			return fmt.Errorf("recipe name %q must not contain whitespace", name)
		}
	}
	return nil
}
