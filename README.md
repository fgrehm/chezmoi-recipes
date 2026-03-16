# chezmoi-recipes

A recipe layer for [chezmoi](https://www.chezmoi.io/).

> **Status: early development.** The core is working but I haven't yet migrated my own dotfiles to it - that's next. Expect rough edges and possible breaking changes. Feedback welcome.
>
> **Note on chezmoi compatibility:** chezmoi deliberately uses a single source directory with a 1:1 mapping to target state ([design FAQ](https://www.chezmoi.io/user-guide/frequently-asked-questions/design/#can-chezmoi-support-multiple-sources-or-multiple-source-states)). chezmoi-recipes works outside that model by overlaying multiple recipe fragments into the source directory. This is not endorsed by chezmoi and may interact poorly with `chezmoi add`, `chezmoi edit`, or future chezmoi changes. Use at your own risk.

## What is this?

chezmoi-recipes lets you split a chezmoi source directory into modular, self-contained **recipes**. Each recipe is a directory containing a chezmoi source fragment (configs, scripts, templates) and a README. chezmoi-recipes overlays recipe files into chezmoi's source directory via a [hook](https://www.chezmoi.io/reference/configuration-file/hooks/), then chezmoi applies as normal.

chezmoi already handles dotfile management, script execution, and templating. chezmoi-recipes adds structure on top: group related chezmoi files into named recipes you can understand and remove without hunting through the source tree.

## The problem

chezmoi is great for managing dotfiles. But its source directory is flat by design: every file mirrors a path in your home directory, and there's no built-in way to group related files together.

This works fine at first. Once you're managing 10+ tools, the source directory becomes a wall of `dot_`, `private_`, and `run_once_` files with no logical grouping. It gets hard to tell which files belong to which tool, what scripts are for what, and what you can safely remove.

The usual workarounds within chezmoi:

- Templated `.chezmoiignore` for per-machine differences. Works, but you end up with a giant ignore file that's hard to reason about.
- `.chezmoiroot` to keep repo root clean. Helps with repo organization but doesn't help with the flat namespace inside the source state.
- Splitting shell configs into smaller files. Good practice, but orthogonal to the source directory organization problem.

None of these give you a way to say "these 5 files and 2 scripts are all part of my neovim setup."

### Why not something else?

- **[Nix Home Manager](https://github.com/nix-community/home-manager)** solves this at a deeper level: each program module is self-contained, and packages + configs are declared together. But it requires learning Nix (steep curve) and buying into the Nix ecosystem. If you're already using chezmoi and happy with it, switching to Nix is a big leap.
- **[GNU Stow](https://www.gnu.org/software/stow/)** has the right mental model (per-tool directories symlinked into place), but no templating, encryption, or script execution.
- **[Dotter](https://github.com/SuperCuber/dotter)** is a lightweight alternative with Handlebars templates, but has a smaller ecosystem and fewer features than chezmoi.

chezmoi-recipes takes a different approach: keep chezmoi as the foundation (templating, scripts, encryption, cross-platform support) and add a thin organizational layer on top.

## How it works

chezmoi-recipes overlays recipe files into chezmoi's source directory. It integrates via chezmoi's `read-source-state.pre` hook, so the overlay happens automatically before chezmoi reads source state. Your workflow is just `chezmoi apply`.

```
your-repo/recipes/git/chezmoi/  ───┐
your-repo/recipes/neovim/chezmoi/ ─┼──  overlay  ──>  chezmoi source dir  ──>  chezmoi apply  ──>  ~/
your-repo/recipes/docker/chezmoi/ ─┘
```

Instead of a flat source directory:

```
~/.local/share/chezmoi/
  dot_gitconfig
  dot_config/git/ignore
  dot_config/nvim/init.lua.tmpl
  .chezmoiscripts/run_once_install-git.sh
  .chezmoiscripts/run_once_install-neovim.sh
  .chezmoiscripts/run_after_configure-neovim.sh
```

You organize recipes in your own repo:

```
my-dotfiles/
  recipes/
    git/
      README.md
      chezmoi/
        dot_gitconfig
        dot_config/git/ignore
        .chezmoiscripts/run_once_install-git.sh
    neovim/
      README.md
      chezmoi/
        dot_config/nvim/init.lua.tmpl
        .chezmoiscripts/run_once_install-neovim.sh
        .chezmoiscripts/run_after_configure-neovim.sh
```

Everything about a tool is co-located. You can understand, share, or remove a recipe without touching anything else.

## Core principles

- Convention over configuration: a recipe is discovered by its directory structure, no manifest files
- chezmoi does the work: no reinventing dotfile management, scripting, or templating
- Recipes are independent: no composition or dependencies between recipes
- Shareable by default: recipes work for anyone; personal data lives in chezmoi's `.chezmoi.toml.tmpl`
- Incremental adoption: migrate one tool at a time from an existing chezmoi setup
- Stay thin: chezmoi-recipes overlays files into the source directory, nothing more. If something starts looking like Ansible, it doesn't belong here

## Recipe structure

A recipe is a directory with a `README.md`:

```
neovim/
  README.md                                # documentation (required, used for discovery)
  chezmoi/                                 # chezmoi source state fragment
    .chezmoiscripts/
      run_once_install-neovim.sh           # package installation
      run_after_configure-neovim.sh        # post-config setup
    dot_config/nvim/
      init.lua.tmpl                        # config files using chezmoi naming
```

The directory name is the recipe name. The `chezmoi/` subdirectory uses chezmoi's naming conventions (`dot_`, `private_`, `symlink_`, `.tmpl`, etc.) and gets overlaid as-is into chezmoi's source directory.

See `docs/chezmoi-integration.md` for the full integration design.

## Tech stack

| Component  | Choice            |
|------------|-------------------|
| Language   | Go                |
| Foundation | chezmoi           |
| Target OS  | Debian 13 (Trixie)|
| Platform   | Linux (macOS and Windows support planned) |
| License    | MIT               |

## Installation

**Linux only** (macOS and Windows support planned).

**Install script** (recommended): installs chezmoi and chezmoi-recipes, then optionally clones and applies a dotfiles repo.

```bash
# Install binaries only
sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)"

# Install and apply from GitHub (assumes repo named "dotfiles")
sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- username

# Install and apply from an explicit repo
sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- username/repo
sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- https://github.com/username/repo
sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- git@github.com:username/repo
```

**Go install:**
```bash
go install github.com/fgrehm/chezmoi-recipes@latest
```

**Build from source:**
```bash
git clone https://github.com/fgrehm/chezmoi-recipes
cd chezmoi-recipes
make build
make install  # Installs to ~/.local/bin
```

## Getting started

1. Create a repo for your dotfiles:
   ```bash
   mkdir my-dotfiles && cd my-dotfiles
   git init
   ```

2. Initialize chezmoi-recipes (creates `recipes/` dir, chezmoi source dir, config template, and a `Makefile` with shell lint targets):
   ```bash
   chezmoi-recipes init
   ```

3. Run `chezmoi init` to process the config template (prompts for name, email, auto-detects environment):
   ```bash
   chezmoi init --source ~/.local/share/chezmoi-recipes/source
   ```

4. Scaffold a new recipe or copy an example:
   ```bash
   # Generate a starter recipe with annotated example files
   chezmoi-recipes scaffold git
   ```

5. Apply it:
   ```bash
   chezmoi apply
   ```

chezmoi-recipes integrates via chezmoi's `read-source-state.pre` hook, so `chezmoi apply` automatically overlays your recipes first. You can also run the overlay manually:

```bash
chezmoi-recipes overlay          # overlay all recipes into source dir
chezmoi-recipes overlay git      # overlay a specific recipe
```

## Example recipes

Example recipes live in `examples/` in the chezmoi-recipes repo. They are reference implementations you can copy into your own `recipes/` directory.

| Recipe | What it does |
|--------|-------------|
| `git` | Git config (templated user identity, SSH signing), global gitignore, shell aliases |
| `neovim` | Installs Neovim from GitHub releases, LazyVim plugin framework, full editor config |
| `alacritty` | GPU terminal config (font, keybindings), sets as default terminal |
| `shell` | Oh My Zsh, zshrc with custom git prompt, bashrc, modular shellrc.d loader |
| `ripgrep` | Installs ripgrep via apt |

## Usage

```bash
# Initialize (sets up config template, recipes dir, shared scripts)
chezmoi-recipes init

# Configure user data (prompts for name, email; auto-detects environment)
chezmoi init

# List available recipes
chezmoi-recipes list

# Apply all recipes (via chezmoi hook)
chezmoi apply

# Preview what chezmoi would do
chezmoi diff

# Overlay manually (without running chezmoi)
chezmoi-recipes overlay

# Preview what overlay would change
chezmoi-recipes overlay --dry-run

# Scaffold a new recipe (generates annotated starter files)
chezmoi-recipes scaffold mytool

# Remove a recipe (deletes files from source dir, does not undo scripts)
chezmoi-recipes remove git

# Show applied recipes and their files
chezmoi-recipes status

# Use a custom recipes directory
chezmoi-recipes list --recipes-dir /path/to/recipes
```

### Shell lint and format

`chezmoi-recipes init` writes a `Makefile` to your project root with targets for linting and formatting recipe shell scripts:

```bash
make shell-lint       # shellcheck all .sh / .sh.tmpl / .bash files
make shell-fmt        # format with shfmt (writes in place)
make shell-fmt-check  # check formatting without modifying (exit 1 if dirty)
make check            # shell-fmt-check + shell-lint
```

Requires [shfmt](https://github.com/mvdan/sh) and [shellcheck](https://www.shellcheck.net/).

### Shared script utilities

chezmoi-recipes deploys logging helpers (`log_info`, `log_skip`, `log_error`, `run_quiet`) to the chezmoi source directory. Recipe scripts can source them:

```bash
source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"
```

### Template data

`chezmoi-recipes init` writes a `.chezmoi.toml.tmpl` to the chezmoi source directory. When you run `chezmoi init`, chezmoi processes this template, prompting for user data and auto-detecting the environment. The rendered output becomes chezmoi's config file (`chezmoi.toml`).

| Variable | Source |
|----------|--------|
| `name` | Prompted via `promptStringOnce` at `chezmoi init` |
| `email` | Prompted via `promptStringOnce` at `chezmoi init` |
| `isContainer` | Auto-detected (/.dockerenv, env vars, etc.) |
| `isDebian` | Auto-detected from `.chezmoi.osRelease.id` |
| `hasNvidiaGPU` | Auto-detected via `lspci` (skipped in containers) |

Recipes reference these via chezmoi templates (e.g., `{{ .name }}` in `.tmpl` files). The hook config (`[hooks.read-source-state.pre]`) is also included in the template, so `chezmoi apply` automatically triggers the overlay.

## Development

See [docs/development.md](docs/development.md) for a complete guide to local setup, testing, and development workflows.

Quick start:

```bash
# Build
make build

# Run
./bin/chezmoi-recipes list

# Test
go test ./...
```

**Using a devcontainer?** See [docs/devcontainer.md](docs/devcontainer.md) for setup and usage instructions.

## License

MIT. See [LICENSE](LICENSE).
