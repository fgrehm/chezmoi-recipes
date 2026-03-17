.PHONY: build test test-e2e vet fmt lint install help

INSTALL_DIR ?= $(HOME)/.local/bin

VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/fgrehm/chezmoi-recipes/cmd/chezmoi-recipes/cmd.version=$(VERSION) \
           -X github.com/fgrehm/chezmoi-recipes/cmd/chezmoi-recipes/cmd.commit=$(COMMIT) \
           -X github.com/fgrehm/chezmoi-recipes/cmd/chezmoi-recipes/cmd.date=$(DATE)

help: ## show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*##/ { printf "  %-10s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## compile the binary to bin/chezmoi-recipes
	go build -ldflags "$(LDFLAGS)" -o bin/chezmoi-recipes ./cmd/chezmoi-recipes

install: build ## build and symlink binary to $$INSTALL_DIR (default: $(INSTALL_DIR))
	mkdir -p $(INSTALL_DIR)
	ln -sf $(CURDIR)/bin/chezmoi-recipes $(INSTALL_DIR)/chezmoi-recipes

test: ## run unit tests with race detector
	go test -race ./...

test-e2e: ## run e2e tests with bats (requires container or CHEZMOI_RECIPES_E2E=1)
	bats test/e2e/

vet: ## run go vet
	go vet ./...

fmt: ## format source with gofmt
	gofmt -w .

lint: ## run golangci-lint
	golangci-lint run ./...
