package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// makefileTemplate is a Makefile with shell lint and format targets for a
// chezmoi-recipes project. %s is replaced with the relative path to the
// recipes directory. Recipe lines use real tab characters as required by make.
// %% and \n in the awk command are intentional: %% becomes % after fmt.Sprintf,
// and \n is literal backslash-n that awk interprets as a newline in printf.
const makefileTemplate = `SHELL_FILES := $(shell find %s \( -name "*.sh" -o -name "*.sh.tmpl" -o -name "*.bash" \) 2>/dev/null | sort)

.DEFAULT_GOAL := help

.PHONY: help shell-fmt shell-fmt-check shell-lint check

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  make %%-18s %%s\n", $$1, $$2}'

shell-fmt: ## Format shell scripts (shfmt -w)
	shfmt -w $(SHELL_FILES)

shell-fmt-check: ## Check shell formatting without modifying (shfmt -d)
	shfmt -d $(SHELL_FILES)

shell-lint: ## Lint shell scripts (shellcheck)
	shellcheck $(SHELL_FILES)

check: shell-fmt-check shell-lint ## Run shell formatting check and shellcheck
`

// EnsureMakefile creates a Makefile with shell lint and format targets in
// projectDir, if one does not already exist. It returns true if the file was
// created, false if it was already present.
//
// absRecipesDir must be an absolute path. The Makefile uses the path of the
// recipes directory relative to projectDir in the generated find command.
func EnsureMakefile(projectDir, absRecipesDir string) (bool, error) {
	makefilePath := filepath.Join(projectDir, "Makefile")

	relRecipesDir, err := filepath.Rel(projectDir, absRecipesDir)
	if err != nil {
		relRecipesDir = absRecipesDir
	}

	f, err := os.OpenFile(makefilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("writing Makefile: %w", err)
	}

	if _, err := fmt.Fprintf(f, makefileTemplate, relRecipesDir); err != nil {
		_ = f.Close()
		return false, fmt.Errorf("writing Makefile: %w", err)
	}
	if err := f.Close(); err != nil {
		return false, fmt.Errorf("writing Makefile: %w", err)
	}
	return true, nil
}
