#!/usr/bin/env bats
# Guard hook tests: verify that chezmoi add/edit/etc. are blocked
# when using the generated config template.
#
# The guard hooks use "sh -c 'echo Error... >&2; exit 1'" to print
# an error message and exit non-zero, preventing chezmoi from
# modifying the generated compiled-home/ directory.

load test_helper

setup() {
  skip_if_not_container
  isolate_home
  build_chezmoi_recipes

  DOTFILES="$(mktemp -d)"
  cd "$DOTFILES"
  git init -q -b main
  mkdir -p recipes

  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  # Replace config template with non-interactive version that keeps guard hooks
  cat > "$DOTFILES/home/.chezmoi.toml.tmpl" << TMPL
[hooks.read-source-state.pre]
    command = "chezmoi-recipes"
    args = ["overlay", "--quiet", "--recipes-dir", "$DOTFILES/recipes"]

[hooks.add.pre]
    command = "sh"
    args = ["-c", "echo 'Error: compiled-home/ is generated' >&2; exit 1"]

[hooks.forget.pre]
    command = "sh"
    args = ["-c", "echo 'Error: compiled-home/ is generated' >&2; exit 1"]

[data]
    name = "Test User"
    email = "test@example.com"
TMPL

  # Deploy a file so we have something to target
  printf '# bashrc\n' > "$DOTFILES/home/dot_bashrc"
  chezmoi-recipes overlay --quiet --recipes-dir "$DOTFILES/recipes"
  chezmoi init --no-tty --source "$DOTFILES"
  chezmoi apply --source "$DOTFILES"
}

teardown() {
  cleanup
}

@test "chezmoi add is blocked by guard hook" {
  run chezmoi add --source "$DOTFILES" "$HOME/.bashrc"
  [ "$status" -ne 0 ]
  [[ "$output" == *"compiled-home/ is generated"* ]]
}

@test "chezmoi forget is blocked by guard hook" {
  run chezmoi forget --source "$DOTFILES" "$HOME/.bashrc"
  [ "$status" -ne 0 ]
  [[ "$output" == *"compiled-home/ is generated"* ]]
}

@test "chezmoi apply still works (no guard hook)" {
  run chezmoi apply --source "$DOTFILES"
  [ "$status" -eq 0 ]
}

@test "chezmoi diff still works (no guard hook)" {
  run chezmoi diff --source "$DOTFILES"
  [ "$status" -eq 0 ]
}

@test "chezmoi managed still works (no guard hook)" {
  run chezmoi managed --source "$DOTFILES"
  [ "$status" -eq 0 ]
  [[ "$output" == *".bashrc"* ]]
}
