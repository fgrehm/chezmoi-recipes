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

## Build and test

```bash
go build ./...
go test ./...
go vet ./...
```

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
