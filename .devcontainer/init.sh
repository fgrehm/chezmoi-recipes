#!/bin/bash
set -e

# Activate mise in shell profiles
echo 'eval "$(mise activate bash)"' >>~/.bashrc
echo 'eval "$(mise activate zsh)"' >>~/.zshrc

# Trust and install tools from .tool-versions
mise trust
mise install

# Activate for this shell and generate shims for non-interactive use (crib run)
eval "$(mise activate bash)"
MISE_DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/mise" mise reshim

# Download Go dependencies
go mod download

# Build the project
make build

# Install binary locally for easy access
mkdir -p ~/.local/bin
ln -sf /workspace/bin/chezmoi-recipes ~/.local/bin/chezmoi-recipes

# Create XDG directories for testing
mkdir -p "$XDG_CONFIG_HOME" "$XDG_DATA_HOME"

echo "Development environment ready"
echo "  - Go tools installed via mise (go, bats, chezmoi, golangci-lint, goreleaser)"
echo "  - Project built at bin/chezmoi-recipes"
echo "  - Use 'make test' to run tests"
echo "  - Use 'make test-e2e' to run e2e tests (bats)"
echo "  - Use 'make lint' to lint code"
