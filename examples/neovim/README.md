# Neovim

Neovim editor with LazyVim plugin framework and custom configuration.

## What it does

- Downloads and installs the latest Neovim release from GitHub
- Deploys a full Neovim configuration based on LazyVim
- Bootstraps plugin installation via `Lazy! sync` on first run
- Includes plugins for: auto-save, bufferline, completion, colorscheme (TokyoNight), terminal, multi-cursor, and more

## Prerequisites

- Internet access for downloading Neovim binary and plugins
- `curl` must be available
- `sudo` access for installing to `/opt` and symlinking to `/usr/local/bin`

## Config

The bundled Neovim config uses LazyVim with:
- TokyoNight colorscheme (night style)
- ToggleTerm terminal (Ctrl-`)
- CaskaydiaMono Nerd Font (configure your terminal accordingly)
- OSC 52 clipboard (works over SSH)

Edit files under `chezmoi/private_dot_config/nvim/lua/plugins/` to customize plugins.
