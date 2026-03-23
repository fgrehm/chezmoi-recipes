#!/usr/bin/env bats

# Tests for install.sh (binaries-only installer).
# Requires Linux (the script rejects other OSes).

INSTALL_SH="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." && pwd)/install.sh"

setup() {
  [ "$(uname -s)" = "Linux" ] || skip "install.sh only supports Linux"

  # Isolate HOME so the install script cannot read or write the real
  # user's git config, XDG dirs, or ~/.local/bin.
  TEST_HOME="$(mktemp -d)"
  export HOME="$TEST_HOME"
  export XDG_CONFIG_HOME="$TEST_HOME/.config"
  export XDG_DATA_HOME="$TEST_HOME/.local/share"
  mkdir -p "$XDG_CONFIG_HOME" "$XDG_DATA_HOME"

  MOCK_BIN="$(mktemp -d)"

  # Mock chezmoi
  cat > "$MOCK_BIN/chezmoi" << 'MOCK'
#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "chezmoi version v0.0.0-mock"
  exit 0
fi
MOCK
  chmod +x "$MOCK_BIN/chezmoi"

  # Mock chezmoi-recipes
  cat > "$MOCK_BIN/chezmoi-recipes" << 'MOCK'
#!/bin/sh
echo "chezmoi-recipes 0.0.0-mock"
MOCK
  chmod +x "$MOCK_BIN/chezmoi-recipes"

  # Point BIN_DIR at the mock dir so the script doesn't prepend
  # $HOME/.local/bin (which may contain the real chezmoi) to PATH.
  export BIN_DIR="$MOCK_BIN"
  export PATH="$MOCK_BIN:$PATH"
}

teardown() {
  rm -rf "$MOCK_BIN"
  rm -rf "$TEST_HOME"
}

@test "installs and prints success message" {
  run sh "$INSTALL_SH"
  [ "$status" -eq 0 ]
  [[ "$output" == *"installed to"* ]]
}

@test "-b flag sets custom bin dir" {
  local custom_bin
  custom_bin="$(mktemp -d)"
  cp "$MOCK_BIN/chezmoi" "$custom_bin/chezmoi"
  cp "$MOCK_BIN/chezmoi-recipes" "$custom_bin/chezmoi-recipes"

  run sh "$INSTALL_SH" -b "$custom_bin"
  [ "$status" -eq 0 ]
  [[ "$output" == *"installed to $custom_bin"* ]]
  rm -rf "$custom_bin"
}

@test "skips already-installed binaries" {
  run sh "$INSTALL_SH"
  [ "$status" -eq 0 ]
  [[ "$output" == *"chezmoi already installed"* ]]
  [[ "$output" == *"chezmoi-recipes already installed"* ]]
}
