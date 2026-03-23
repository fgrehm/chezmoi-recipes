#!/bin/sh
# Install chezmoi and chezmoi-recipes, then optionally run chezmoi.
#
# Usage:
#   # Install binaries only
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)"
#
#   # Install and init dotfiles
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- \
#     init --apply username
#
#   # With prompt values and explicit repo
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/fgrehm/chezmoi-recipes/main/install.sh)" -- \
#     --bin-dir ~/.local/bin \
#     init --apply username/repo \
#       --promptString "Full name=Fabio Rehm" \
#       --promptString "Email address=me@example.com"
#
# Options (must come before chezmoi args):
#   -b, --bin-dir DIR        Where to install binaries (default: ~/.local/bin)
#   -t, --tag TAG            chezmoi-recipes release tag (default: latest)
#       --chezmoi-tag TAG    chezmoi release tag (default: latest)
#
# Any arguments after the installer options are forwarded to chezmoi.

set -eu

BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
TAG="${TAG:-latest}"
CHEZMOI_TAG="${CHEZMOI_TAG:-latest}"

_log() { printf '\033[1;34m==> %s\033[0m\n' "$*"; }
_die() { printf '\033[1;31merror: %s\033[0m\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    -b|--bin-dir)      BIN_DIR="$2";      shift 2 ;;
    -t|--tag)          TAG="$2";          shift 2 ;;
    --chezmoi-tag)     CHEZMOI_TAG="$2";  shift 2 ;;
    --)                shift; break ;;
    *)                 break ;;
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
  sh -c "$(curl -fsLS get.chezmoi.io)" -- -b "$BIN_DIR" -t "$CHEZMOI_TAG"
fi

# Install chezmoi-recipes
if command -v chezmoi-recipes >/dev/null 2>&1; then
  _log "chezmoi-recipes already installed ($(chezmoi-recipes version 2>&1))"
else
  _log "Installing chezmoi-recipes"
  if [ "$TAG" = "latest" ]; then
    URL="https://github.com/fgrehm/chezmoi-recipes/releases/latest/download/chezmoi-recipes_linux_${ARCH}.tar.gz"
  else
    URL="https://github.com/fgrehm/chezmoi-recipes/releases/download/${TAG}/chezmoi-recipes_linux_${ARCH}.tar.gz"
  fi
  curl -fsSL "$URL" | tar xz -C "$BIN_DIR"
fi

# No remaining args: binaries-only install
if [ $# -eq 0 ]; then
  printf '\nchezmoi and chezmoi-recipes installed to %s\n' "$BIN_DIR"
  exit 0
fi

# Forward remaining args to chezmoi (e.g., init --apply username ...)
_log "Running chezmoi $*"
exec chezmoi "$@"
