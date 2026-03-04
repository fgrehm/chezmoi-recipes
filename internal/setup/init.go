package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// InitResult reports what RunInit did.
type InitResult struct {
	ConfigSkipped bool
}

// RunInit creates the source directory, writes .chezmoi.toml.tmpl, deploys shared scripts,
// and scaffolds the recipes directory. When force is false and .chezmoi.toml.tmpl already
// exists, the config write is skipped to preserve user customizations.
func RunInit(sourceDir, recipesDir string, force bool) (*InitResult, error) {
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating source directory: %w", err)
	}

	absRecDir, err := filepath.Abs(recipesDir)
	if err != nil {
		absRecDir = recipesDir
	}

	skipped, err := WriteChezmoiConfig(sourceDir, absRecDir, force)
	if err != nil {
		return nil, err
	}

	if err := DeploySharedScripts(sourceDir); err != nil {
		return nil, fmt.Errorf("deploying shared scripts: %w", err)
	}

	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating recipes directory: %w", err)
	}

	return &InitResult{ConfigSkipped: skipped}, nil
}

// chezmoiConfigTemplate is a chezmoi .chezmoi.toml.tmpl that captures user
// data at `chezmoi init` time via promptStringOnce and auto-detects the
// environment using chezmoi template functions. The recipes-dir path is
// injected by chezmoi-recipes init.
const chezmoiConfigTemplate = `{{- /* Auto-detect environment */ -}}
{{- $isContainer := or (stat "/.dockerenv") (stat "/run/.containerenv") (stat "/var/devcontainer") (env "CODESPACES") (env "REMOTE_CONTAINERS") (env "container") | not | not -}}
{{- $isDebian := eq .chezmoi.osRelease.id "debian" -}}
{{- $hasNvidiaGPU := false -}}
{{- if and (not $isContainer) (lookPath "lspci") -}}
{{-   $hasNvidiaGPU = output "lspci" | lower | contains "nvidia" -}}
{{- end -}}

sourceDir = %[1]q

[hooks.read-source-state.pre]
    command = "chezmoi-recipes"
    args = ["overlay", "--quiet", "--recipes-dir", %[2]q]

[data]
    recipesDir = %[2]q
    name = {{ promptStringOnce . "name" "Full name" | quote }}
    email = {{ promptStringOnce . "email" "Email" | quote }}
    isContainer = {{ $isContainer }}
    isDebian = {{ $isDebian }}
    hasNvidiaGPU = {{ $hasNvidiaGPU }}
`

// WriteChezmoiConfig writes .chezmoi.toml.tmpl to the source directory.
// chezmoi processes this template at `chezmoi init` time, prompting for
// user data and auto-detecting the environment. The rendered output
// becomes chezmoi's config file. When force is false and the file already
// exists, the write is skipped and (true, nil) is returned.
func WriteChezmoiConfig(sourceDir, recipesDir string, force bool) (skipped bool, err error) {
	dest := filepath.Join(sourceDir, ".chezmoi.toml.tmpl")
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("checking .chezmoi.toml.tmpl: %w", err)
		}
	}

	content := fmt.Sprintf(chezmoiConfigTemplate, sourceDir, recipesDir)
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("writing .chezmoi.toml.tmpl: %w", err)
	}
	return false, nil
}
