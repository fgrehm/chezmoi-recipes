# test

TODO: One sentence describing what this recipe sets up.

## What it does

- Installs test
- Deploys test configuration to ~/.config/test/
- Adds shell aliases via ~/.shellrc.d/

## Requirements

- Debian 13 (Trixie)

## Template variables

| Variable | Description | Source |
|----------|-------------|--------|
| `.name` | User's full name | `chezmoi init` prompt |
| `.email` | User's email address | `chezmoi init` prompt |
| `.isContainer` | true in Docker/devcontainers | auto-detected |
