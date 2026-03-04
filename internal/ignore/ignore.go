// Package ignore implements .recipeignore processing. A .recipeignore file
// lists recipe names to skip during overlay, optionally using Go template
// conditionals evaluated against chezmoi's rendered config data.
package ignore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
)

// Load reads .recipeignore from recipesDir and processes it as a Go template
// using data from chezmoi's rendered config file. Returns a set of recipe
// names to skip.
//
// If .recipeignore does not exist, returns an empty set (no recipes ignored).
// If the chezmoi config does not exist, the template executes with an empty data map.
func Load(recipesDir, chezmoiConfigFile string) (map[string]bool, error) {
	ignoreFile := filepath.Join(recipesDir, ".recipeignore")
	content, err := os.ReadFile(ignoreFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("reading .recipeignore: %w", err)
	}

	data, err := readTemplateData(chezmoiConfigFile)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(".recipeignore").Option("missingkey=zero").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing .recipeignore template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing .recipeignore template: %w", err)
	}

	return parseLines(buf.String()), nil
}

// readTemplateData reads chezmoi's rendered config file and extracts
// the [data] section into a map. Returns an empty map if the file
// does not exist or has no [data] section.
func readTemplateData(chezmoiConfigFile string) (map[string]any, error) {
	data := make(map[string]any)

	content, err := os.ReadFile(chezmoiConfigFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return data, nil
		}
		return nil, fmt.Errorf("reading chezmoi config: %w", err)
	}

	var cfg map[string]any
	if err := toml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parsing chezmoi config: %w", err)
	}

	dataSection, ok := cfg["data"]
	if !ok {
		return data, nil
	}

	dataMap, ok := dataSection.(map[string]any)
	if !ok {
		return data, nil
	}

	return dataMap, nil
}

// parseLines splits the rendered template output into recipe names to ignore.
// Blank lines and lines starting with # are skipped. Whitespace is trimmed.
func parseLines(s string) map[string]bool {
	result := make(map[string]bool)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result[line] = true
	}
	return result
}
