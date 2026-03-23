#!/usr/bin/env bats

# Tests for install.sh argument parsing and forwarding.
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

  # Mock chezmoi: prints args so tests can verify forwarding.
  cat > "$MOCK_BIN/chezmoi" << 'MOCK'
#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "chezmoi version v0.0.0-mock"
  exit 0
fi
echo "CHEZMOI_EXEC"
for arg in "$@"; do
  echo "ARG:$arg"
done
MOCK
  chmod +x "$MOCK_BIN/chezmoi"

  # Mock chezmoi-recipes: just enough for the version check.
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

@test "no args: binaries-only install" {
  run sh "$INSTALL_SH"
  [ "$status" -eq 0 ]
  [[ "$output" == *"installed to"* ]]
  [[ "$output" != *"CHEZMOI_EXEC"* ]]
}

@test "forwards init args to chezmoi" {
  run sh "$INSTALL_SH" init --apply testuser
  [ "$status" -eq 0 ]
  [[ "$output" == *"CHEZMOI_EXEC"* ]]
  [[ "$output" == *"ARG:init"* ]]
  [[ "$output" == *"ARG:--apply"* ]]
  [[ "$output" == *"ARG:testuser"* ]]
}

@test "forwards --promptString to chezmoi" {
  run sh "$INSTALL_SH" init --apply testuser \
    --promptString "Full name=Test User" \
    --promptString "Email address=test@example.com"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ARG:--promptString"* ]]
  [[ "$output" == *"ARG:Full name=Test User"* ]]
  [[ "$output" == *"ARG:Email address=test@example.com"* ]]
}

@test "-b flag is consumed, not forwarded" {
  local custom_bin
  custom_bin="$(mktemp -d)"
  cp "$MOCK_BIN/chezmoi" "$custom_bin/chezmoi"
  cp "$MOCK_BIN/chezmoi-recipes" "$custom_bin/chezmoi-recipes"

  run sh "$INSTALL_SH" -b "$custom_bin" init --apply testuser
  [ "$status" -eq 0 ]
  [[ "$output" == *"ARG:init"* ]]
  [[ "$output" != *"ARG:-b"* ]]
  [[ "$output" != *"ARG:$custom_bin"* ]]
  rm -rf "$custom_bin"
}

@test "--bin-dir flag is consumed, not forwarded" {
  local custom_bin
  custom_bin="$(mktemp -d)"
  cp "$MOCK_BIN/chezmoi" "$custom_bin/chezmoi"
  cp "$MOCK_BIN/chezmoi-recipes" "$custom_bin/chezmoi-recipes"

  run sh "$INSTALL_SH" --bin-dir "$custom_bin" init --apply testuser
  [ "$status" -eq 0 ]
  [[ "$output" == *"ARG:init"* ]]
  [[ "$output" != *"ARG:--bin-dir"* ]]
  rm -rf "$custom_bin"
}

@test "-t flag is consumed, not forwarded" {
  run sh "$INSTALL_SH" -t v0.3.0 init --apply testuser
  [ "$status" -eq 0 ]
  [[ "$output" == *"ARG:init"* ]]
  [[ "$output" != *"ARG:-t"* ]]
  [[ "$output" != *"ARG:v0.3.0"* ]]
}

@test "--chezmoi-tag flag is consumed, not forwarded" {
  run sh "$INSTALL_SH" --chezmoi-tag v2.70.0 init --apply testuser
  [ "$status" -eq 0 ]
  [[ "$output" == *"ARG:init"* ]]
  [[ "$output" != *"ARG:--chezmoi-tag"* ]]
  [[ "$output" != *"ARG:v2.70.0"* ]]
}

@test "-- separates installer flags from chezmoi args" {
  run sh "$INSTALL_SH" -t v0.3.0 -- init --apply testuser
  [ "$status" -eq 0 ]
  [[ "$output" == *"CHEZMOI_EXEC"* ]]
  [[ "$output" == *"ARG:init"* ]]
  [[ "$output" != *"ARG:-t"* ]]
}

@test "explicit repo URL is forwarded" {
  run sh "$INSTALL_SH" init --apply git@github.com:user/repo
  [ "$status" -eq 0 ]
  [[ "$output" == *"ARG:git@github.com:user/repo"* ]]
}
