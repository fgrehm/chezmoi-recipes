# Changelog

## [v0.3.0] - 2026-03-21

### Features

- `init` generates `.editorconfig` (2-space indent for shell files), `.shellcheckrc` (chezmoi template noise suppressions), and a starter `README.md`. All skip-if-exists.
- Config template uses `{{ .chezmoi.workingTree }}/<relPath>` instead of hardcoded absolute paths, so the generated config works when the repo is cloned to a different location or by a different user.
- Scaffold templates include vim modelines (`ft=bash.gotmpl`, `ft=toml.gotmpl`).
- Config template now sets `sourceDir` and `[diff] pager = "cat"`.

### Fixes

- Removed `--quiet` from overlay hook args (was swallowing useful output).
- Fixed scaffold shebang from `#!/bin/env bash` to `#!/usr/bin/env bash`.
- `writeIfMissing` and `WriteChezmoiConfig` use `os.Lstat` to detect symlinks without following them, preventing writes through symlinks.
- `WriteChezmoiConfig` rejects non-regular files even with `--force`.
- Next-step hint after init now quotes the repo root path for copy-paste safety.

### Tests

- New e2e test exercising the real generated config template end-to-end: `chezmoi-recipes init` -> `chezmoi init` (with `--promptString`) -> `chezmoi apply` (hook fires overlay) -> files deployed.
- E2e tests for `.editorconfig`, `.shellcheckrc`, and `README.md` generation.
- Skip-if-exists tests for all `writeIfMissing` targets.
- `XDG_CONFIG_HOME` override added to all setup tests.
- E2e job added to CI (bats + chezmoi installed on ubuntu-latest).

### Docs

- Recipe ordering: shell/base recipe must exist before recipes that ship `dot_shellrc.d/` fragments.
- `lookPath` gotcha: template functions evaluate before scripts run, with three workaround patterns.
- Critical vs optional install scripts: when to use `set -euo pipefail` vs the `_install()` wrapper.
- Roadmap: scaffold archetypes, `init --with-tests`, custom linters, `chezmoi docker` testing.

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
