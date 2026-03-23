#!/usr/bin/env bats

# E2E tests for the generated install.sh (project-specific installer).
# Exercises the real overlay and chezmoi apply in a container.

load test_helper

setup() {
  skip_if_not_container
  build_chezmoi_recipes
  isolate_home

  # Create a dotfiles repo with chezmoi-recipes structure
  SRC="$(mktemp -d)"
  git -C "$SRC" init -q -b main
  git -C "$SRC" config user.email "test@example.com"
  git -C "$SRC" config user.name "Test User"

  printf 'compiled-home\n' > "$SRC/.chezmoiroot"
  printf 'compiled-home/\n' > "$SRC/.gitignore"
  mkdir -p "$SRC/home" "$SRC/recipes/hello/chezmoi" "$SRC/compiled-home"

  cat > "$SRC/home/.chezmoi.toml.tmpl" << 'TMPL'
sourceDir = "{{ .chezmoi.workingTree }}"

[hooks.read-source-state.pre]
    command = "chezmoi-recipes"
    args = ["overlay", "--recipes-dir", "{{ .chezmoi.workingTree }}/recipes"]

[data]
    name = {{ promptStringOnce . "name" "Full name" | quote }}
    email = {{ promptStringOnce . "email" "Email" | quote }}
TMPL

  printf '# hello\n' > "$SRC/recipes/hello/README.md"
  printf 'hello world\n' > "$SRC/recipes/hello/chezmoi/dot_hellorc"

  git -C "$SRC" add -A
  git -C "$SRC" commit -q -m "initial"

  # Bare clone for chezmoi init
  BARE_REPO="$(mktemp -d)/dotfiles.git"
  git clone --bare "$SRC" "$BARE_REPO" 2>/dev/null

  # Generate install.sh using chezmoi-recipes init (with repo URL)
  INSTALL_SH="$SRC/install.sh"
  chezmoi-recipes init --recipes-dir "$SRC/recipes" <<< "$BARE_REPO"
}

teardown() {
  rm -rf "$SRC" "${BARE_REPO%/*}"
  cleanup
}

@test "generated install.sh applies recipe files with --promptString" {
  run sh "$INSTALL_SH" \
    --promptString "Full name=Test User" \
    --promptString "Email=test@example.com"
  echo "$output"
  [ "$status" -eq 0 ]

  # Recipe file applied
  [ -f "$HOME/.hellorc" ]
  [ "$(cat "$HOME/.hellorc")" = "hello world" ]

  # Config template was processed (chezmoi.toml exists with data vars)
  [ -f "$XDG_CONFIG_HOME/chezmoi/chezmoi.toml" ]
  grep -q 'name = "Test User"' "$XDG_CONFIG_HOME/chezmoi/chezmoi.toml"
  grep -q 'email = "test@example.com"' "$XDG_CONFIG_HOME/chezmoi/chezmoi.toml"
}
