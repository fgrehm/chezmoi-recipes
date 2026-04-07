.PHONY: build test test-e2e lint fmt coverage vendor deadcode govulncheck audit install setup-hooks clean help

MODULE := github.com/fgrehm/chezmoi-recipes/cmd/chezmoi-recipes/cmd

BASE_VERSION := $(shell cat VERSION 2>/dev/null || echo "0.0.0")
GIT_TAG := $(shell git describe --exact-match --tags 2>/dev/null)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ifeq ($(GIT_TAG),)
  VERSION := $(BASE_VERSION)-dev+$(shell date -u +"%Y%m%d%H%M%S")
else
  VERSION := $(patsubst v%,%,$(GIT_TAG))
endif

LDFLAGS := -X $(MODULE).version=$(VERSION) \
           -X $(MODULE).commit=$(COMMIT) \
           -X $(MODULE).date=$(DATE)


help: ## show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## compile to dist/chezmoi-recipes
	@mkdir -p dist
	@go build -ldflags "$(LDFLAGS)" -o dist/chezmoi-recipes ./cmd/chezmoi-recipes
	@echo "Built to dist/chezmoi-recipes"

test: ## run unit tests (-race -shuffle=on)
	@go test -race -shuffle=on ./...

test-e2e: ## run e2e tests with bats (requires CHEZMOI_RECIPES_E2E=1)
	bats test/e2e/

lint: ## run golangci-lint
	@go tool golangci-lint run ./...

fmt: ## format with gofumpt/goimports
	@go tool golangci-lint fmt ./...

coverage: ## generate HTML coverage report
	go test -race -shuffle=on -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

vendor: ## tidy and vendor dependencies
	go mod tidy
	go mod vendor

deadcode: ## check for unreachable functions
	@out=$$(go tool deadcode ./...); \
	if [ -n "$$out" ]; then \
		echo "Unreachable functions detected:"; \
		echo "$$out"; \
		exit 1; \
	fi; \
	echo "No dead code found."

govulncheck: ## run vulnerability check
	@go tool govulncheck ./...

audit: ## run complexity and vulnerability checks (informational)
	@echo "=== Cyclomatic complexity (>15) ==="
	@go tool gocyclo -over 15 -ignore 'vendor/' . || true
	@echo ""
	@echo "=== Vulnerability check ==="
	@go tool govulncheck ./... || true

install: build ## build and symlink to ~/.local/bin
	@mkdir -p "$(HOME)/.local/bin"
	@ln -sf "$(CURDIR)/dist/chezmoi-recipes" "$(HOME)/.local/bin/chezmoi-recipes"
	@echo "Installed to ~/.local/bin/chezmoi-recipes"

setup-hooks: ## configure .githooks/ pre-commit hook
	@git config core.hooksPath .githooks
	@chmod +x .githooks/*
	@echo "Git hooks configured"

clean: ## remove build artifacts
	rm -rf dist/ coverage.out coverage.html
