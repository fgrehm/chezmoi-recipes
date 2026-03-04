# Devcontainer Guide

This project includes a devcontainer configuration for consistent development across machines.

## Quick start

### Requirements

- VS Code with the [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension
- Docker (Docker Desktop or a compatible Docker daemon)

### Opening the project

1. Clone the repository:
   ```bash
   git clone <repo-url> chezmoi-recipes
   cd chezmoi-recipes
   ```

2. Open in VS Code:
   ```bash
   code .
   ```

3. VS Code will detect the devcontainer and prompt to reopen in a container. Click "Reopen in Container" or use the command palette (`Cmd+Shift+P`, search "Dev Containers: Open Folder in Container").

4. Wait for the container to build and initialize (first run takes a minute or two).

Once ready, you'll see a green indicator in the bottom-left corner with the container name.

## What the devcontainer provides

The devcontainer:
- Debian 13 (Trixie) base image matching the project's target OS
- Go 1.25.0 (via mise)
- golangci-lint 1.62.2 (via mise)
- VS Code extensions: Go, golangci-lint
- Pre-built binary at `~/.local/bin/chezmoi-recipes`
- Environment variables configured for safe testing (XDG paths isolated to `/tmp`)

## Working inside the container

### Build and test

```bash
# Build the project
make build

# Run tests
make test

# Lint code
make lint

# Format code
make fmt

# Run a specific test
go test -v -run TestOverlay ./cmd/chezmoi-recipes/cmd
```

### Run the tool

```bash
# Using the installed binary
chezmoi-recipes list
chezmoi-recipes overlay --dry-run

# Or directly from the bin directory
./bin/chezmoi-recipes status
```

### Edit code

Changes made in VS Code are immediately reflected in the container (files are mounted).

### Terminal access

Open a terminal inside the container (`Ctrl+` ` in VS Code). You're in a bash/zsh shell inside the container with all tools available.

## Advanced usage

### Rebuild the container

If you update `.tool-versions` or `init.sh`, rebuild the container:

1. Run the command palette (`Cmd+Shift+P`)
2. Search "Dev Containers: Rebuild Container"
3. Click to rebuild

Or from the terminal inside the container:
```bash
# If you manually updated .tool-versions
mise install
make build
```

### Access the workspace directory

The container's working directory is `/workspace` (mapped to your repo root on the host).

```bash
# List repo files from inside the container
ls /workspace
cd /workspace
```

### Inspect XDG paths

The devcontainer sets XDG paths to `/tmp/` for test isolation:

```bash
echo $XDG_CONFIG_HOME    # /tmp/xdg-config
echo $XDG_DATA_HOME      # /tmp/xdg-data
```

These paths are temporary and deleted when you close the container, so tests don't pollute your host system.

### Install additional tools inside the container

If you need to install tools for debugging:

```bash
# Update package lists
sudo apt-get update

# Install a tool (e.g., jq for JSON inspection)
sudo apt-get install -y jq

# Or use mise to install additional versions
mise install node@20
```

Changes are lost when you rebuild the container. For persistent changes, update the devcontainer configuration.

### Run Go tests with breakpoints

VS Code's Go extension supports debugging:

1. Set a breakpoint by clicking in the gutter next to a line
2. Open the Run and Debug panel (`Cmd+Shift+D`)
3. Click "run and debug" or press `F5`
4. Select "Go" from the dropdown
5. Choose "debug test"

The debugger will pause at breakpoints, allowing you to inspect variables and step through code.

## Troubleshooting

### Container fails to start

Check Docker is running:
```bash
docker ps
```

If Docker is not available or the container failed to build, review the build logs. In VS Code, open the "Dev Containers" panel and check "Container Log".

### Changes don't appear in the container

Files mounted into the container should update automatically. If not:
- Reload the VS Code window (`Cmd+Shift+P`, search "reload window")
- Rebuild the container

### Port forwarding

If the project needs to expose a port (e.g., for a local server), add to `devcontainer.json`:

```json
"forwardPorts": [3000, 8080]
```

Then rebuild the container.

### Too much disk space used

The container image and volumes can accumulate. Clean up:

```bash
# Remove unused containers
docker container prune

# Remove unused images
docker image prune

# Remove unused volumes
docker volume prune
```

Or use Docker Desktop's UI (Settings > Resources > Clean Now).

### Connecting to the host's network

The container can access the host network. To connect to a service on the host:

```bash
# Get the host's IP (usually 172.17.0.1 on Linux)
ip route show default | awk '{print $3}'

# Use that IP in the container (e.g., for a database on the host)
curl http://172.17.0.1:5432
```

## Performance

### Slow file operations

If file I/O feels slow, Docker Desktop on macOS can have performance issues with mounted volumes. Consider:
- Using Docker Desktop's "enhanced container isolation" (Settings > Resources)
- Using the container's filesystem for temporary build artifacts

On Linux, performance is usually excellent.

### Slow builds

Go compilation can be slow if the module cache is not persisted. The devcontainer includes the cache in the image, so subsequent builds should be faster.

To speed up initial builds, increase Docker's memory allocation (Settings > Resources > Memory).

## Exiting and re-entering the container

### Switching back to local development

Click the green container indicator in the bottom-left and select "Reopen Folder Locally".

### Re-entering the container

Open the folder in VS Code again and accept the prompt to reopen in the container.

## Further reading

- [Dev Containers documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [devcontainer.json reference](https://containers.dev/)
- [Docker documentation](https://docs.docker.com/)
