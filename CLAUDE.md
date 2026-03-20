# chezmoi-recipes

## Build and test

```bash
go build ./...
go test ./...
go vet ./...
```

## Testing rules (follow exactly)

- Use `t.TempDir()` for all filesystem operations.
- Override `HOME`, `XDG_DATA_HOME`, and `XDG_CONFIG_HOME` via `t.Setenv()` in every test — prevents touching host directories.
- Test the extracted `run*` functions directly, not Cobra command execution.
- Never invoke the real `chezmoi` binary in unit tests.

## Code style

**Error handling:** Wrap with context at every layer.

```go
// correct
return fmt.Errorf("loading state from %s: %w", path, err)

// wrong
return err
```

**Command output:** Business logic accepts `io.Writer`; `RunE` passes `cmd.OutOrStdout()`.

```go
// correct
func runList(recipesDir string, jsonOutput bool, w io.Writer) error { ... }

RunE: func(cmd *cobra.Command, args []string) error {
    return runList(recipesDir(), false, cmd.OutOrStdout())
}
```

**Context:** Pass `context.Context` as the first parameter in all command and loader functions. Cobra's `cmd.Context()` provides the root context.

**Dependencies:** Prefer stdlib. Current third-party: Cobra, BurntSushi/toml only. Avoid adding new deps.

## Standards

**Commits:** Conventional commits, present tense, under 72 characters. Use scopes when they clarify the component.

```
feat(recipe): add neovim recipe
fix(overlay): handle missing home directory
chore(deps): update cobra to v1.11
```

**File paths:** Use `internal/paths` for all runtime paths (XDG-aware, returns `string, error`).
**Exit codes:** 0 success, 1 general error, 2 usage error.
**Timestamps:** ISO 8601.

## What this is

A Go CLI that adds a recipe layer on top of [chezmoi](https://www.chezmoi.io/). The user's dotfiles repo is the chezmoi working tree. A `.chezmoiroot` file points chezmoi at a gitignored `compiled-home/` directory, rebuilt from `home/` (tracked source files) + `recipes/` (recipe fragments) on every `read-source-state.pre` hook. chezmoi does all the real work (dotfile management, scripts, templates).

**Tech stack:** Go, [Cobra](https://github.com/spf13/cobra), [BurntSushi/toml](https://github.com/BurntSushi/toml). Target: Debian 13 (Trixie).

## Architecture decisions

- **Recipes are directories.** A recipe is a directory with a `README.md` and optional `chezmoi/` subdirectory. The directory name is the recipe name. Any subdirectory with a `README.md` is a recipe.
- **Flat structure.** No composition or inheritance between recipes. Each is independent.
- **chezmoi integration via `.chezmoiroot`.** The dotfiles repo is the chezmoi working tree. `.chezmoiroot` points at a gitignored `compiled-home/` directory. `read-source-state.pre` runs `chezmoi-recipes overlay` to rebuild `compiled-home/` from `home/` + `recipes/`. Guard hooks block commands that would write to `compiled-home/` (`add`, `edit`, `re-add`, etc.). `chezmoi update` works natively. See `docs/chezmoi-integration.md`.
- **Stay thin.** chezmoi-recipes overlays files only. Package management, dependency resolution, and idempotency belong to chezmoi scripts inside recipes. Crossing that line means reimplementing Ansible.
- **Remove deletes, does not undo.** `remove` deletes files from the source directory and state. It does not reverse script side effects (installed packages, system config changes).
- **User data stays out of recipes.** Name, email, machine paths live in `.chezmoi.toml.tmpl` via chezmoi template variables, not in recipe files.
- **Atomic state writes.** State is written via temp file + `os.Rename`, not `os.WriteFile` directly.

## Directory layout

```
cmd/chezmoi-recipes/
  main.go               # signal context, calls ExecuteContext
  cmd/                  # one file per Cobra subcommand
internal/
  overlay/              # ClearDir + CopyTree + recipe overlay -> compiled-home/
  paths/                # path helpers (CompiledHomeDir, HomeDir, XDG state dir)
  recipe/               # discover and load recipe directories
  scaffold/             # generate new recipe skeletons
  setup/                # init: .chezmoiroot, home/, config template, .gitignore, .editorconfig, .shellcheckrc
  state/                # JSON state file (atomic write via rename)
  ignore/               # .recipeignore parsing (Go template + TOML data)
examples/               # reference recipe implementations
  <name>/
    README.md
    chezmoi/            # source state fragment overlaid into compiled-home/
```

Runtime:
- State: `$XDG_DATA_HOME/chezmoi-recipes/` (default `~/.local/share/chezmoi-recipes/`)

## CLI commands

```
chezmoi-recipes init                  # set up .chezmoiroot, home/, config template, .gitignore, .editorconfig, .shellcheckrc, README
chezmoi-recipes overlay [recipe...]   # rebuild compiled-home/ from home/ + recipes/ (called by read-source-state.pre hook)
chezmoi-recipes overlay --dry-run     # preview without writing
chezmoi-recipes list [--json]         # list available recipes
chezmoi-recipes scaffold <recipe>     # generate new recipe skeleton
chezmoi-recipes remove <recipe>       # remove recipe files from compiled-home/ and state
chezmoi-recipes status                # show applied recipes and their files
```

Global flags: `--recipes-dir` (default `./recipes`).

Primary user workflow: `chezmoi apply` overlays recipes automatically via the `read-source-state.pre` hook. `chezmoi update` pulls and applies.
