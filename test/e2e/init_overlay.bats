#!/usr/bin/env bats
# Integration tests: chezmoi-recipes init + overlay + chezmoi apply.
#
# These tests use the actual chezmoi-recipes binary (built from source)
# together with chezmoi to exercise the full .chezmoiroot workflow.

load test_helper

setup() {
  skip_if_not_container
  isolate_home
  build_chezmoi_recipes

  # Create a bare dotfiles repo (init will set up the structure)
  DOTFILES="$(mktemp -d)"
  cd "$DOTFILES"
  git init -q -b main
}

teardown() {
  cleanup
}

@test "chezmoi-recipes init creates .chezmoiroot and home/" {
  mkdir -p "$DOTFILES/recipes"

  run chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"
  [ "$status" -eq 0 ]

  # .chezmoiroot should exist
  [ -f "$DOTFILES/.chezmoiroot" ]
  run cat "$DOTFILES/.chezmoiroot"
  [[ "$output" == "compiled-home" ]]

  # home/ should exist with config template
  [ -d "$DOTFILES/home" ]
  [ -f "$DOTFILES/home/.chezmoi.toml.tmpl" ]

  # compiled-home/ should be populated (initial overlay)
  [ -f "$DOTFILES/compiled-home/.chezmoi.toml.tmpl" ]

  # .gitignore should include compiled-home/
  run cat "$DOTFILES/.gitignore"
  [[ "$output" == *"compiled-home/"* ]]
}

@test "chezmoi-recipes init + overlay + chezmoi apply deploys files" {
  mkdir -p "$DOTFILES/recipes"
  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  # Replace config template with non-interactive version for testing
  cat > "$DOTFILES/home/.chezmoi.toml.tmpl" << 'TMPL'
[data]
    name = "Test User"
    email = "test@example.com"
TMPL

  # Add a recipe
  mkdir -p "$DOTFILES/recipes/git/chezmoi"
  printf '# git recipe\n' > "$DOTFILES/recipes/git/README.md"
  printf '[user]\n    name = test\n' > "$DOTFILES/recipes/git/chezmoi/dot_gitconfig"

  # Add a home file
  printf '# my bashrc\n' > "$DOTFILES/home/dot_bashrc"

  # Run overlay
  run chezmoi-recipes overlay --recipes-dir "$DOTFILES/recipes"
  [ "$status" -eq 0 ]

  # compiled-home should have both home/ and recipe files
  [ -f "$DOTFILES/compiled-home/dot_bashrc" ]
  [ -f "$DOTFILES/compiled-home/dot_gitconfig" ]

  # chezmoi init + apply
  chezmoi init --no-tty --source "$DOTFILES"
  run chezmoi apply --source "$DOTFILES"
  [ "$status" -eq 0 ]

  # Target files should exist in HOME
  [ -f "$HOME/.bashrc" ]
  [ -f "$HOME/.gitconfig" ]

  run cat "$HOME/.bashrc"
  [[ "$output" == *"my bashrc"* ]]
}

@test "chezmoi-recipes overlay is idempotent" {
  mkdir -p "$DOTFILES/recipes"
  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  cat > "$DOTFILES/home/.chezmoi.toml.tmpl" << 'TMPL'
[data]
    name = "Test User"
TMPL

  mkdir -p "$DOTFILES/recipes/git/chezmoi"
  printf '# git\n' > "$DOTFILES/recipes/git/README.md"
  printf '[user]\n' > "$DOTFILES/recipes/git/chezmoi/dot_gitconfig"

  # Run overlay twice
  chezmoi-recipes overlay --recipes-dir "$DOTFILES/recipes"
  run chezmoi-recipes overlay --recipes-dir "$DOTFILES/recipes"
  [ "$status" -eq 0 ]

  [ -f "$DOTFILES/compiled-home/dot_gitconfig" ]
  [ -f "$DOTFILES/compiled-home/.chezmoi.toml.tmpl" ]
}

@test "chezmoi-recipes overlay --dry-run does not write files" {
  mkdir -p "$DOTFILES/recipes"
  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  cat > "$DOTFILES/home/.chezmoi.toml.tmpl" << 'TMPL'
[data]
    name = "Test User"
TMPL

  mkdir -p "$DOTFILES/recipes/git/chezmoi"
  printf '# git\n' > "$DOTFILES/recipes/git/README.md"
  printf '[user]\n' > "$DOTFILES/recipes/git/chezmoi/dot_gitconfig"

  # Clear compiled-home to verify dry-run doesn't populate it
  rm -rf "$DOTFILES/compiled-home"
  mkdir -p "$DOTFILES/compiled-home"

  run chezmoi-recipes overlay --dry-run --recipes-dir "$DOTFILES/recipes"
  [ "$status" -eq 0 ]
  [[ "$output" == *"dry-run"* ]]

  # compiled-home should still be empty (except for what init left)
  [ ! -f "$DOTFILES/compiled-home/dot_gitconfig" ]
}
