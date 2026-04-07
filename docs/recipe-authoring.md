# Recipe authoring guide

## Rules (follow exactly)

- Every recipe directory must contain `README.md` and a `chezmoi/` subdirectory with at least one file
- Each file in `chezmoi/` has exactly one owning recipe. Two recipes writing the same path causes a conflict error at overlay time
- Name scripts after the tool: `run_once_install-gh.sh`, `run_once_install-neovim.sh`. Generic names like `run_once_install-packages.sh` collide across recipes
- Use `private_dot_config` for all files under `.config`. Never use `dot_config` for the `.config` directory. chezmoi rejects an overlay where two recipes disagree on privacy for the same target directory with `inconsistent state`
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
    ├── .chezmoiexternals/  # External binaries/files (GitHub releases, archives)
    ├── .chezmoiscripts/    # Scripts (install, configure, completions)
    ├── dot_*               # dot_ becomes . in target path
    ├── private_dot_*       # 0600/0700 permissions
    └── ...                 # Any valid chezmoi source state structure
```

chezmoi naming reference: https://www.chezmoi.io/reference/source-state-attributes/

### Canonical recipe layout (install + config + shell integration)

```
recipes/git/
  README.md
  chezmoi/
    .chezmoiexternals/
      gh.toml                               # 1. download binary from GitHub
    .chezmoiscripts/
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
  $SUDO apt-get install -y <package>
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
- `$SUDO` variable via template: keeps every line parseable by shfmt (instead of inline `{{ if }}sudo{{ end }}`). Always use `$SUDO` unquoted (see "$SUDO must be unquoted" in Common Pitfalls)
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

**Trade-off:** chezmoi only records a `run_once_` script as "done" on exit 0. A script
that exits non-zero is retried on the next `chezmoi apply`. The `_install()` wrapper
always exits 0, so chezmoi marks it done even on failure. The user must run
`chezmoi state delete-bucket --bucket=scriptState` to force a retry. Use the wrapper
for optional tools where a transient failure should not block other recipes. For
critical dependencies, use `set -eo pipefail` at the top level instead (see
"Critical vs optional install scripts" below).

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
  $SUDO apt-get update -qq
  $SUDO apt-get install -y "${MISSING[@]}"
}

if ! _install; then
  log_error "Failed to install packages"
  log_info "Run 'chezmoi apply' again after fixing the issue."
fi
```

### Binary from GitHub releases

Use chezmoi's [`.chezmoiexternals/`](https://www.chezmoi.io/reference/special-directories/chezmoiexternals/) directory instead of a shell install script. Place a `<tool>.toml` file inside `chezmoi/.chezmoiexternals/` in your recipe. chezmoi-recipes overlays it into `compiled-home/.chezmoiexternals/`, and chezmoi downloads and installs the binary at apply time.

Each file in `.chezmoiexternals/` is always rendered as a template (no `.tmpl` extension needed), so chezmoi template functions work directly.

**Always pin the version explicitly** using a `$version` variable at the top of the file. This makes installs reproducible and the version string is an obvious bump target (see `make check-versions`):

```toml
# chezmoi/.chezmoiexternals/diffnav.toml
{{- $version := "0.7.2" -}}
{{- $arch := .chezmoi.arch -}}
{{- if eq $arch "amd64" -}}{{- $arch = "x86_64" -}}{{- end -}}
[".local/bin/diffnav"]
  type = "archive-file"
  url = "https://github.com/dlvhdr/diffnav/releases/download/v{{ $version }}/diffnav_Linux_{{ $arch }}.tar.gz"
  executable = true
  path = "diffnav"
```

Do not use `gitHubLatestReleaseAssetURL` or `gitHubLatestRelease`. These make GitHub API calls at template render time, causing rate-limit failures in CI and unit tests (even with `--exclude=externals`). Pinned `/releases/download/<version>/` URLs avoid the API entirely.

#### Arch translation

Two naming schemes appear in the wild:

- **GOARCH style** (`amd64`/`arm64`): used by Go-built tools. `.chezmoi.arch` passes through directly, no translation needed.
- **GNU/uname style** (`x86_64`/`aarch64`): used by Rust/C tools. Requires translation:

```
{{- $arch := .chezmoi.arch -}}
{{- if eq $arch "amd64" -}}{{- $arch = "x86_64" -}}{{- else if eq $arch "arm64" -}}{{- $arch = "aarch64" -}}{{- end -}}
```

Check the project's release filenames to determine which scheme it uses.

#### Archive with version-prefixed directory

When the archive has a version-prefixed top-level directory (common in Rust/Go releases), use `$version` in the `path` field to match:

```toml
# chezmoi/.chezmoiexternals/delta.toml
{{- $version := "0.19.1" -}}
{{- $arch := .chezmoi.arch -}}
{{- if eq $arch "amd64" -}}{{- $arch = "x86_64" -}}{{- else if eq $arch "arm64" -}}{{- $arch = "aarch64" -}}{{- end -}}
[".local/bin/delta"]
  type = "archive-file"
  url = "https://github.com/dandavison/delta/releases/download/{{ $version }}/delta-{{ $version }}-{{ $arch }}-unknown-linux-gnu.tar.gz"
  path = "delta-{{ $version }}-{{ $arch }}-unknown-linux-gnu/delta"
  executable = true
```

#### Direct binary download (no archive)

Some tools release a pre-built binary rather than an archive. Use `type = "file"` instead of `type = "archive-file"`:

```toml
# chezmoi/.chezmoiexternals/ttyd.toml
{{- $version := "1.7.7" -}}
{{- $arch := .chezmoi.arch -}}
{{- if eq $arch "amd64" -}}{{- $arch = "x86_64" -}}{{- else if eq $arch "arm64" -}}{{- $arch = "aarch64" -}}{{- end -}}
[".local/bin/ttyd"]
  type = "file"
  url = "https://github.com/tsl0922/ttyd/releases/download/{{ $version }}/ttyd.{{ $arch }}"
  executable = true
```

#### Tag prefix convention

Some projects tag releases as `v1.2.3`, others use `1.2.3`. Either include the `v` literally in the URL (e.g., `download/v{{ $version }}/` with `$version := "1.2.3"`) or include it in the version variable itself (e.g., `$version := "v1.2.3"` with `download/{{ $version }}/`). Check the repository's GitHub releases page and match the pattern it uses.

#### Multiple recipes, no conflicts

Multiple recipes can each have their own `.chezmoiexternals/*.toml` files. Since each file has a unique name (the tool name), there are no overlay conflicts.

Use a shell install script only when `.chezmoiexternals/` is not sufficient: apt packages or tools that need post-install setup.

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

During a full overlay, all per-recipe `.chezmoiignore` files are merged into `compiled-home/.chezmoiignore`. Template syntax is passed through verbatim for chezmoi to evaluate at apply time.

### Conditional behavior within a recipe

Use chezmoi template conditionals:

```bash
# {{ if not .isContainer }}
# laptop-only logic here
# {{ end }}
```

To skip an entire recipe by environment, use `.recipeignore` instead.

## Recipe ordering and dependencies

### The shell/base recipe comes first

Recipes that drop files in `dot_shellrc.d/` depend on a shell recipe that sets up the loader (`dot_shellrc` sourcing `~/.shellrc.d/*.sh`). Without the shell recipe, those fragments are never sourced.

If you have a shell/base recipe, make sure it exists before creating recipes that ship shell fragments. chezmoi-recipes does not enforce ordering between recipes, but the shell recipe is foundational. Document this in your repo's README.

### `lookPath` evaluates before scripts run

chezmoi template functions like `lookPath` evaluate at template render time, before any install scripts run. This means `{{ if lookPath "diffnav" }}` won't detect a tool installed by a script in the same `chezmoi apply`.

Options for conditional config based on tool presence:

1. **Runtime check in a shellrc.d fragment.** Instead of a template conditional in a config file, use a shell fragment that runs `git config --global ...` at shell init time. The tool will be present by then.

2. **Always include the config and accept the no-op.** Most tools ignore config for features they don't use. A git pager setting for a tool that isn't installed just falls back to the default.

3. **Defer to a second apply.** Run `chezmoi apply` once to install tools, then again to pick up the new `lookPath` results. This is manual but works for rare cases.

The first option (runtime shell check) is the most reliable pattern for cross-tool integration.

### Critical vs optional install scripts

The `_install()` wrapper pattern (see "Resilient install scripts" above) is correct for optional tools like ripgrep or bat. If they fail to install, the user can still work.

**Do not use the wrapper for critical dependencies.** If oh-my-zsh fails to install, zsh is completely broken because `dot_zshrc` sources `$ZSH/oh-my-zsh.sh` unconditionally. Core infrastructure scripts should fail hard so the user knows immediately:

```bash
#!/usr/bin/env bash
# Critical dependency: hard fail on error
set -euo pipefail
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

if command -v zsh &>/dev/null && [[ -d "$HOME/.oh-my-zsh" ]]; then
  log_skip "oh-my-zsh already installed"
  exit 0
fi

log_info "Installing oh-my-zsh..."
# No _install() wrapper: if this fails, chezmoi apply stops.
# That's intentional: zsh config depends on oh-my-zsh.
wget -qO- https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh | sh -s -- --unattended
```

Use this pattern when other files in the recipe (or other recipes) depend on the tool being present. The graceful `_install()` pattern is for tools where failure is inconvenient but not breaking.

## Common pitfalls

### Vim modeline must go at the bottom of `.chezmoiexternals/*.toml`

Adding `# vim: ft=toml.gotmpl` as the first line of a `.chezmoiexternals/*.toml` file breaks chezmoi silently. The `{{- $version := "..." -}}` trim markers eat the newlines before and after the action, merging the modeline comment with the table header onto a single line:

```
# vim: ft=toml.gotmpl[".local/bin/clotilde"]
```

TOML treats the entire line as a comment, so the table definition is dropped. chezmoi then fails with: `type mismatch for chezmoi.External: expected table but found string`.

Put the modeline on the last line instead. Vim reads modelines from the last few lines too (controlled by the `modelines` option, default 5).

### `dot_config` and `private_dot_config` cannot coexist for the same target

chezmoi maps both `dot_config` and `private_dot_config` to the target directory `.config`, but with different permissions. If two recipes contribute to `.config` with different privacy prefixes, chezmoi refuses to apply:

```
chezmoi: .config: inconsistent state (...dot_config, ...private_dot_config)
```

Always use `private_dot_config` for files under `.config`. The `.config` directory holds user application state and is private by design. This applies to every recipe, every time -- there is no environment where mixing is acceptable.

### `$SUDO` must be unquoted

When `SUDO` is empty (running as root), `"$SUDO" apt-get install ...` becomes `"" apt-get install ...`. Bash tries to execute an empty string as a command and fails with `bash: : command not found`.

Use `$SUDO` unquoted:

```bash
$SUDO apt-get install -y <package>
```

This is safe because `SUDO` is either `"sudo"` (single word, no splitting) or `""` (expands to nothing when unquoted). The template conditional guarantees no other values.

### `chezmoi cd` goes to the repo root, not `recipes/`

`chezmoi cd` opens a shell in the working tree root (your dotfiles repo), not in `recipes/`. This is correct behavior: the working tree is the repo, not `compiled-home/`. Navigate from there:

```bash
chezmoi cd          # lands in ~/dotfiles/
cd recipes/neovim   # then navigate to the recipe you want
```

### `.chezmoidata/` directories are global

chezmoi's `.chezmoidata/` directory is global: it cannot be split across
recipes. If two recipes each provide a `.chezmoidata/` directory, the overlay
merges them into one and the last writer wins for any conflicting key.

Keep recipe-specific data inline: hard-code values in script bodies, use
`.tmpl` files for template variables, or use `.chezmoidata.toml` files at the
source root with namespaced keys (e.g., `[packages]`, `[completions]`).

### Partial overlay failure

If `chezmoi apply` is interrupted mid-overlay (disk error, killed process), `compiled-home/` may be in an incomplete state. This is self-healing: the next overlay clears and rebuilds `compiled-home/` from scratch. Just run `chezmoi apply` again.

Use `chezmoi-recipes overlay --dry-run` to preview what the overlay would write.

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

### Symlink pattern for large or frequently-edited configs

For tools with large config directories (like Neovim), consider keeping the config at the repo root and pointing `~/` at it via chezmoi's native [`symlink_` source attribute](https://www.chezmoi.io/reference/source-state-attributes/). This lets you edit config files and have changes take effect immediately, without `chezmoi apply`.

**Layout:**

```
recipes/neovim/
  README.md
  chezmoi/                   <- overlay input (scripts, small configs)
    .chezmoiscripts/
      ...
  config/
    nvim/                    <- the actual config, tracked in git
      init.lua
      lua/
        ...
home/
  private_dot_config/
    symlink_nvim.tmpl        <- chezmoi symlink descriptor
```

The `config/` subdirectory sits alongside `chezmoi/` inside the recipe. It is not overlay input - the overlay only reads `chezmoi/`. It's just tracked files that `~/` points at directly.

**`home/private_dot_config/symlink_nvim.tmpl`:**

```
{{ .chezmoi.workingTree }}/recipes/neovim/config/nvim
```

chezmoi reads this file, sees the `symlink_` prefix, and creates `~/.config/nvim` as a symlink pointing at `<repo-root>/recipes/neovim/config/nvim`. The template expands `.chezmoi.workingTree` to the dotfiles repo root, so it works on any machine regardless of where the repo is cloned.

After the initial `chezmoi apply`, edits to `config/nvim/` take effect immediately. No re-apply needed.

This pattern works well when the config is large, changes frequently, or you want to edit it directly without chezmoi's naming conventions (`private_dot_config/`, `.tmpl`, etc.) on every file. Keeping `config/` inside the recipe means everything for that tool lives in one place, and removing the recipe removes the config too.

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
