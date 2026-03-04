#!/bin/bash
# Smoke test for chezmoi-recipes inside the devcontainer.
# Run from anywhere inside the container.
set -euo pipefail

PASS=0
FAIL=0
WORKSPACE="${WORKSPACE:-/workspace}"
DOTFILES="$HOME/dotfiles"
SOURCE_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/chezmoi-recipes/source"
STATE_FILE="${XDG_DATA_HOME:-$HOME/.local/share}/chezmoi-recipes/state.json"
CHEZMOI_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/chezmoi/chezmoi.toml"

# --- helpers ---

pass() {
  PASS=$((PASS + 1))
  echo "  PASS: $1"
}

fail() {
  FAIL=$((FAIL + 1))
  echo "  FAIL: $1"
}

assert() {
  local desc="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    pass "$desc"
  else
    fail "$desc"
  fi
}

assert_output_contains() {
  local desc="$1"
  local expected="$2"
  shift 2
  local output
  output=$("$@" 2>&1) || true
  if echo "$output" | grep -qF "$expected"; then
    pass "$desc"
  else
    fail "$desc (expected '$expected' in output)"
    echo "    got: $output"
  fi
}

assert_output_not_contains() {
  local desc="$1"
  local unexpected="$2"
  shift 2
  local output
  output=$("$@" 2>&1) || true
  if echo "$output" | grep -qF "$unexpected"; then
    fail "$desc (found '$unexpected' in output)"
    echo "    got: $output"
  else
    pass "$desc"
  fi
}

assert_fails() {
  local desc="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    fail "$desc (expected failure but succeeded)"
  else
    pass "$desc"
  fi
}

section() {
  echo ""
  echo "=== $1 ==="
}

cleanup() {
  rm -rf "$DOTFILES"
  rm -rf "${XDG_DATA_HOME:-$HOME/.local/share}/chezmoi-recipes"
  rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/chezmoi"
}

# --- start fresh ---
cleanup

section "1. Prerequisites"

assert "chezmoi-recipes is on PATH" command -v chezmoi-recipes
assert "chezmoi is on PATH" command -v chezmoi

section "2. Version and help"

assert "version prints output" chezmoi-recipes version
assert_output_contains "help lists overlay command" "overlay" chezmoi-recipes --help
assert_output_contains "help lists init command" "init" chezmoi-recipes --help
assert_output_contains "help lists list command" "list" chezmoi-recipes --help
assert_output_contains "help lists remove command" "remove" chezmoi-recipes --help
assert_output_contains "help lists status command" "status" chezmoi-recipes --help

section "3. Set up test dotfiles"

mkdir -p "$DOTFILES/recipes"
cd "$DOTFILES"
git init -q -b main
cp -r "$WORKSPACE/examples/"* "$DOTFILES/recipes/"

assert "alacritty recipe exists" test -d "$DOTFILES/recipes/alacritty"
assert "git recipe exists" test -d "$DOTFILES/recipes/git"
assert "neovim recipe exists" test -d "$DOTFILES/recipes/neovim"
assert "ripgrep recipe exists" test -d "$DOTFILES/recipes/ripgrep"
assert "shell recipe exists" test -d "$DOTFILES/recipes/shell"

section "4. Init"

cd "$DOTFILES"
chezmoi-recipes init

assert ".chezmoi.toml.tmpl created" test -f "$SOURCE_DIR/.chezmoi.toml.tmpl"
assert_output_contains "template has hook config" "read-source-state.pre" cat "$SOURCE_DIR/.chezmoi.toml.tmpl"
assert_output_contains "template has promptStringOnce for name" "promptStringOnce" cat "$SOURCE_DIR/.chezmoi.toml.tmpl"
assert_output_contains "template has isContainer" "isContainer" cat "$SOURCE_DIR/.chezmoi.toml.tmpl"

# Run chezmoi init to render the config template (non-interactive with defaults).
chezmoi init --source "$SOURCE_DIR" --promptString name="Test User" --promptString email="test@example.com"

assert "chezmoi.toml created" test -f "$CHEZMOI_CONFIG"
assert_output_contains "chezmoi.toml has hook config" "read-source-state.pre" cat "$CHEZMOI_CONFIG"
assert_output_contains "data has name" "Test User" cat "$CHEZMOI_CONFIG"
assert_output_contains "data has email" "test@example.com" cat "$CHEZMOI_CONFIG"
assert_output_contains "data has isContainer" "isContainer" cat "$CHEZMOI_CONFIG"

section "5. List"

cd "$DOTFILES"

assert_output_contains "list shows alacritty" "alacritty" chezmoi-recipes list
assert_output_contains "list shows git" "git" chezmoi-recipes list
assert_output_contains "list shows neovim" "neovim" chezmoi-recipes list
assert_output_contains "list shows ripgrep" "ripgrep" chezmoi-recipes list
assert_output_contains "list shows shell" "shell" chezmoi-recipes list

section "6. Overlay --dry-run"

cd "$DOTFILES"
DRY_OUTPUT=$(chezmoi-recipes overlay --dry-run 2>&1)

assert_output_contains "dry-run mentions alacritty" "alacritty" echo "$DRY_OUTPUT"
assert_output_contains "dry-run mentions git" "git" echo "$DRY_OUTPUT"
assert_output_contains "dry-run mentions neovim" "neovim" echo "$DRY_OUTPUT"
assert_output_contains "dry-run mentions ripgrep" "ripgrep" echo "$DRY_OUTPUT"
assert_output_contains "dry-run mentions shell" "shell" echo "$DRY_OUTPUT"

# dry-run should not write files
assert "dry-run did not create git config" test ! -f "$SOURCE_DIR/private_dot_config/git/config.tmpl"

section "7. Overlay"

cd "$DOTFILES"
chezmoi-recipes overlay

assert "source dir has git config" test -f "$SOURCE_DIR/private_dot_config/git/config.tmpl"
assert "source dir has dot_gitignore" test -f "$SOURCE_DIR/dot_gitignore"
assert "source dir has dot_bashrc" test -f "$SOURCE_DIR/dot_bashrc"
assert "source dir has alacritty config" test -f "$SOURCE_DIR/private_dot_config/alacritty/alacritty.toml"
assert "source dir has nvim init.lua" test -f "$SOURCE_DIR/private_dot_config/nvim/init.lua"

section "8. Status"

cd "$DOTFILES"
STATUS_OUTPUT=$(chezmoi-recipes status 2>&1)

assert_output_contains "status shows alacritty" "alacritty" echo "$STATUS_OUTPUT"
assert_output_contains "status shows git" "git" echo "$STATUS_OUTPUT"
assert_output_contains "status shows neovim" "neovim" echo "$STATUS_OUTPUT"
assert_output_contains "status shows ripgrep" "ripgrep" echo "$STATUS_OUTPUT"
assert_output_contains "status shows shell" "shell" echo "$STATUS_OUTPUT"

section "9. Chezmoi hook integration"

cd "$DOTFILES"
# chezmoi diff should work (hook fires overlay automatically)
assert "chezmoi diff runs via hook" chezmoi --source "$SOURCE_DIR" diff

section "10. Remove"

cd "$DOTFILES"
chezmoi-recipes remove ripgrep

assert_output_not_contains "status no longer shows ripgrep" "ripgrep" chezmoi-recipes status
assert "ripgrep install script removed" test ! -f "$SOURCE_DIR/.chezmoiscripts/run_once_install-ripgrep.sh"

section "11. .recipeignore"

cd "$DOTFILES"

# Write a .recipeignore that skips alacritty and neovim when isContainer is true
cat > "$DOTFILES/recipes/.recipeignore" << 'IGNORE'
{{ if .isContainer }}
alacritty
neovim
{{ end }}
IGNORE

# Re-overlay (need to re-add ripgrep recipe dir since remove only removes from source)
IGNORE_OUTPUT=$(chezmoi-recipes overlay --dry-run 2>&1)

# Check recipe header lines (e.g. "[1/3] git"), not full output which may
# mention filtered recipe names in stale file paths.
RECIPE_HEADERS=$(echo "$IGNORE_OUTPUT" | grep -E '^\[|^Overlaying [0-9]')

assert_output_not_contains "recipeignore skips alacritty" "alacritty" echo "$RECIPE_HEADERS"
assert_output_not_contains "recipeignore skips neovim" "neovim" echo "$RECIPE_HEADERS"
assert_output_contains "recipeignore keeps git" "git" echo "$RECIPE_HEADERS"
assert_output_contains "recipeignore keeps shell" "shell" echo "$RECIPE_HEADERS"

# Clean up .recipeignore for subsequent tests
rm "$DOTFILES/recipes/.recipeignore"

section "12. Idempotency"

cd "$DOTFILES"
chezmoi-recipes overlay

SECOND_OUTPUT=$(chezmoi-recipes overlay 2>&1)
assert_output_contains "second overlay reports no changes" "no changes" echo "$SECOND_OUTPUT"

section "13. Conflict detection"

cd "$DOTFILES"
# Create a conflict: ripgrep recipe also has dot_gitignore (owned by git)
mkdir -p "$DOTFILES/recipes/ripgrep/chezmoi"
cp "$DOTFILES/recipes/git/chezmoi/dot_gitignore" "$DOTFILES/recipes/ripgrep/chezmoi/dot_gitignore"

assert_fails "overlay fails on conflict" chezmoi-recipes overlay

# Clean up the conflicting file
rm "$DOTFILES/recipes/ripgrep/chezmoi/dot_gitignore"

# --- summary ---

echo ""
echo "==============================="
echo "  Results: $PASS passed, $FAIL failed"
echo "==============================="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
