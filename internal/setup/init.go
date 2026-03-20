package setup

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fgrehm/chezmoi-recipes/internal/overlay"
	"github.com/fgrehm/chezmoi-recipes/internal/paths"
)

// InitResult reports what RunInit did.
type InitResult struct {
	ConfigSkipped bool
}

// RunInit sets up the .chezmoiroot architecture: creates home/ and recipes/
// directories, writes .chezmoiroot, appends compiled-home/ to .gitignore,
// writes .chezmoi.toml.tmpl into home/, and runs an initial overlay to
// populate compiled-home/ so that `chezmoi init` can find the config template.
// When force is false and .chezmoi.toml.tmpl already exists in home/, the
// config write is skipped to preserve user customizations.
func RunInit(repoRoot, recipesDir string, force bool) (*InitResult, error) {
	homeDir := paths.HomeDir(repoRoot)
	compiledHome := paths.CompiledHomeDir(repoRoot)

	// Create home/ directory.
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating home directory: %w", err)
	}

	// Write .chezmoiroot at repo root.
	chezmoirootPath := filepath.Join(repoRoot, ".chezmoiroot")
	if err := os.WriteFile(chezmoirootPath, []byte("compiled-home\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing .chezmoiroot: %w", err)
	}

	// Append compiled-home/ to .gitignore (idempotent).
	if err := ensureGitignoreEntry(repoRoot, "compiled-home/"); err != nil {
		return nil, fmt.Errorf("updating .gitignore: %w", err)
	}

	// Write .chezmoi.toml.tmpl into home/.
	absRecDir, err := filepath.Abs(recipesDir)
	if err != nil {
		absRecDir = recipesDir
	}
	skipped, err := WriteChezmoiConfig(homeDir, repoRoot, absRecDir, force)
	if err != nil {
		return nil, err
	}

	// Create recipes/ directory.
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating recipes directory: %w", err)
	}

	// Write .editorconfig (2-space indent for shell scripts).
	if err := writeIfMissing(filepath.Join(repoRoot, ".editorconfig"), editorConfig); err != nil {
		return nil, fmt.Errorf("writing .editorconfig: %w", err)
	}

	// Write .shellcheckrc (suppress chezmoi template noise).
	if err := writeIfMissing(filepath.Join(repoRoot, ".shellcheckrc"), shellcheckRC); err != nil {
		return nil, fmt.Errorf("writing .shellcheckrc: %w", err)
	}

	// Write README.md if none exists.
	if err := writeIfMissing(filepath.Join(repoRoot, "README.md"), readmeTemplate); err != nil {
		return nil, fmt.Errorf("writing README.md: %w", err)
	}

	// Run initial overlay: copy home/ -> compiled-home/ so chezmoi init finds
	// the config template.
	if err := os.MkdirAll(compiledHome, 0o755); err != nil {
		return nil, fmt.Errorf("creating compiled-home directory: %w", err)
	}
	if _, err := overlay.CopyTree(homeDir, compiledHome); err != nil {
		return nil, fmt.Errorf("initial overlay (home -> compiled-home): %w", err)
	}

	return &InitResult{ConfigSkipped: skipped}, nil
}

// writeIfMissing writes content to path if the file does not already exist.
func writeIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

const editorConfig = `root = true

[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8

[*.{sh,bash,zsh}]
indent_style = space
indent_size = 2

[*.sh.tmpl]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
`

const shellcheckRC = `# Sourced files are checked independently; don't warn about missing includes.
disable=SC1091
# Allow chezmoi template directives in shebang position.
disable=SC1128
# chezmoi template conditionals look like constant comparisons.
disable=SC2050
# Functions defined inside _install() wrappers appear unreachable.
disable=SC2317
`

const readmeTemplate = `# dotfiles

Managed with [chezmoi](https://www.chezmoi.io/) and [chezmoi-recipes](https://github.com/fgrehm/chezmoi-recipes).

## Quick start

` + "```bash" + `
# Install chezmoi and chezmoi-recipes, then apply
chezmoi init --source .
chezmoi apply
` + "```" + `
`

// ensureGitignoreEntry appends entry to .gitignore if not already present.
func ensureGitignoreEntry(repoRoot, entry string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	if data, err := os.ReadFile(gitignorePath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == entry {
				return nil
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, entry); err != nil {
		return err
	}
	return nil
}

// chezmoiConfigTemplate is a chezmoi .chezmoi.toml.tmpl that captures user
// data at `chezmoi init` time via promptStringOnce and auto-detects the
// environment using chezmoi template functions. The recipes-dir relative path
// is injected by chezmoi-recipes init; {{ .chezmoi.workingTree }} resolves it
// portably so the config works when cloned to a different location.
//
// Guard hooks block commands that would write to compiled-home/ (a generated
// directory). Users should edit files in home/ or recipes/ instead.
const chezmoiConfigTemplate = `{{- /* Auto-detect environment */ -}}
{{- $isContainer := or (stat "/.dockerenv") (stat "/run/.containerenv") (stat "/var/devcontainer") (env "CODESPACES") (env "REMOTE_CONTAINERS") (env "container") | not | not -}}
{{- $isDebian := eq .chezmoi.osRelease.id "debian" -}}
{{- $hasNvidiaGPU := false -}}
{{- if and (not $isContainer) (lookPath "lspci") -}}
{{-   $hasNvidiaGPU = output "lspci" | lower | contains "nvidia" -}}
{{- end -}}

sourceDir = "{{ .chezmoi.workingTree }}"

[hooks.read-source-state.pre]
    command = "chezmoi-recipes"
    args = ["overlay", "--recipes-dir", "{{ .chezmoi.workingTree }}/%s"]

[hooks.add.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi add (compiled-home/ is generated)' >&2; exit 1"]

[hooks.edit.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi edit (compiled-home/ is generated)' >&2; exit 1"]

[hooks.re-add.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi re-add (compiled-home/ is generated)' >&2; exit 1"]

[hooks.merge.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi merge (compiled-home/ is generated)' >&2; exit 1"]

[hooks.chattr.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi chattr (compiled-home/ is generated)' >&2; exit 1"]

[hooks.import.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi import (compiled-home/ is generated)' >&2; exit 1"]

[hooks.forget.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi forget (compiled-home/ is generated)' >&2; exit 1"]

[hooks.destroy.pre]
    command = "sh"
    args = ["-c", "echo 'Error: use home/ or recipes/ instead of chezmoi destroy (compiled-home/ is generated)' >&2; exit 1"]

[diff]
    pager = "cat"

[data]
    recipesDir = "{{ .chezmoi.workingTree }}/%s"
    name = {{ promptStringOnce . "name" "Full name" | quote }}
    email = {{ promptStringOnce . "email" "Email" | quote }}
    isContainer = {{ $isContainer }}
    isDebian = {{ $isDebian }}
    hasNvidiaGPU = {{ $hasNvidiaGPU }}
`

// WriteChezmoiConfig writes .chezmoi.toml.tmpl to the given directory (home/).
// chezmoi processes this template at `chezmoi init` time, prompting for
// user data and auto-detecting the environment. When force is false and the
// file already exists, the write is skipped and (true, nil) is returned.
//
// repoRoot is the dotfiles repo root. recipesDir is the absolute path to the
// recipes directory. The relative path from repoRoot to recipesDir is embedded
// in the template as {{ .chezmoi.workingTree }}/<relPath>, so the generated
// config works when the repo is cloned to a different location.
func WriteChezmoiConfig(homeDir, repoRoot, recipesDir string, force bool) (skipped bool, err error) {
	dest := filepath.Join(homeDir, ".chezmoi.toml.tmpl")
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("checking .chezmoi.toml.tmpl: %w", err)
		}
	}

	relRecipesDir, err := filepath.Rel(repoRoot, recipesDir)
	if err != nil {
		return false, fmt.Errorf("computing relative recipes path: %w", err)
	}

	content := fmt.Sprintf(chezmoiConfigTemplate, relRecipesDir, relRecipesDir)
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("writing .chezmoi.toml.tmpl: %w", err)
	}
	return false, nil
}
