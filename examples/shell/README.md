# Shell

Zsh and bash shell configuration with Oh My Zsh and modular drop-in system.

## What it does

- Installs Oh My Zsh (unattended, preserves existing `.zshrc`)
- Deploys `.zshrc` with custom git prompt (ahead/behind, dirty count, stash count)
- Deploys `.bashrc` with standard Debian/Ubuntu defaults
- Sets up modular shell config: `~/.shellrc` sources all `~/.shellrc.d/*.sh` modules
- Includes core modules:
  - `env.sh` - editor setup (nvim), PATH, history settings
  - `aliases.sh` - color defaults, ls shortcuts

## Prerequisites

- Run `chezmoi-recipes init` to set template variables (container detection for prompt)
- `zsh` and `curl` must be available

## Extending

Other recipes can add modules to `dot_shellrc.d/` (e.g., the `git` recipe adds `git.sh`).
The loader sources all `*.sh` files in alphabetical order. Machine-local overrides
go in `~/.shellrc.local` (not managed by chezmoi).

## Template variables

- `.isContainer` - shows a container indicator in the zsh prompt
- `.name`, `.email` - warns if still set to placeholders
