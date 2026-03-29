# chezmoi-recipes

A Go CLI that adds a recipe layer on top of [chezmoi](https://www.chezmoi.io/). The user's dotfiles repo is the chezmoi working tree. A `.chezmoiroot` file points chezmoi at a gitignored `compiled-home/` directory, rebuilt from `home/` (tracked source files) + `recipes/` (recipe fragments) on every `read-source-state.pre` hook. chezmoi does all the real work.

## Tech stack

- Go (see `go.mod` for version), [Cobra](https://github.com/spf13/cobra), [BurntSushi/toml](https://github.com/BurntSushi/toml)
- No test dependencies beyond stdlib
- Target: Debian 13 (Trixie)

## Directory layout

```
cmd/chezmoi-recipes/
  main.go               # signal context + ExecuteContext
  cmd/                  # one file per Cobra subcommand
internal/
  overlay/              # ClearDir + CopyTree + recipe overlay -> compiled-home/
  paths/                # path helpers (CompiledHomeDir, HomeDir, XDG state dir)
  recipe/               # recipe discovery and loading
  scaffold/             # new recipe skeleton generation
  setup/                # init: .chezmoiroot, home/, config template, .gitignore, .editorconfig, .shellcheckrc, README
  state/                # JSON state file, atomic write via rename
  ignore/               # .recipeignore: Go template parsed against chezmoi TOML data
examples/               # reference recipe implementations
```

## Tooling

- Go version: see `go.mod`.
- Linter: golangci-lint v2, managed as a Go tool dependency. Run `make lint` or
  `go tool golangci-lint run ./...`. Config in `.golangci.yml`.
- Formatting: `make fmt` runs gofumpt + goimports via `go tool golangci-lint fmt`.
- Dead code: `make deadcode` runs `go tool deadcode ./...` (hard gate in CI).
- Complexity: `make audit` runs gocyclo (informational at 15, hard gate at 30 in CI).
- Vulnerability check: `make govulncheck` runs `go tool govulncheck ./...` (hard gate in CI).
- Tests: `make test` runs with `-race -shuffle=on`. E2e tests: `make test-e2e` (bats).
- Pre-commit hook: `.githooks/pre-commit` auto-formats and lints staged files.
  Run `make setup-hooks` to activate.
- Release: tag-triggered via GoReleaser. Release notes extracted from `CHANGELOG.md`.
  See the Releasing section in CLAUDE.md.

## CHANGELOG

When reviewing PRs, verify that `CHANGELOG.md` has an `[Unreleased]` entry for any
user-facing change (features, fixes, breaking changes). Use
[Keep a Changelog](https://keepachangelog.com/) format.

## Coding conventions

**Error wrapping:** Always add context.

```go
// correct
return fmt.Errorf("loading recipe %q: %w", name, err)
```

**Command output:** Extract a `run*` function that accepts `io.Writer`; pass `cmd.OutOrStdout()` from `RunE`.

```go
func runList(recipesDir string, jsonOutput bool, w io.Writer) error { ... }

RunE: func(cmd *cobra.Command, args []string) error {
    return runList(recipesDir(), false, cmd.OutOrStdout())
},
```

**Context:** First parameter of every command and loader function is `context.Context`.

**Paths:** Use `internal/paths` for all runtime paths. Functions return `(string, error)`, never silently fall back on error.

**Commit format:** Conventional commits, present tense, scoped when useful.

```
feat(recipe): add neovim recipe
fix(overlay): handle missing home directory
```

## Testing rules

- `t.TempDir()` for all filesystem operations.
- `t.Setenv("HOME", ...)`, `t.Setenv("XDG_DATA_HOME", ...)`, `t.Setenv("XDG_CONFIG_HOME", ...)` in every test.
- Test `run*` functions directly — not `cmd.Execute()`.
- Never invoke the real `chezmoi` binary.

## Key constraints

- **Stay thin.** chezmoi-recipes overlays files only. Package management and script execution belong to chezmoi.
- **Minimal deps.** Only add a dependency if stdlib cannot do the job. Current deps: Cobra, BurntSushi/toml.
- **Flat recipes.** No composition or inheritance between recipes.
- **Atomic state.** Write state via `os.CreateTemp` + `os.Rename`, not `os.WriteFile`.
- **XDG paths.** All runtime paths go through `internal/paths`. No hardcoded `~/.config` or `~/.local`.
- **`chezmoi update` works natively.** The dotfiles repo is the chezmoi working tree. `chezmoi update` pulls the repo, then `read-source-state.pre` rebuilds `compiled-home/`. See `docs/chezmoi-integration.md`.
- **`private_dot_config` only.** All recipes that deploy files under `.config` must use `private_dot_config`, never `dot_config`. If two recipes disagree on privacy for the same target directory, chezmoi refuses to apply with `inconsistent state`.
- **GitHub release binaries go in `.chezmoiexternals/`.** Each recipe places a `<tool>.toml` file in `chezmoi/.chezmoiexternals/`. These are overlaid to `compiled-home/.chezmoiexternals/` with unique names (no conflicts). Files in `.chezmoiexternals/` are always rendered as templates by chezmoi; prefer direct `https://github.com/<owner>/<repo>/releases/latest/download/<asset>` URLs over `gitHubLatestReleaseAssetURL` (no API call, works offline and in rate-limited CI). See `docs/recipe-authoring.md`. Shell install scripts are for apt packages and tools needing post-install setup only.
