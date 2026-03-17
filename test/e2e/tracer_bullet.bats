#!/usr/bin/env bats
# Tracer bullet: validate that chezmoi works with a gitignored .chezmoiroot target.
#
# This test exercises chezmoi directly (no chezmoi-recipes Go code changes).
# It manually sets up the Approach A directory layout and verifies that
# chezmoi init, apply, diff, managed, and doctor all work correctly.
#
# Run: bats test/e2e/tracer_bullet.bats
#   or: make test-e2e

load test_helper

setup() {
  skip_if_not_container
  isolate_home
  setup_dotfiles_repo
  write_minimal_config_template
}

teardown() {
  cleanup
}

@test "chezmoi init finds config template via .chezmoiroot" {
  add_home_file "dot_bashrc" "# bashrc from home"
  run_manual_overlay

  run chezmoi_init
  [ "$status" -eq 0 ]

  # Config file should have been created
  [ -f "$XDG_CONFIG_HOME/chezmoi/chezmoi.toml" ]

  # Verify rendered values
  run cat "$XDG_CONFIG_HOME/chezmoi/chezmoi.toml"
  [[ "$output" == *"Test User"* ]]
  [[ "$output" == *"test@example.com"* ]]
}

@test "chezmoi apply deploys files from compiled-home" {
  add_home_file "dot_bashrc" "# bashrc from home"
  add_recipe "git" "dot_gitconfig" "[user]\n\tname = test"
  run_manual_overlay

  chezmoi_init

  run chezmoi apply --source "$DOTFILES"
  [ "$status" -eq 0 ]

  # Target files should exist in HOME
  [ -f "$HOME/.bashrc" ]
  [ -f "$HOME/.gitconfig" ]

  # Content should match source
  run cat "$HOME/.bashrc"
  [[ "$output" == *"bashrc from home"* ]]
}

@test "chezmoi managed lists files from home/ and recipes" {
  add_home_file "dot_bashrc" "# bashrc"
  add_recipe "git" "dot_gitconfig" "[user]"
  run_manual_overlay

  chezmoi_init

  run chezmoi managed --source "$DOTFILES"
  [ "$status" -eq 0 ]
  [[ "$output" == *".bashrc"* ]]
  [[ "$output" == *".gitconfig"* ]]
}

@test "chezmoi diff exits cleanly after apply" {
  add_home_file "dot_bashrc" "# bashrc"
  run_manual_overlay

  chezmoi_init
  chezmoi apply --source "$DOTFILES"

  run chezmoi diff --source "$DOTFILES"
  [ "$status" -eq 0 ]
}

@test "chezmoi doctor does not report hard errors" {
  add_home_file "dot_bashrc" "# bashrc"
  run_manual_overlay

  chezmoi_init

  # chezmoi doctor exits 0 if no errors (warnings are ok)
  run chezmoi doctor --source "$DOTFILES"
  [ "$status" -eq 0 ]
}

@test "chezmoi source-path resolves to compiled-home inside the repo" {
  add_home_file "dot_bashrc" "# bashrc"
  run_manual_overlay

  chezmoi_init

  run chezmoi source-path --source "$DOTFILES"
  [ "$status" -eq 0 ]
  [[ "$output" == *"compiled-home"* ]]
}

@test "chezmoi update pulls and applies when repo has a remote" {
  # Set up a bare remote so chezmoi update has something to pull from
  local bare_remote
  bare_remote="$(mktemp -d)"
  git init -q --bare -b main "$bare_remote"

  # Push initial state to remote
  add_home_file "dot_bashrc" "# version 1"
  run_manual_overlay
  cd "$DOTFILES"
  git add -A
  git commit -q -m "initial"
  git remote add origin "$bare_remote"
  git push -q -u origin main

  # Clone to a fresh location (simulating a user's machine)
  local user_dotfiles
  user_dotfiles="$(mktemp -d)/dotfiles"
  git clone -q "$bare_remote" "$user_dotfiles"

  # Manually overlay in the clone (simulating what the hook would do)
  DOTFILES="$user_dotfiles" run_manual_overlay

  chezmoi_init "$user_dotfiles"

  chezmoi apply --source "$user_dotfiles"
  [ -f "$HOME/.bashrc" ]
  run cat "$HOME/.bashrc"
  [[ "$output" == *"version 1"* ]]

  # Push an update to the remote from the original repo
  cd "$DOTFILES"
  printf '# version 2\n' > home/dot_bashrc
  DOTFILES="$DOTFILES" run_manual_overlay
  git add -A
  git commit -q -m "update bashrc"
  git push -q origin main

  # chezmoi update pulls the working tree (has .git + remote)
  cd "$user_dotfiles"
  run chezmoi update --source "$user_dotfiles"
  [ "$status" -eq 0 ]

  # git pull updated home/ in the clone, re-overlay and re-apply
  DOTFILES="$user_dotfiles" run_manual_overlay
  chezmoi apply --source "$user_dotfiles"

  run cat "$HOME/.bashrc"
  [[ "$output" == *"version 2"* ]]

  rm -rf "$bare_remote" "$user_dotfiles"
}
