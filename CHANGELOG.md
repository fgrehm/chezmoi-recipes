# Changelog

## [v0.2.0] - 2026-03-18

### Breaking changes

- **Architecture rewrite: `.chezmoiroot` + `compiled-home/`.**
  The overlay model changed completely. chezmoi-recipes now uses `.chezmoiroot` to point chezmoi at a gitignored `compiled-home/` directory, which is rebuilt on every `read-source-state.pre` hook by overlaying `home/` + active recipes. This is incompatible with repos set up under v0.1.0. See the [migration guide](docs/migration-guide.md).

- **`--source-dir` flag removed.** `init` and `overlay` now derive the repo root automatically from the working directory. The explicit flag is gone.

### Features

- `pull` command and `apply.pre` hook. Running `chezmoi apply` now pulls the latest dotfiles repo changes automatically via the hook. The `pull` command can also be invoked directly.
- `install.sh` improvements: `-b` (binary dir), `-t` (tag), and `--chezmoi-tag` flags added. Script is now POSIX sh compatible.

### Fixes

- Fixed stale references and overlay summary output bug.
- Fixed `install.sh` for the `.chezmoiroot` approach.

### Tests

- End-to-end test suite using bats covering the full `.chezmoiroot` workflow.
- Additional unit test coverage for `paths.ChezmoiConfigFile` and XDG fallback branches.

### Docs

- All documentation updated for the `.chezmoiroot` architecture.
- Added migration guide (step 0).
- Added hooks crib sheet and symlink pattern to recipe authoring docs.
- Documented `chezmoi update` limitation and workaround.
- Examples trimmed to `git` and `ripgrep`; both verified working.

## [v0.1.0] - 2026-03-16

Initial release.
