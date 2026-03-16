# Migrating from a flat chezmoi setup

How to convert an existing chezmoi dotfiles repo into chezmoi-recipes recipes.

## Before you start

### What changes

- Your flat chezmoi source directory (`dot_`, `private_`, `.chezmoiscripts/`) splits into per-tool recipe directories under `recipes/`.
- `.chezmoiignore` environment filtering becomes `.recipeignore` (skip entire recipes instead of listing individual files).
- Centralized package lists (`.chezmoidata/packages.yaml`) become per-recipe install scripts.
- External downloads (`.chezmoiexternal.toml.tmpl`) become per-recipe install scripts.
- Shell completions move from a centralized generation script to per-recipe scripts.

### What stays the same

- chezmoi is still the engine. Templates, scripts, encryption, `chezmoi apply` all work the same way.
- Your `.chezmoi.toml.tmpl` (user data, environment detection) carries over.
- chezmoi naming conventions (`dot_`, `private_`, `run_once_`, `.tmpl`) are unchanged.
- Your daily workflow is still `chezmoi apply`.

### Prerequisites

- A working chezmoi dotfiles repo
- chezmoi-recipes binary (build from source with `make build` or grab a [release](https://github.com/fgrehm/chezmoi-recipes/releases))
- Familiarity with [chezmoi source state attributes](https://www.chezmoi.io/reference/source-state-attributes/)

## Step 1: Inventory your dotfiles

Group your chezmoi source files by tool or concern. For each file, ask: "what tool does this belong to?"

| File | Recipe |
|------|--------|
| `dot_zshrc.tmpl`, `dot_bashrc`, `dot_shellrc.d/env.sh` | `shell` |
| `private_dot_config/git/config.tmpl`, `dot_shellrc.d/git.sh` | `git` |
| `private_dot_config/alacritty/alacritty.toml` | `alacritty` |
| `run_once_install-neovim.sh.tmpl` | `neovim` |

Some files don't belong to any single tool. These need special handling (see Step 4):

- **`.chezmoiignore`** -- environment filtering moves to `.recipeignore`
- **`.chezmoidata/packages.yaml`** -- each recipe installs its own packages
- **`.chezmoiexternal.toml.tmpl`** -- each recipe downloads its own binaries
- **`.chezmoidata/completions.yaml`** -- each recipe generates its own completions

## Step 2: Set up chezmoi-recipes

```bash
# Initialize: creates recipes/ dir, chezmoi source dir, .chezmoi.toml.tmpl, shared scripts
chezmoi-recipes init --recipes-dir ./recipes

# Re-configure chezmoi to use the chezmoi-recipes source directory
# (prompts for name and email if not already in the config)
chezmoi init --source ~/.local/share/chezmoi-recipes/source
```

Create a `.recipeignore` in your recipes directory for environment filtering. This replaces the per-file entries you had in `.chezmoiignore`:

```
{{ if .isContainer }}
alacritty
brave
kde
1password
podman
ssh
flatpak
dropbox
cryptomator
claude
cartage
{{ end }}

{{ if or .isContainer (not (and (hasKey . "hasNvidiaGPU") .hasNvidiaGPU)) }}
cuda
{{ end }}
```

One line per recipe name, wrapped in the same template conditionals you already use in `.chezmoi.toml.tmpl`.

## Step 3: Create your first recipe

Pick something self-contained. `git` is a good candidate: a config file, a shell module, and an install script.

```bash
mkdir -p recipes/git/chezmoi/{.chezmoiscripts,private_dot_config/git,dot_shellrc.d}
```

Move files from your chezmoi source into the recipe:

```bash
# Config file
mv home/private_dot_config/git/config.tmpl recipes/git/chezmoi/private_dot_config/git/

# Shell aliases
mv home/dot_shellrc.d/git.sh recipes/git/chezmoi/dot_shellrc.d/

# Install script
mv home/.chezmoiscripts/run_once_install-gh.sh.tmpl recipes/git/chezmoi/.chezmoiscripts/
```

Write a README:

```markdown
# git

Git configuration with templated user identity and SSH commit signing.

## What it does

- Configures git user name and email from chezmoi template variables
- Enables SSH commit signing when `~/.ssh/id_ed25519.pub` exists
- Installs GitHub CLI (`gh`)
- Adds git shell aliases (`ga`, `gc`, `gd`, `gs`, `gl`) sourced via `~/.shellrc.d/`

## Template variables

| Variable | Description |
|----------|-------------|
| `.name` | Git commit author name |
| `.email` | Git commit author email |
```

Test it:

```bash
chezmoi-recipes overlay --dry-run    # preview what would be overlaid
chezmoi-recipes overlay git          # overlay just the git recipe
chezmoi diff                         # see what chezmoi would change
```

## Step 4: Deal with shared files

### Package installation

**Before (centralized `packages.yaml` consumed by one big script):**

```yaml
packages:
  common:
    - ripgrep
    - fzf
    - jq
```

**After (each recipe installs its own packages):**

```bash
#!/bin/bash
# chezmoi:template:left-delimiter="# {{" right-delimiter="}}"
set -euo pipefail
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

if dpkg -l ripgrep &>/dev/null; then
  log_skip "ripgrep already installed"
  exit 0
fi

# {{ if ne .chezmoi.username "root" }}
SUDO="sudo"
# {{ else }}
SUDO=""
# {{ end }}

log_info "Installing ripgrep..."
$SUDO apt-get install -y ripgrep
```

If a recipe needs multiple packages, check each one and only install what's missing.

### Shell completions

**Before (centralized `completions.yaml` consumed by one script):**

One script generates completions for all tools in a single pass.

**After (each recipe generates its own completions):**

```bash
#!/bin/bash
# .chezmoiscripts/run_onchange_after_completions-gh.sh
set -euo pipefail

BASH_DIR="$HOME/.local/share/bash-completion/completions"
ZSH_DIR="$HOME/.local/share/zsh/site-functions"
mkdir -p "$BASH_DIR" "$ZSH_DIR"

if command -v gh &>/dev/null; then
  gh completion -s bash > "$BASH_DIR/gh"
  gh completion -s zsh > "$ZSH_DIR/_gh"
fi
```

Use `run_onchange_after_` so completions regenerate when the script changes but run after the tool's install script.

### Environment filtering

**Before (`.chezmoiignore` listing every file):**

```
{{ if .isContainer }}
.config/alacritty/
.chezmoiscripts/install-nerdfonts.sh
.chezmoiscripts/set-alacritty-default.sh
{{ end }}
```

**After (`.recipeignore` listing recipe names):**

```
{{ if .isContainer }}
alacritty
{{ end }}
```

One line per recipe instead of tracking every file. When you add files to a recipe, the filtering just works.

For cases where you want the recipe to run in all environments but skip specific files (e.g., a desktop config file that should only exist on laptops), use a per-recipe `chezmoi/.chezmoiignore` instead:

```
{{ if .isContainer }}
private_dot_config/alacritty/
.chezmoiscripts/run_once_install-nerdfonts.sh
{{ end }}
```

During full overlay, all per-recipe `.chezmoiignore` files are merged into the source directory's `.chezmoiignore`. See the [recipe authoring guide](recipe-authoring.md) for details.

### External downloads

**Before (`.chezmoiexternal.toml.tmpl`):**

```toml
[".local/bin/mise"]
    type = "archive-file"
    url = "https://github.com/jdx/mise/releases/download/v2026.2.21/mise-v2026.2.21-linux-x64.tar.gz"
    path = "mise/bin/mise"
    executable = true
```

**After (recipe install script):**

```bash
#!/bin/bash
set -euo pipefail
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

if command -v mise &>/dev/null; then
  log_skip "mise already installed"
  exit 0
fi

log_info "Installing mise..."
DEST="$HOME/.local/bin/mise"
mkdir -p "$(dirname "$DEST")"
curl -fsSL "https://github.com/jdx/mise/releases/download/v2026.2.21/mise-v2026.2.21-linux-x64.tar.gz" \
  | tar -xz -C /tmp mise/bin/mise
mv /tmp/mise/bin/mise "$DEST"
chmod +x "$DEST"
```

More code per recipe, but no shared file to coordinate across them.

## Step 5: Migrate incrementally

Don't convert everything at once. Migrate one recipe at a time:

1. Pick a tool
2. Move its files into `recipes/<name>/chezmoi/`
3. Write a README
4. Remove the files from your old chezmoi source directory
5. Run `chezmoi-recipes overlay --dry-run` to verify
6. Run `chezmoi apply` to test
7. Commit

Each commit should leave `chezmoi apply` in a working state. If something breaks, the blast radius is one recipe.

In practice, you'll likely hit a few surprises: a script that assumed a file from another tool existed, a completion script that ran too early, a naming collision. Fix them as you go. The incremental approach keeps these manageable.

### Suggested migration order

Start simple, work toward the complicated stuff:

1. Small tools (git, ripgrep, heroku) -- few files, easy to verify
2. Shell foundation (zsh, bash, oh-my-zsh) -- other recipes drop files into `shellrc.d/`, so get this right first
3. Development tools (mise, neovim, zellij) -- self-contained but may have more files
4. Desktop apps (alacritty, brave, 1password) -- laptop-only, filtered by `.recipeignore`
5. System config (KDE, podman, CUDA) -- complex, platform-specific, multiple scripts

## Patterns

### Shell modules (shellrc.d)

The `shell` recipe owns the loader (`dot_shellrc.tmpl` which sources `~/.shellrc.d/*.sh`). Other recipes drop files into `dot_shellrc.d/`:

```
recipes/shell/chezmoi/dot_shellrc.tmpl       # loads ~/.shellrc.d/*.sh
recipes/git/chezmoi/dot_shellrc.d/git.sh     # git aliases
recipes/mise/chezmoi/dot_shellrc.d/mise.sh   # mise activation
recipes/cuda/chezmoi/dot_shellrc.d/cuda.sh   # CUDA PATH
```

Different files in the same target directory work fine. chezmoi-recipes only detects conflicts when two recipes write to the same file path.

### Script ordering

chezmoi runs scripts alphabetically within each phase (`before_`, then unnamed, then `after_`). Use numeric prefixes when ordering matters:

```
recipes/mise/chezmoi/.chezmoiscripts/
  run_once_00_install-mise.sh              # install mise binary first
  run_onchange_after_mise-install.sh.tmpl  # then install mise-managed tools
```

### Symlinked configs

If you keep configs outside chezmoi's source state (e.g., a `config/nvim/` directory at the repo root symlinked to `~/.config/nvim`), a recipe can create the symlink using chezmoi's `symlink_` prefix:

```
recipes/neovim/chezmoi/private_dot_config/symlink_nvim.tmpl
```

The file's contents are the symlink target:

```
{{ .chezmoi.workingTree }}/config/nvim
```

### Desktop app with icons

For `.desktop` files that reference icons from the repo:

```
recipes/brave/
  README.md
  chezmoi/
    .chezmoiscripts/run_once_install-brave.sh
    dot_local/share/applications/chatgpt.desktop.tmpl
```

Icons can stay at the repo root (e.g., `assets/icons/`) and be referenced via `{{ .chezmoi.workingTree }}/assets/icons/chatgpt.png` in templates.

## Pitfalls

### Don't split chezmoi's global files across recipes

chezmoi reads one `.chezmoiignore`, one `.chezmoiexternal.toml.tmpl`, and one `.chezmoidata/` directory. Put these in the chezmoi source directory directly (outside any recipe), or replace them with per-recipe alternatives as described in Step 4.

### Don't create dependencies between recipes

Each recipe should work on its own. If recipe B needs a tool from recipe A, recipe B should check for it and handle the missing case (install it, skip gracefully, or fail with a clear message).

### Use tool-specific script names

Two recipes can't both have `.chezmoiscripts/run_once_install-packages.sh`. Name scripts after the tool: `run_once_install-gh.sh`, `run_once_install-neovim.sh`.

### Watch out for `.chezmoiroot`

If your dotfiles repo uses `.chezmoiroot` (e.g., source state lives under `home/`), that convention applies to the old flat layout. With chezmoi-recipes, the source dir is a separate location (`~/.local/share/chezmoi-recipes/source/`), so `.chezmoiroot` no longer applies to your recipe files. Recipes use chezmoi naming directly in their `chezmoi/` subdirectory.

## End state

After migration, your repo looks like this:

```
my-dotfiles/
├── recipes/
│   ├── .recipeignore           # environment filtering
│   ├── shell/
│   │   ├── README.md
│   │   └── chezmoi/
│   │       ├── dot_bashrc
│   │       ├── dot_zshrc.tmpl
│   │       ├── dot_shellrc.tmpl
│   │       ├── dot_shellrc.d/
│   │       │   ├── env.sh
│   │       │   └── aliases.sh
│   │       └── .chezmoiscripts/
│   │           └── run_once_install-ohmyzsh.sh
│   ├── git/
│   │   ├── README.md
│   │   └── chezmoi/
│   │       ├── private_dot_config/git/config.tmpl
│   │       ├── dot_shellrc.d/git.sh
│   │       └── .chezmoiscripts/
│   │           ├── run_once_install-gh.sh.tmpl
│   │           └── run_onchange_after_completions-gh.sh
│   ├── alacritty/
│   │   ├── README.md
│   │   └── chezmoi/
│   │       ├── private_dot_config/alacritty/alacritty.toml
│   │       └── .chezmoiscripts/
│   │           ├── run_once_install-nerdfonts.sh
│   │           └── run_onchange_set-alacritty-default.sh.tmpl
│   └── ...
├── config/nvim/                # stays at repo root (symlinked by neovim recipe)
├── assets/icons/               # stays at repo root (referenced by .desktop templates)
├── README.md
├── CLAUDE.md
└── Makefile
```

Each tool's files live together. Removing a tool means deleting one directory instead of hunting through a flat tree.
