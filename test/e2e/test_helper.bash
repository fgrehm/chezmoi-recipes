# Shared helpers for chezmoi-recipes e2e tests.
# Sourced by bats tests via: load test_helper

# Skip the entire test if not running inside a container.
skip_if_not_container() {
  if [ ! -f "/.dockerenv" ] \
    && [ ! -f "/run/.containerenv" ] \
    && [ ! -f "/var/devcontainer" ] \
    && [ -z "${CODESPACES:-}" ] \
    && [ -z "${REMOTE_CONTAINERS:-}" ] \
    && [ -z "${container:-}" ] \
    && [ -z "${CHEZMOI_RECIPES_E2E:-}" ]; then
    skip "not running inside a container (set CHEZMOI_RECIPES_E2E=1 to force)"
  fi
}

# Create a temporary dotfiles repo with the Approach A layout:
#   .chezmoiroot, .gitignore, home/, recipes/
# Sets DOTFILES to the created directory.
setup_dotfiles_repo() {
  DOTFILES="$(mktemp -d)"
  cd "$DOTFILES"
  git init -q -b main

  # .chezmoiroot tells chezmoi to look in compiled-home/
  printf 'compiled-home\n' > .chezmoiroot

  # compiled-home/ is generated, not tracked
  printf 'compiled-home/\n' > .gitignore

  mkdir -p home recipes compiled-home
}

# Write a minimal .chezmoi.toml.tmpl into home/ that has no hooks
# and no interactive prompts (tracer bullet tests validate .chezmoiroot, not templating).
write_minimal_config_template() {
  cat > "$DOTFILES/home/.chezmoi.toml.tmpl" << 'TMPL'
[data]
    name = "Test User"
    email = "test@example.com"
TMPL
}

# Add a recipe with a single chezmoi source file.
# Usage: add_recipe <name> <relpath> <content>
# Example: add_recipe git dot_gitconfig "[user]\n\tname = test"
add_recipe() {
  local name="$1" relpath="$2" content="$3"
  local dir="$DOTFILES/recipes/$name/chezmoi"
  mkdir -p "$dir/$(dirname "$relpath")"
  printf '%s\n' "$content" > "$dir/$relpath"
  # Every recipe needs a README.md
  printf '# %s\n' "$name" > "$DOTFILES/recipes/$name/README.md"
}

# Add a file to home/.
# Usage: add_home_file <relpath> <content>
add_home_file() {
  local relpath="$1" content="$2"
  mkdir -p "$DOTFILES/home/$(dirname "$relpath")"
  printf '%s\n' "$content" > "$DOTFILES/home/$relpath"
}

# Run the overlay manually (no chezmoi-recipes binary needed).
# Copies home/ then each recipe's chezmoi/ into compiled-home/.
run_manual_overlay() {
  rm -rf "$DOTFILES/compiled-home"
  mkdir -p "$DOTFILES/compiled-home"

  # Copy home/ first
  if [ -d "$DOTFILES/home" ] && [ "$(ls -A "$DOTFILES/home")" ]; then
    cp -a "$DOTFILES/home/." "$DOTFILES/compiled-home/"
  fi

  # Copy each recipe's chezmoi/ dir
  for recipe_dir in "$DOTFILES"/recipes/*/chezmoi; do
    [ -d "$recipe_dir" ] || continue
    cp -a "$recipe_dir/." "$DOTFILES/compiled-home/"
  done
}

# Override HOME and XDG dirs to isolate from the host.
# Call this in setup() so chezmoi writes to temp dirs.
isolate_home() {
  TEST_HOME="$(mktemp -d)"
  export HOME="$TEST_HOME"
  export XDG_CONFIG_HOME="$TEST_HOME/.config"
  export XDG_DATA_HOME="$TEST_HOME/.local/share"
  mkdir -p "$XDG_CONFIG_HOME" "$XDG_DATA_HOME"

  # git needs a user identity for commits
  git config --global user.email "test@example.com"
  git config --global user.name "Test User"
}

# Run chezmoi init with test defaults (non-interactive).
chezmoi_init() {
  local source_dir="${1:-$DOTFILES}"
  chezmoi init --no-tty --source "$source_dir"
}

# Clean up temp dirs. Call in teardown().
cleanup() {
  [ -n "${DOTFILES:-}" ] && rm -rf "$DOTFILES"
  [ -n "${TEST_HOME:-}" ] && rm -rf "$TEST_HOME"
}
