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
func RunInit(repoRoot, recipesDir, repoURL string, force bool) (*InitResult, error) {
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
	skipped, err := WriteChezmoiConfig(homeDir, repoRoot, recipesDir, force)
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

	// Write install.sh if a repo URL was provided.
	if repoURL != "" {
		relRecDir, err := filepath.Rel(repoRoot, recipesDir)
		if err != nil {
			return nil, fmt.Errorf("computing relative recipes path from %q to %q: %w", repoRoot, recipesDir, err)
		}
		if relRecDir == ".." || strings.HasPrefix(relRecDir, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("recipes directory %q is outside repo root %q", recipesDir, repoRoot)
		}
		script := generateInstallScript(repoURL, relRecDir)
		installPath := filepath.Join(repoRoot, "install.sh")
		_, statErr := os.Lstat(installPath)
		newFile := errors.Is(statErr, os.ErrNotExist)
		if err := writeIfMissing(installPath, script); err != nil {
			return nil, fmt.Errorf("writing install.sh: %w", err)
		}
		if newFile {
			if err := os.Chmod(installPath, 0o755); err != nil {
				return nil, fmt.Errorf("setting install.sh permissions: %w", err)
			}
		}
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
// Uses Lstat to detect symlinks without following them. Returns an error if
// the path exists but is not a regular file (e.g. a directory or symlink).
func writeIfMissing(path, content string) error {
	fi, err := os.Lstat(path)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s exists but is not a regular file", path)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
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

// generateInstallScript returns an install.sh with the repo URL and recipes
// directory relative path baked in. The URL is embedded inside single quotes
// with internal single quotes escaped so it is safe regardless of URL content.
func generateInstallScript(repoURL, recipesRelDir string) string {
	return fmt.Sprintf(installScriptTemplate, shellEscapeSingleQuoted(repoURL), shellEscapeSingleQuoted(recipesRelDir))
}

// shellEscapeSingleQuoted escapes s for embedding inside single quotes in
// shell by replacing each ' with '\''.
func shellEscapeSingleQuoted(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

const installScriptTemplate = `#!/bin/sh
# One-liner dotfiles setup: installs chezmoi + chezmoi-recipes, clones this
# repo, builds the overlay, initializes chezmoi, and applies.
#
# Usage:
#   sh -c "$(curl -fsSL <raw-url-to-this-file>)"
#
#   # Non-interactive (provide prompt values)
#   sh -c "$(curl -fsSL <raw-url-to-this-file>)" -- \
#     --promptString "Full name=Your Name" \
#     --promptString "Email=you@example.com"
#
# Arguments (e.g. --promptString) are forwarded to chezmoi init.
# Do not pass --apply; the script always runs chezmoi apply as the final step.

set -eu

BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

_log() { printf '\033[1;34m==> %%s\033[0m\n' "$*"; }
_die() { printf '\033[1;31merror: %%s\033[0m\n' "$*" >&2; exit 1; }

[ "$(uname -s)" = "Linux" ] || _die "only Linux is supported"

for _arg in "$@"; do
  [ "$_arg" != "--apply" ] || _die "--apply is not needed: this script always runs chezmoi apply"
done
unset _arg

case "$(uname -m)" in
  x86_64)        ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) _die "unsupported architecture: $(uname -m)" ;;
esac

mkdir -p "$BIN_DIR"
export PATH="$BIN_DIR:$PATH"

# Install chezmoi
if ! command -v chezmoi >/dev/null 2>&1; then
  _log "Installing chezmoi"
  BINDIR="$BIN_DIR" sh -c "$(curl -fsLS get.chezmoi.io)"
fi

# Install chezmoi-recipes
if ! command -v chezmoi-recipes >/dev/null 2>&1; then
  _log "Installing chezmoi-recipes"
  curl -fsSL "https://github.com/fgrehm/chezmoi-recipes/releases/latest/download/chezmoi-recipes_linux_${ARCH}.tar.gz" \
    | tar xz -C "$BIN_DIR"
fi

REPO_URL='%s'
SOURCE_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/chezmoi"

# Clone dotfiles
if [ ! -d "$SOURCE_DIR/.git" ]; then
  _log "Cloning $REPO_URL"
  git clone "$REPO_URL" "$SOURCE_DIR"
else
  _log "Dotfiles already cloned at $SOURCE_DIR"
fi

# Build compiled-home/ so chezmoi can find the config template
_log "Building overlay"
chezmoi-recipes overlay --recipes-dir "$SOURCE_DIR"/'%s'

# Initialize chezmoi (processes config template, prompts for user data)
_log "Initializing chezmoi"
chezmoi init --source "$SOURCE_DIR" "$@"

# Apply dotfiles
_log "Applying dotfiles"
exec chezmoi apply
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
	if fi, err := os.Lstat(dest); err == nil {
		if !fi.Mode().IsRegular() {
			return false, fmt.Errorf(".chezmoi.toml.tmpl exists but is not a regular file")
		}
		if !force {
			return true, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("checking .chezmoi.toml.tmpl: %w", err)
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
