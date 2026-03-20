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

@test "chezmoi-recipes init generates working config for chezmoi apply" {
  # This test exercises the REAL generated config template end-to-end:
  #   1. chezmoi-recipes init generates .chezmoi.toml.tmpl with
  #      {{ .chezmoi.workingTree }}/recipes paths, sourceDir, hooks
  #   2. chezmoi init processes the template (expands workingTree, prompts)
  #   3. chezmoi apply fires read-source-state.pre hook -> overlay runs
  #   4. chezmoi deploys files from compiled-home/ to ~/

  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  # Verify the generated template uses portable paths (not absolute)
  run cat "$DOTFILES/home/.chezmoi.toml.tmpl"
  [[ "$output" == *'{{ .chezmoi.workingTree }}/recipes'* ]]
  [[ "$output" != *"$DOTFILES"* ]]

  # Add a recipe and a home file
  mkdir -p "$DOTFILES/recipes/git/chezmoi"
  printf '# git recipe\n' > "$DOTFILES/recipes/git/README.md"
  printf '[user]\n    name = test\n' > "$DOTFILES/recipes/git/chezmoi/dot_gitconfig"
  printf '# my bashrc\n' > "$DOTFILES/home/dot_bashrc"

  # chezmoi init with the real template (non-interactive via --promptString)
  chezmoi init --no-tty --source "$DOTFILES" \
    --promptString "name=Test User" \
    --promptString "email=test@example.com"

  # Verify rendered config has expanded {{ .chezmoi.workingTree }}
  local config="$XDG_CONFIG_HOME/chezmoi/chezmoi.toml"
  [ -f "$config" ]
  run cat "$config"

  # sourceDir should point to the dotfiles repo
  [[ "$output" == *"sourceDir"* ]]
  [[ "$output" == *"$DOTFILES"* ]]

  # recipesDir should be the expanded absolute path (not a template)
  [[ "$output" == *"$DOTFILES/recipes"* ]]

  # Overlay hook should be configured
  [[ "$output" == *"chezmoi-recipes"* ]]
  [[ "$output" == *"overlay"* ]]

  # chezmoi apply: the hook fires overlay automatically, then deploys
  run chezmoi apply --no-tty --source "$DOTFILES"
  [ "$status" -eq 0 ]

  # Target files should exist in HOME
  [ -f "$HOME/.bashrc" ]
  [ -f "$HOME/.gitconfig" ]

  run cat "$HOME/.bashrc"
  [[ "$output" == *"my bashrc"* ]]

  run cat "$HOME/.gitconfig"
  [[ "$output" == *"name = test"* ]]
}

@test "chezmoi-recipes init creates .editorconfig .shellcheckrc and README" {
  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  [ -f "$DOTFILES/.editorconfig" ]
  [ -f "$DOTFILES/.shellcheckrc" ]
  [ -f "$DOTFILES/README.md" ]

  run cat "$DOTFILES/.editorconfig"
  [[ "$output" == *"indent_size = 2"* ]]

  run cat "$DOTFILES/.shellcheckrc"
  [[ "$output" == *"SC1091"* ]]
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
