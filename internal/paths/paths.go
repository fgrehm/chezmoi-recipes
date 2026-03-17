// Package paths provides path helpers for chezmoi-recipes: repo-relative paths
// (CompiledHomeDir, HomeDir) and XDG-compliant runtime paths (state file,
// chezmoi config file).
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultStateFile returns the default path for the chezmoi-recipes state file.
// Uses $XDG_DATA_HOME/chezmoi-recipes/state.json, falling back to ~/.local/share/chezmoi-recipes/state.json.
func DefaultStateFile() (string, error) {
	base, err := dataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "state.json"), nil
}

// ChezmoiConfigFile returns the path to chezmoi's config file.
// Uses $XDG_CONFIG_HOME/chezmoi/chezmoi.toml, falling back to ~/.config/chezmoi/chezmoi.toml.
func ChezmoiConfigFile() (string, error) {
	dir, err := chezmoiConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "chezmoi.toml"), nil
}

// CompiledHomeDir returns the compiled-home directory path within a repo root.
// This is where the overlay writes merged files from home/ and recipes/.
func CompiledHomeDir(repoRoot string) string {
	return filepath.Join(repoRoot, "compiled-home")
}

// HomeDir returns the home directory path within a repo root.
// This is where users place freeform chezmoi source files.
func HomeDir(repoRoot string) string {
	return filepath.Join(repoRoot, "home")
}

func chezmoiConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "chezmoi"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".config", "chezmoi"), nil
}

func dataHome() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "chezmoi-recipes"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "chezmoi-recipes"), nil
}
