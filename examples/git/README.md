# Git

Git version control configuration with sensible defaults and shell aliases.

## What it does

- Configures git user identity (name, email) via chezmoi template variables
- Enables SSH commit signing when `~/.ssh/id_ed25519-sign.pub` exists
- Sets default branch to `main`, enables histogram diff, fetch pruning
- Adds global gitignore for editor temp files, OS junk, and dev artifacts
- Provides shell aliases (`ga`, `gc`, `gd`, `gs`, `gl`, etc.) with zsh completions

## Prerequisites

- Run `chezmoi init` to set name and email template variables (prompted at init time)
- Shell aliases in `dot_shellrc.d/git.sh` require a `~/.shellrc` loader that sources `~/.shellrc.d/*.sh`

## Template variables

- `.name` - git user name
- `.email` - git user email
