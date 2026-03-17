# chezmoi-recipes

A Go CLI that overlays modular recipe directories into a [chezmoi](https://www.chezmoi.io/) source directory. On `chezmoi apply`, an `apply.pre` hook runs `chezmoi-recipes pull` (git pull on the dotfiles repo), then a `read-source-state.pre` hook runs `chezmoi-recipes overlay` (copies recipe files into the source dir). Both steps are automatic.

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
  overlay/              # Plan + Execute: copy recipe chezmoi/ files → source dir
  paths/                # XDG path resolution, returns (string, error)
  recipe/               # recipe discovery and loading
  scaffold/             # new recipe skeleton generation
  setup/                # init command: config template + shared scripts
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
