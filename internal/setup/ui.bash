#!/bin/bash
# Shared logging utilities for chezmoi-recipes recipe scripts.
# Source this at the top of chezmoi scripts:
#   source "$CHEZMOI_SOURCE_DIR/scripts/ui.bash"

_UI_LOGFILE=$(mktemp)
_ui_cleanup() { rm -f "$_UI_LOGFILE"; }
trap _ui_cleanup EXIT

_ts() { date -u '+%H:%M:%S'; }
log_info()  { printf '[%s] ==> %s\n' "$(_ts)" "$*"; }
log_skip()  { printf '[%s] ==> %s (skipped)\n' "$(_ts)" "$*"; }
log_error() { printf '[%s] ==> ERROR: %s\n' "$(_ts)" "$*" >&2; }

# Run a command with stdout/stderr captured.
# On success: silent. On failure: dump captured output, then return the exit code.
# Empty arguments are stripped so callers can write: run_quiet "$SUDO" cmd args...
# When SUDO is empty, the empty string is dropped rather than executed.
run_quiet() {
  : >"$_UI_LOGFILE"
  local cmd=()
  for arg in "$@"; do [[ -n "$arg" ]] && cmd+=("$arg"); done
  "${cmd[@]}" >>"$_UI_LOGFILE" 2>&1 && return 0
  local rc=$?
  log_error "command failed (exit $rc): ${cmd[*]}"
  cat "$_UI_LOGFILE" >&2
  return "$rc"
}
