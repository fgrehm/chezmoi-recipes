# Git

Git version control configuration with sensible defaults and shell aliases.

## What it does

- Configures git user identity (name, email) via chezmoi template variables
- Enables SSH commit signing when `~/.ssh/id_ed25519-sign.pub` exists
- Sets default branch to `main`, enables histogram diff, fetch pruning
- Adds global gitignore for editor temp files, OS junk, and dev artifacts
- Provides shell aliases (`ga`, `gc`, `gd`, `gs`, `gl`, etc.) with zsh completions

## Prerequisites

- Run `chezmoi-recipes init` to set name and email template variables
- Apply the `shell` recipe for aliases to be sourced via `~/.shellrc.d/`

## Template variables

- `.name` - git user name
- `.email` - git user email
