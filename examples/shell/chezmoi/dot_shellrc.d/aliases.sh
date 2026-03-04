# shellcheck shell=bash
# General-purpose aliases shared across bash and zsh.

# --- Colors ---
alias ls='ls --color=auto'
alias grep='grep --color=auto'
alias fgrep='fgrep --color=auto'
alias egrep='egrep --color=auto'

# --- ls shortcuts ---
alias ll='ls -alFh'
alias la='ls -A'
alias l='ls -CF'

# --- other ---
if command -v clotilde >/dev/null 2>&1; then
  alias clo='clotilde'
fi
