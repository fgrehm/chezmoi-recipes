// Package setup implements chezmoi-recipes initialization: writing the
// .chezmoi.toml.tmpl config template, deploying shared script utilities, and
// managing the .chezmoiignore merge.
package setup

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed ui.bash
var uiBash []byte

// DeploySharedScripts writes shared script utilities into the chezmoi source
// directory so that recipe scripts can source them at runtime.
// It also ensures .chezmoiignore contains "scripts/" so chezmoi does not
// try to deploy the helper directory to the home directory.
func DeploySharedScripts(sourceDir string) error {
	scriptsDir := filepath.Join(sourceDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("creating scripts directory: %w", err)
	}

	dest := filepath.Join(scriptsDir, "ui.bash")
	if err := os.WriteFile(dest, uiBash, 0o644); err != nil {
		return fmt.Errorf("writing ui.bash: %w", err)
	}

	if err := EnsureChezmoiIgnore(sourceDir, []string{"scripts/"}); err != nil {
		return fmt.Errorf("updating .chezmoiignore: %w", err)
	}

	return nil
}

// EnsureChezmoiIgnore ensures the given entries exist in the .chezmoiignore
// file inside sourceDir. Entries that are already present are not duplicated.
func EnsureChezmoiIgnore(sourceDir string, entries []string) error {
	ignorePath := filepath.Join(sourceDir, ".chezmoiignore")

	existing := make(map[string]bool)
	if data, err := os.ReadFile(ignorePath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			existing[strings.TrimSpace(scanner.Text())] = true
		}
	}

	var toAdd []string
	for _, entry := range entries {
		if !existing[entry] {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, entry := range toAdd {
		if _, err := fmt.Fprintln(f, entry); err != nil {
			return err
		}
	}

	return nil
}
