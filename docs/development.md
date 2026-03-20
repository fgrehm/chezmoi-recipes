# Development Guide

This guide covers setting up a development environment and running tests.

## Local Setup

### Requirements

- Go 1.25.0 or later (specify version via `.tool-versions`)
- golangci-lint 1.62.2 or later (for linting)
- `make` (standard build tool)

### Using mise for version management

The project uses [mise](https://mise.jdx.dev/) to manage tool versions consistently. If you have mise installed:

```bash
# Trust and install tools from .tool-versions
mise trust
mise install

# Verify tools are installed
go version
golangci-lint version
```

Without mise, install Go 1.25.0+ and golangci-lint 1.62.2+ manually.

### Build and test

```bash
# Build the binary
make build

# Run all tests
make test

# Run a specific test
go test ./cmd/chezmoi-recipes/cmd -v -run TestOverlay

# Lint code
make lint

# Format code
make fmt
```

## Using a devcontainer

The project includes a devcontainer configuration for VS Code or other devcontainer-compatible editors.

### Quick start

**VS Code** ([Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension):

1. Open the project in VS Code.
2. Use the command palette (`Cmd+Shift+P` on macOS, `Ctrl+Shift+P` on Linux/Windows) and run "Dev Containers: Open Folder in Container".
3. Wait for the container to build and initialize.

**[crib](https://fgrehm.github.io/crib)** (terminal, no IDE required):

```bash
crib up     # build and start the container
crib shell  # enter an interactive shell
```

The devcontainer automatically:
- Installs Go and golangci-lint via mise
- Downloads Go module dependencies
- Builds the project binary
- Sets up XDG paths for testing

### Development inside the container

Once inside the container, use standard make commands:

```bash
make build    # Build the binary
make test     # Run tests
make lint     # Lint code
make fmt      # Format code
```

Or run Go directly:

```bash
go test ./...
go run ./cmd/chezmoi-recipes list
```

The binary is installed to `~/.local/bin/chezmoi-recipes` for quick access from anywhere in the container.

## Testing

### Test setup

Tests use temporary directories for filesystem operations and override XDG path environment variables to prevent touching the host system:

```go
func TestMyFeature(t *testing.T) {
    // All tests use t.TempDir() for isolated filesystem operations
    tmpDir := t.TempDir()

    // Override HOME and XDG paths
    t.Setenv("HOME", tmpDir)
    t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, ".local", "share"))
    t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
}
```

### Running tests

```bash
# Run all tests
make test

# Run with verbose output
go test -v ./...

# Run a specific package
go test ./internal/recipe

# Run a specific test
go test -v -run TestMyTest ./cmd/chezmoi-recipes/cmd

# Run with test coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test structure

- Unit tests live alongside source code (`_test.go` files)
- Test helpers are defined in `cmd/chezmoi-recipes/cmd/helpers_test.go` for CLI tests
- Common helpers include:
  - `setTestEnv()` - set up HOME and XDG variables
  - `setupTestRecipe()` - create a test recipe structure
  - `writeState()` - create test state files

## Code style

The project follows standard Go conventions:

- Format with `gofmt` (or `make fmt`)
- Lint with golangci-lint (or `make lint`)
- Use `go vet` for static analysis (run via `make vet`)
- Standard Go naming: exported names capitalized, unexported lowercase

## Common tasks

### Adding a new test

1. Create a file named `filename_test.go` in the same directory
2. Write test functions prefixed with `Test`
3. Use `t.TempDir()` and `t.Setenv()` for isolation
4. Run with `go test -v -run TestName ./path/to/package`

### Adding a new command

1. Add a file under `cmd/chezmoi-recipes/cmd/`
2. Use the Cobra command framework (see existing commands for examples)
3. Register the command in `cmd/chezmoi-recipes/cmd/root.go`
4. Write tests in `*_test.go` files

### Running chezmoi-recipes locally

```bash
# Build the project
make build

# Run a command
./bin/chezmoi-recipes list
./bin/chezmoi-recipes overlay --dry-run

# Or if installed locally
chezmoi-recipes list
```

## Testing dotfiles with chezmoi docker

chezmoi has built-in Docker support for testing dotfiles in a clean environment:

```bash
# Run chezmoi apply inside a fresh Debian container
chezmoi docker run -- apply

# Open a shell in a container with your dotfiles applied
chezmoi docker run -- sh

# Execute a command in a running chezmoi docker container
chezmoi docker exec -- bash
```

This can be a simpler alternative to devcontainers for quick smoke tests. Note that the container needs `chezmoi-recipes` available for the overlay hook to work. You may need to bind-mount the binary or install it inside the container. See the [chezmoi Docker docs](https://www.chezmoi.io/reference/commands/docker/) for details.

## Troubleshooting

### "command not found: go"

Ensure Go is installed and on your PATH. If using mise:

```bash
mise install
eval "$(mise activate)"
```

### Test failures with "permission denied"

Tests should not require sudo. If a test is failing due to permissions, ensure you're using `t.TempDir()` and not touching system directories.

### XDG paths pointing to host directories

The test setup must override XDG paths with `t.Setenv()`. If tests are touching `~/.local` or `~/.config`, the test's `setTestEnv()` helper is not being called.

### Module download failures

If `go mod download` fails, ensure you have internet access and the module cache is not corrupted:

```bash
go clean -modcache
go mod download
```
