# shellcheck shell=bash
# Environment variables, editor setup, and PATH configuration.

# --- Editor ---
if command -v nvim >/dev/null 2>&1; then
  EDITOR=$(which nvim)
  export EDITOR
  export SUDO_EDITOR="$EDITOR"
  alias vim='nvim'
fi

# --- PATH ---
if [ -d "$HOME/.local/bin" ]; then
  PATH="$HOME/.local/bin:$PATH"
fi

# --- History ---
export HISTSIZE=32768
export HISTFILESIZE="${HISTSIZE}"
export HISTCONTROL=ignoreboth
