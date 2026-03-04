# Recipe authoring guide

## Rules (follow exactly)

- Every recipe directory must contain `README.md` and a `chezmoi/` subdirectory with at least one file
- Each file in `chezmoi/` has exactly one owning recipe. Two recipes writing the same path causes a conflict error at overlay time
- Name scripts after the tool: `run_once_install-gh.sh`, `run_once_install-neovim.sh`. Generic names like `run_once_install-packages.sh` collide across recipes
- Use `$HOME` in scripts and `{{ .chezmoi.homeDir }}` in templates for home directory paths
- Use `$CHEZMOI_SOURCE_DIR` to reference the chezmoi source directory in scripts
- Keep recipe data inline (in scripts, templates, config files). The `.chezmoidata/` directory is global to chezmoi and cannot be split across recipes
- A recipe can include `chezmoi/.chezmoiignore` with per-recipe ignore entries. During full overlay, all per-recipe `.chezmoiignore` files are merged into the source directory's `.chezmoiignore`. Use this for fine-grained environment filtering within a recipe (e.g., skipping files in containers). For skipping an entire recipe, use `.recipeignore` instead
- Recipes must work independently. If your recipe needs a tool managed by another recipe, check for it within your own script (install it, skip, or fail with a message)
- All `.sh.tmpl` files must use custom template delimiters (`# {{` / `}}`) so shfmt and shellcheck can parse them. Add the directive on line 2 after the shebang

## Recipe structure

A directory under `recipes/` with a `README.md` is a recipe. The directory name is the recipe name.

```
<recipe-name>/
├── README.md            # Required. Documents what the recipe does.
└── chezmoi/             # Required. chezmoi source state fragment.
    ├── .chezmoiscripts/  # Scripts (install, configure, completions)
    ├── dot_*             # dot_ becomes . in target path
    ├── private_dot_*     # 0600/0700 permissions
    └── ...               # Any valid chezmoi source state structure
```

chezmoi naming reference: https://www.chezmoi.io/reference/source-state-attributes/

### Canonical recipe layout (install + config + shell integration)

```
recipes/git/
  README.md
  chezmoi/
    .chezmoiscripts/
      run_once_install-gh.sh.tmpl           # 1. install the tool
      run_onchange_after_completions-gh.sh  # 2. generate completions
    private_dot_config/git/config.tmpl      # 3. config files
    dot_shellrc.d/git.sh                    # 4. shell integration
```

## Script template

Use this as the starting point for every install script:

```bash
#!/bin/env bash
# chezmoi:template:left-delimiter="# {{" right-delimiter="}}"
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

# Skip if already installed
if command -v <tool> &>/dev/null; then
  log_skip "<tool> already installed"
  exit 0
fi

# Conditional sudo
# {{ if ne .chezmoi.username "root" }}
SUDO="sudo"
# {{ else }}
SUDO=""
# {{ end }}

_install() {
  set -e
  log_info "Installing <tool>..."
  "$SUDO" apt-get install -y <package>
}

if ! _install; then
  log_error "Failed to install <tool> (network unavailable?)"
  log_info "Run 'chezmoi apply' again after fixing the issue."
fi
```

What this encodes:

- `# chezmoi:template:left-delimiter=...` on line 2: custom delimiters for shfmt/shellcheck compatibility
- `#!/bin/env bash`: portable shebang that works regardless of bash location
- `source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"`: logging functions (`log_info`, `log_skip`, `log_error`, `run_quiet`), deployed by `chezmoi-recipes init`
- `command -v` guard: check before installing, making scripts safe to re-run
- `$SUDO` variable via template: keeps every line parseable by shfmt (instead of inline `{{ if }}sudo{{ end }}`)
- `_install()` wrapper: see "Resilient install scripts" below

### Resilient install scripts

**Do not use `set -euo pipefail` at the top level of install scripts.** If a download
or network operation fails, a top-level `set -e` causes the script to exit non-zero,
which aborts the entire `chezmoi apply`. Every recipe whose scripts haven't run yet
is silently skipped.

Instead, scope `set -e` inside an `_install()` function:

```bash
_install() {
  set -e
  # ... install commands ...
}

if ! _install; then
  log_error "Failed to install <tool> (network unavailable?)"
  log_info "Run 'chezmoi apply' again after fixing the issue."
fi
```

If `_install` fails, the outer script exits 0 and chezmoi continues applying the
remaining recipes. The error is visible in the output but doesn't block anything.

Note: `run_once_` scripts are tracked by content hash regardless of exit code. Once
run (even on failure), chezmoi won't auto-retry. The user re-runs `chezmoi apply`
after fixing the underlying issue (usually a network or auth problem).

### Script execution order

chezmoi runs scripts in this order:

1. `run_*_before_*` scripts (alphabetically)
2. File deployment (mixed with scripts that lack `before_`/`after_`, alphabetically)
3. `run_*_after_*` scripts (alphabetically)

Use numeric prefixes when ordering matters within a recipe:

```
run_once_00_install-mise.sh              # install mise binary first
run_onchange_after_mise-install.sh.tmpl  # then install mise-managed tools
```

## README template

```markdown
# <recipe-name>

<One sentence describing what this recipe sets up.>

## What it does

- <Action: installs, configures, enables>
- <List packages installed, files deployed, services configured>
- <Mention shell aliases or PATH changes>

## Requirements

- Debian 13 (Trixie)
- <Other: sudo, internet, specific hardware>

## Template variables

| Variable | Description | Source |
|----------|-------------|--------|
| `.name` | Git commit author name | `chezmoi init` prompt |
```

## Common patterns

### apt packages

```bash
#!/bin/env bash
# chezmoi:template:left-delimiter="# {{" right-delimiter="}}"
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

PACKAGES=(ripgrep fzf jq)
MISSING=()

for pkg in "${PACKAGES[@]}"; do
  if ! dpkg -l "$pkg" &>/dev/null; then
    MISSING+=("$pkg")
  fi
done

if [[ ${#MISSING[@]} -eq 0 ]]; then
  log_skip "All packages already installed"
  exit 0
fi

# {{ if ne .chezmoi.username "root" }}
SUDO="sudo"
# {{ else }}
SUDO=""
# {{ end }}

_install() {
  set -e
  log_info "Installing: ${MISSING[*]}"
  "$SUDO" apt-get update -qq
  "$SUDO" apt-get install -y "${MISSING[@]}"
}

if ! _install; then
  log_error "Failed to install packages"
  log_info "Run 'chezmoi apply' again after fixing the issue."
fi
```

### Binary from GitHub releases

```bash
#!/bin/env bash
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

TOOL="mytool"
VERSION="1.2.3"
DEST="$HOME/.local/bin/$TOOL"

if [[ -x "$DEST" ]]; then
  log_skip "$TOOL already installed"
  exit 0
fi

_install() {
  set -e
  log_info "Installing $TOOL v$VERSION..."
  mkdir -p "$(dirname "$DEST")"
  curl -fsSL "https://github.com/org/$TOOL/releases/download/v$VERSION/${TOOL}_${VERSION}_linux_amd64.tar.gz" \
    | tar -xz -C /tmp "$TOOL"
  mv "/tmp/$TOOL" "$DEST"
  chmod +x "$DEST"
}

if ! _install; then
  log_error "Failed to install $TOOL (network unavailable?)"
  log_info "Run 'chezmoi apply' again after fixing network access."
fi
```

### Shell completions

Use `run_onchange_after_` so completions generate after the tool's install script runs:

```bash
#!/bin/env bash
# .chezmoiscripts/run_onchange_after_completions-<tool>.sh
set -euo pipefail

BASH_DIR="$HOME/.local/share/bash-completion/completions"
ZSH_DIR="$HOME/.local/share/zsh/site-functions"
mkdir -p "$BASH_DIR" "$ZSH_DIR"

if command -v gh &>/dev/null; then
  gh completion -s bash > "$BASH_DIR/gh"
  gh completion -s zsh > "$ZSH_DIR/_gh"
fi
```

### Shell module (shellrc.d drop-in)

Place files in `dot_shellrc.d/`. The `shell` recipe's loader sources all `~/.shellrc.d/*.sh` files alphabetically.

```
recipes/git/chezmoi/dot_shellrc.d/git.sh      # git aliases
recipes/mise/chezmoi/dot_shellrc.d/mise.sh    # mise activation
recipes/cuda/chezmoi/dot_shellrc.d/cuda.sh    # CUDA PATH
```

Guard optional dependencies:

```bash
# Only activate mise if installed
if command -v mise &>/dev/null; then
  eval "$(mise activate bash)"
fi
```

### Systemd user service

```
recipes/cartage/
  README.md
  chezmoi/
    .chezmoiscripts/
      run_once_install-cartage.sh                    # install binary
      run_onchange_after_enable-cartage.sh           # enable after .service deployed
    private_dot_config/systemd/user/cartage.service  # unit file
```

Enable script:

```bash
#!/bin/bash
set -euo pipefail
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

log_info "Enabling cartage service..."
systemctl --user daemon-reload
systemctl --user enable --now cartage.service
```

### Desktop application (.desktop file)

```
recipes/brave/
  README.md
  chezmoi/
    .chezmoiscripts/run_once_install-brave.sh
    dot_local/share/applications/chatgpt.desktop.tmpl
```

Reference repo-root icons via `{{ .chezmoi.workingTree }}`:

```ini
Icon={{ .chezmoi.workingTree }}/assets/icons/chatgpt.png
```

### Symlinked config directory

For configs you want to edit directly (live-editable), keep the config files inside the recipe directory but outside `chezmoi/`, and use a `symlink_` file to link to them:

```
recipes/neovim/
  config/nvim/          # actual config files (live-editable)
    init.lua
    lua/plugins/...
  chezmoi/
    private_dot_config/
      symlink_nvim.tmpl # creates symlink ~/.config/nvim -> recipes/neovim/config/nvim
```

File contents (the symlink target):

```
{{ .recipesDir }}/neovim/config/nvim
```

The `.recipesDir` template variable is set by `chezmoi-recipes init` and points to the absolute path of your recipes directory.

### Per-recipe .chezmoiignore

To skip specific files by environment (rather than skipping the entire recipe via `.recipeignore`), add a `chezmoi/.chezmoiignore` to your recipe:

```
recipes/alacritty/
  README.md
  chezmoi/
    .chezmoiignore
    .chezmoiscripts/run_once_install-nerdfonts.sh
    private_dot_config/alacritty/alacritty.toml
```

`chezmoi/.chezmoiignore` contents:

```
{{ if .isContainer }}
private_dot_config/alacritty/
.chezmoiscripts/run_once_install-nerdfonts.sh
{{ end }}
```

During a full overlay, all per-recipe `.chezmoiignore` files are merged into the source directory's `.chezmoiignore`. Template syntax is passed through verbatim for chezmoi to evaluate at apply time.

### Conditional behavior within a recipe

Use chezmoi template conditionals:

```bash
# {{ if not .isContainer }}
# laptop-only logic here
# {{ end }}
```

To skip an entire recipe by environment, use `.recipeignore` instead.

## Common pitfalls

### `chezmoi cd` lands in the overlay output, not your recipes repo

`chezmoi cd` opens a shell in `~/.local/share/chezmoi-recipes/source/` (the
overlay output directory), not in your recipes repository. Edits there are
overwritten on the next `chezmoi apply`.

Always edit files under your `recipes/` directory. Use the path from
`chezmoi-recipes status` or open the file directly in your editor.

### `.chezmoidata/` directories are global

chezmoi's `.chezmoidata/` directory is global: it cannot be split across
recipes. If two recipes each provide a `.chezmoidata/` directory, the overlay
merges them into one and the last writer wins for any conflicting key.

Keep recipe-specific data inline: hard-code values in script bodies, use
`.tmpl` files for template variables, or use `.chezmoidata.toml` files at the
source root with namespaced keys (e.g., `[packages]`, `[completions]`).

### Partial overlay failure leaves untracked files

If `chezmoi apply` is interrupted mid-overlay (network failure, disk error, killed
process), some recipe files may have been written to the source directory without the
state store being updated. On the next run, those files appear as untracked conflicts.

Resolution: remove the conflicting files from the source directory manually, then run
`chezmoi apply` again. The overlay will re-copy them and record ownership correctly.

```bash
rm -rf ~/.local/share/chezmoi-recipes/source/<path/to/conflicting/file>
chezmoi apply
```

Or use `chezmoi-recipes overlay --dry-run` first to see what would be written.

### `.chezmoiignore` entries strip the lifecycle prefix and `.tmpl` suffix

When ignoring a script file, use the bare script name without `run_once_`,
`run_onchange_`, `before_`, `after_`, or `.tmpl`:

```
# chezmoi/.chezmoiignore
{{ if .isContainer }}
# Source: run_once_install-flatpak.sh.tmpl
.chezmoiscripts/install-flatpak.sh
{{ end }}
```

## Recipe sizing (use judgment)

- **Single install script, no config** (heroku, 1password): fine as a standalone recipe
- **Install + config + shell integration** (git, mise): the typical recipe
- **Multiple related tools** (ripgrep + fzf + jq grouped as "dev-tools"): acceptable when always installed together
- **System-level config** (KDE plasma + panel + cedilla + inotify): group by platform concern

## Template variables reference

Set by `.chezmoi.toml.tmpl`, available in all `.tmpl` files:

| Variable | Type | Description |
|----------|------|-------------|
| `.recipesDir` | string | Absolute path to the recipes directory |
| `.name` | string | User's full name |
| `.email` | string | User's email address |
| `.isContainer` | bool | `true` in Docker, devcontainers, Codespaces |
| `.hasNvidiaGPU` | bool | `true` when NVIDIA GPU detected (always `false` in containers) |
| `.chezmoi.sourceDir` | string | chezmoi source directory path |
| `.chezmoi.homeDir` | string | User's home directory |
| `.chezmoi.workingTree` | string | Git working tree root (for referencing repo files) |
| `.chezmoi.username` | string | Current username (use for sudo detection) |
| `.chezmoi.osRelease.id` | string | OS ID (e.g., "debian") |

Full list: https://www.chezmoi.io/reference/templates/variables/
