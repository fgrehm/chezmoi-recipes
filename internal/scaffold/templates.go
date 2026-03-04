package scaffold

const readmeTmpl = `# <name>

TODO: One sentence describing what this recipe sets up.

## What it does

- Installs <name>
- Deploys <name> configuration to ~/.config/<name>/
- Adds shell aliases via ~/.shellrc.d/

## Requirements

- Debian 13 (Trixie)

## Template variables

| Variable | Description | Source |
|----------|-------------|--------|
| ` + "`.name`" + ` | User's full name | ` + "`chezmoi init`" + ` prompt |
| ` + "`.email`" + ` | User's email address | ` + "`chezmoi init`" + ` prompt |
| ` + "`.isContainer`" + ` | true in Docker/devcontainers | auto-detected |
`

const installScriptTmpl = `#!/bin/env bash
# chezmoi:template:left-delimiter="# {{" right-delimiter="}}"
#
# chezmoi script naming cheat sheet:
#
#   Trigger        | Meaning
#   -------------- | --------------------------------------------------
#   run_once_      | runs once, tracked by content hash (re-runs on edit)
#   run_onchange_  | same as run_once_ (alias)
#   run_always_    | runs on every ` + "`chezmoi apply`" + `
#
#   Ordering       | Meaning
#   -------------- | --------------------------------------------------
#   before_        | runs before chezmoi deploys files
#   after_         | runs after chezmoi deploys files
#   (neither)      | runs interleaved with file deployment, alphabetically
#
#   Use numeric prefixes when order between scripts matters:
#     run_once_00_install-foo.sh   (runs first)
#     run_once_01_configure-foo.sh (runs second)
#
# The .tmpl suffix tells chezmoi to evaluate template directives before
# running the script. The custom delimiters on line 2 keep the file valid
# bash so shfmt and shellcheck still work. Template lines start with "# {{".
#
# Docs: https://www.chezmoi.io/user-guide/use-scripts-to-perform-actions/

# Shared logging utilities deployed by ` + "`chezmoi-recipes init`" + `.
# Available functions: log_info, log_skip, log_error, run_quiet
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

# --- Idempotent guard ---
# Always check before installing so the script is safe to re-run
# and ` + "`chezmoi apply`" + ` stays fast when the tool is already present.
if command -v <name> &>/dev/null; then
  log_skip "<name> already installed"
  exit 0
fi

# --- Conditional sudo ---
# Using a template variable keeps every line parseable by shfmt
# (instead of inline {{ if }}sudo{{ end }}).
# {{ if ne .chezmoi.username "root" }}
SUDO="sudo"
# {{ else }}
SUDO=""
# {{ end }}

# --- Install ---
# Wrap in _install() so a network or download failure exits this function
# without aborting the entire ` + "`chezmoi apply`" + ` (which would skip all later recipes).
_install() {
  set -e

  log_info "Installing <name>..."

  # Option A: apt package
  # run_quiet "$SUDO" apt-get update -qq
  # run_quiet "$SUDO" apt-get install -y <name>

  # Option B: binary from GitHub releases
  # VERSION="1.0.0"
  # curl -fsSL "https://github.com/org/<name>/releases/download/v${VERSION}/<name>_linux_amd64.tar.gz" \
  #   | tar -xz -C /tmp "<name>"
  # "$SUDO" install -m 755 "/tmp/<name>" /usr/local/bin/<name>

  echo "TODO: add install commands for <name>"
}

if ! _install; then
  log_error "Failed to install <name> (network unavailable?)"
  log_info "Run 'chezmoi apply' again after fixing the issue."
fi
`

const chezmoiIgnoreTmpl = `# Per-recipe .chezmoiignore
#
# Entries here are merged into the source directory's .chezmoiignore during
# a full overlay. chezmoi evaluates the Go template conditionals at apply time.
#
# Use this to skip specific files by environment within a recipe.
# To skip the entire recipe instead, use .recipeignore at the recipe root.
#
# Syntax: chezmoi ignore patterns with Go template conditionals.
# Docs: https://www.chezmoi.io/reference/special-files/chezmoiignore/

# Example: skip GUI config in containers
# {{ if .isContainer }}
# private_dot_config/<name>/
# .chezmoiscripts/run_once_install-<name>.sh.tmpl
# {{ end }}
`

const configTmpl = `# chezmoi naming conventions (how this path maps to the target):
#
#   Source path:  private_dot_config/<name>/config.toml.tmpl
#   Target path:  ~/.config/<name>/config.toml  (mode 0600)
#
#   Prefix     | Effect
#   ---------- | -----------------------------------------
#   dot_       | adds a leading dot  (dot_bashrc -> .bashrc)
#   private_   | 0600 files, 0700 directories
#   readonly_  | 0444 files, 0555 directories
#   symlink_   | creates a symlink (file contents = target path)
#   empty_     | ensures file exists even if empty
#   modify_    | script that modifies an existing file
#
# The .tmpl suffix tells chezmoi to process Go template directives
# before writing the file. Remove .tmpl if no templating is needed.
#
# Full reference: https://www.chezmoi.io/reference/source-state-attributes/

# Available template variables (set by .chezmoi.toml.tmpl):
#   {{ .name }}              - user's full name
#   {{ .email }}             - user's email
#   {{ .isContainer }}      - true in Docker/devcontainers
#   {{ .chezmoi.homeDir }}   - home directory path
#   {{ .chezmoi.sourceDir }} - chezmoi source directory
#   {{ .chezmoi.username }}  - current OS username

# Example config for <name>:
# user = {{ .name | quote }}
`

const shellModuleTmpl = `# shellcheck shell=bash
#
# Shell integration for <name>.
# Placed in dot_shellrc.d/ so the shell recipe's loader sources it automatically.
#
# Path mapping: dot_shellrc.d/<name>.sh -> ~/.shellrc.d/<name>.sh
#
# No .tmpl suffix here because this file has no template directives.
# If you need chezmoi variables, rename to <name>.sh.tmpl and add the
# custom delimiter directive after the shebang comment.

# Guard: only run if <name> is installed.
# Prevents errors on first chezmoi apply before install scripts have run.
if ! command -v <name> &>/dev/null; then
  return 0
fi

# --- Aliases ---
# alias <name>s='<name> status'

# --- Environment ---
# export <NAME>_HOME="$HOME/.<name>"

# --- Activation ---
# eval "$(<name> init bash)"
`
