#!/usr/bin/env bats
# Test chezmoi update with the .chezmoiroot architecture.
#
# chezmoi update runs git pull on the working tree, then
# read-source-state.pre fires chezmoi-recipes overlay to rebuild
# compiled-home/ from the updated home/ and recipes/.

load test_helper

setup() {
  skip_if_not_container
  isolate_home
  build_chezmoi_recipes
}

teardown() {
  cleanup
}

@test "chezmoi update pulls changes and applies via overlay hook" {
  # Create the "upstream" dotfiles repo
  DOTFILES="$(mktemp -d)"
  cd "$DOTFILES"
  git init -q -b main
  mkdir -p recipes

  chezmoi-recipes init --recipes-dir "$DOTFILES/recipes"

  # Use a config template with the overlay hook (but no prompts)
  cat > "$DOTFILES/home/.chezmoi.toml.tmpl" << TMPL
[hooks.read-source-state.pre]
    command = "chezmoi-recipes"
    args = ["overlay", "--quiet", "--recipes-dir", "$DOTFILES/recipes"]

[data]
    name = "Test User"
    email = "test@example.com"
TMPL

  printf '# version 1\n' > "$DOTFILES/home/dot_bashrc"
  chezmoi-recipes overlay --quiet --recipes-dir "$DOTFILES/recipes"

  git add -A
  git commit -q -m "initial"

  # Create a bare remote
  local bare_remote
  bare_remote="$(mktemp -d)"
  git init -q --bare -b main "$bare_remote"
  git remote add origin "$bare_remote"
  git push -q -u origin main

  # Clone to simulate a user machine
  local user_dotfiles
  user_dotfiles="$(mktemp -d)/dotfiles"
  git clone -q "$bare_remote" "$user_dotfiles"

  # Create recipes/ dir (empty dirs aren't tracked by git)
  mkdir -p "$user_dotfiles/recipes"

  # Update the clone's config to use its own path
  cat > "$user_dotfiles/home/.chezmoi.toml.tmpl" << TMPL
[hooks.read-source-state.pre]
    command = "chezmoi-recipes"
    args = ["overlay", "--quiet", "--recipes-dir", "$user_dotfiles/recipes"]

[data]
    name = "Test User"
    email = "test@example.com"
TMPL

  chezmoi-recipes overlay --quiet --recipes-dir "$user_dotfiles/recipes"
  chezmoi init --no-tty --source "$user_dotfiles"
  chezmoi apply --source "$user_dotfiles"

  run cat "$HOME/.bashrc"
  [[ "$output" == *"version 1"* ]]

  # Push an update from the original repo
  cd "$DOTFILES"
  printf '# version 2\n' > "$DOTFILES/home/dot_bashrc"
  chezmoi-recipes overlay --quiet --recipes-dir "$DOTFILES/recipes"
  git add -A
  git commit -q -m "update bashrc"
  git push -q origin main

  # chezmoi update on the user clone
  cd "$user_dotfiles"
  run chezmoi update --source "$user_dotfiles"
  [ "$status" -eq 0 ]

  run cat "$HOME/.bashrc"
  [[ "$output" == *"version 2"* ]]

  rm -rf "$bare_remote" "$user_dotfiles"
}
