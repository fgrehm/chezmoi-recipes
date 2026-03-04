# Alacritty

GPU-accelerated terminal emulator configuration.

## What it does

- Deploys Alacritty configuration (font, keyboard bindings, mouse settings)
- Sets Alacritty as the default terminal emulator via `update-alternatives`

## Prerequisites

- Alacritty must be installed (e.g., via `apt install alacritty`)
- CaskaydiaMono Nerd Font Mono installed on the system
- The set-default script runs only if `alacritty` is on PATH

## Config

- Font: CaskaydiaMono Nerd Font Mono
- Shift+Enter sends newline
- Ctrl+Shift+N opens a new window
- Mouse cursor stays visible while typing (workaround for alacritty#6703)
