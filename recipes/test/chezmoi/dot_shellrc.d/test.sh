# shellcheck shell=bash
#
# Shell integration for test.
# Placed in dot_shellrc.d/ so the shell recipe's loader sources it automatically.
#
# Path mapping: dot_shellrc.d/test.sh -> ~/.shellrc.d/test.sh
#
# No .tmpl suffix here because this file has no template directives.
# If you need chezmoi variables, rename to test.sh.tmpl and add the
# custom delimiter directive after the shebang comment.

# Guard: only run if test is installed.
# Prevents errors on first chezmoi apply before install scripts have run.
if ! command -v test &>/dev/null; then
  return 0
fi

# --- Aliases ---
# alias tests='test status'

# --- Environment ---
# export TEST_HOME="$HOME/.test"

# --- Activation ---
# eval "$(test init bash)"
