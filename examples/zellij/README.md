# zellij

Terminal multiplexer installed from GitHub releases via `.chezmoiexternals/`.

## What it does

- Downloads the zellij binary from GitHub releases into `~/.local/bin/`
- Pins a specific version for reproducible installs

## Config

No managed configuration. Demonstrates the `.chezmoiexternals/` pattern for
installing a binary from GitHub releases with version pinning and architecture
translation (GNU/uname style).

See `chezmoi/.chezmoiexternals/zellij.toml` for the full pattern.
