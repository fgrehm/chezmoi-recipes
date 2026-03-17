#!/bin/sh
# Install chezmoi and chezmoi-recipes, clone a dotfiles repo, and apply it.
#
# Usage:
#   # Install binaries only
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)"
#
#   # Install and apply from GitHub (assumes repo named "dotfiles")
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- username
#
#   # Install and apply from explicit repo
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- username/repo
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- https://github.com/username/repo
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- git@github.com:username/repo
#
# Options:
#   --dotfiles-dir DIR   Where to clone the repo (default: ~/dotfiles)
#   --bin-dir DIR        Where to install binaries (default: ~/.local/bin)

set -eu

BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
DOTFILES_DIR="${DOTFILES_DIR:-$HOME/dotfiles}"
REPO=""

_log() { printf '\033[1;34m==> %s\033[0m\n' "$*"; }
_die() { printf '\033[1;31merror: %s\033[0m\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --bin-dir)      BIN_DIR="$2";      shift 2 ;;
    --dotfiles-dir) DOTFILES_DIR="$2"; shift 2 ;;
    --*) _die "unknown option: $1" ;;
    *)   REPO="$1"; shift ;;
  esac
done

[ "$(uname -s)" = "Linux" ] || _die "only Linux is supported"

case "$(uname -m)" in
  x86_64)        ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) _die "unsupported architecture: $(uname -m)" ;;
esac

mkdir -p "$BIN_DIR"
export PATH="$BIN_DIR:$PATH"

# Install chezmoi
if command -v chezmoi >/dev/null 2>&1; then
  _log "chezmoi already installed ($(chezmoi --version))"
else
  _log "Installing chezmoi"
  sh -c "$(curl -fsLS get.chezmoi.io)" -- -b "$BIN_DIR"
fi

# Install chezmoi-recipes
if command -v chezmoi-recipes >/dev/null 2>&1; then
  _log "chezmoi-recipes already installed ($(chezmoi-recipes version 2>&1))"
else
  _log "Installing chezmoi-recipes"
  curl -fsSL \
    "https://github.com/fgrehm/chezmoi-recipes/releases/latest/download/chezmoi-recipes_linux_${ARCH}.tar.gz" \
    | tar xz -C "$BIN_DIR"
fi

# No repo given: binaries only
if [ -z "$REPO" ]; then
  printf '\nchezmoi and chezmoi-recipes installed to %s\n' "$BIN_DIR"
  printf 'Next: chezmoi-recipes init --recipes-dir <path/to/recipes>\n'
  exit 0
fi

# Normalize repo to full URL
case "$REPO" in
  https://*|http://*|git@*) REPO_URL="$REPO" ;;
  */*)                      REPO_URL="https://github.com/$REPO" ;;
  *)                        REPO_URL="https://github.com/$REPO/dotfiles" ;;
esac

# Clone dotfiles repo (skip if already present)
if [ -d "$DOTFILES_DIR/.git" ]; then
  _log "Using existing dotfiles at $DOTFILES_DIR"
else
  _log "Cloning $REPO_URL"
  git clone "$REPO_URL" "$DOTFILES_DIR"
fi

_log "Initializing chezmoi-recipes"
chezmoi-recipes init --recipes-dir "$DOTFILES_DIR/recipes"

_log "Configuring chezmoi"
chezmoi init --source "$DOTFILES_DIR"

_log "Applying dotfiles"
chezmoi apply

printf '\n\033[1;32mDone.\033[0m Open a new shell to pick up the changes.\n'
