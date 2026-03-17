# Roadmap

Future work for chezmoi-recipes. None of these are committed to, just ideas worth tracking.

## Provenance comments in compiled-home/

Embed a comment in each file written to `compiled-home/` during overlay, recording where the file came from:

```bash
# chezmoi-recipes: recipes/git/chezmoi/.chezmoiscripts/run_once_install-git.sh
```

This enables smarter guard hooks (see below) and helps with debugging.

## Smart guard hooks

With provenance comments, guard hooks could redirect instead of blocking:

- `chezmoi edit ~/.gitconfig` reads provenance from `compiled-home/dot_gitconfig`, prints "this file comes from recipes/git/chezmoi/dot_gitconfig, opening that instead", and opens the actual source file.
- `chezmoi add ~/.bashrc` (no provenance, new file) copies to `home/dot_bashrc` and re-runs overlay.

Binary files and files without comment syntax would need a sidecar manifest (`compiled-home/.provenance.json`).

## Incremental overlay mode

Skip unchanged files during overlay using checksums or mtimes instead of clearing and rebuilding `compiled-home/` from scratch every time. Reduces latency for large setups.

## `chezmoi-recipes doctor` command

Validate the directory structure: `.chezmoiroot` exists and points at `compiled-home`, `.gitignore` includes `compiled-home/`, `home/` exists, no orphaned state entries.

## `.chezmoiexternal` support in recipes

Allow recipes to include `.chezmoiexternal.toml` fragments that get merged into a single `.chezmoiexternal.toml` in `compiled-home/`, similar to how `.chezmoiignore` is merged today.
