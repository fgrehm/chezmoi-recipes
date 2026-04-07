package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeploySharedScripts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()

	if err := DeploySharedScripts(sourceDir); err != nil {
		t.Fatalf("DeploySharedScripts() error = %v", err)
	}

	// Verify ui.bash was written.
	uiPath := filepath.Join(sourceDir, "scripts", "ui.bash")
	data, err := os.ReadFile(uiPath)
	if err != nil {
		t.Fatalf("ui.bash not found: %v", err)
	}
	if !strings.Contains(string(data), "log_info") {
		t.Error("ui.bash missing log_info function")
	}
	if !strings.Contains(string(data), "run_quiet") {
		t.Error("ui.bash missing run_quiet function")
	}

	// Verify .chezmoiignore contains scripts/.
	ignorePath := filepath.Join(sourceDir, ".chezmoiignore")
	ignoreData, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatalf(".chezmoiignore not found: %v", err)
	}
	if !strings.Contains(string(ignoreData), "scripts/") {
		t.Error(".chezmoiignore missing scripts/ entry")
	}
}

func TestDeploySharedScripts_Idempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()

	// Run twice.
	if err := DeploySharedScripts(sourceDir); err != nil {
		t.Fatalf("first DeploySharedScripts() error = %v", err)
	}
	if err := DeploySharedScripts(sourceDir); err != nil {
		t.Fatalf("second DeploySharedScripts() error = %v", err)
	}

	// Verify scripts/ appears only once in .chezmoiignore.
	ignorePath := filepath.Join(sourceDir, ".chezmoiignore")
	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatalf(".chezmoiignore not found: %v", err)
	}
	count := strings.Count(string(data), "scripts/")
	if count != 1 {
		t.Errorf("scripts/ appears %d times in .chezmoiignore, want 1", count)
	}
}

func TestDeployCheckVersions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	projectDir := t.TempDir()

	if err := DeployCheckVersions(projectDir); err != nil {
		t.Fatalf("DeployCheckVersions() error = %v", err)
	}

	dest := filepath.Join(projectDir, "scripts", "check-versions.sh")
	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("check-versions.sh not found: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Error("check-versions.sh is not executable")
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading check-versions.sh: %v", err)
	}
	if !strings.Contains(string(data), "fetch_latest") {
		t.Error("check-versions.sh missing fetch_latest function")
	}
}

func TestDeployCheckVersions_SkipsIfExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	projectDir := t.TempDir()
	scriptsDir := filepath.Join(projectDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "#!/bin/bash\n# custom check-versions\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "check-versions.sh"), []byte(existing), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := DeployCheckVersions(projectDir); err != nil {
		t.Fatalf("DeployCheckVersions() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(scriptsDir, "check-versions.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Error("existing check-versions.sh was overwritten")
	}
}

func TestEnsureChezmoiIgnore_AppendsToExisting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()
	ignorePath := filepath.Join(sourceDir, ".chezmoiignore")

	// Write existing content.
	if err := os.WriteFile(ignorePath, []byte("existing_entry\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureChezmoiIgnore(sourceDir, []string{"scripts/"}); err != nil {
		t.Fatalf("EnsureChezmoiIgnore() error = %v", err)
	}

	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "existing_entry") {
		t.Error("existing content was lost")
	}
	if !strings.Contains(content, "scripts/") {
		t.Error("new entry was not added")
	}
}

func TestEnsureChezmoiIgnore_SkipsDuplicates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sourceDir := t.TempDir()
	ignorePath := filepath.Join(sourceDir, ".chezmoiignore")

	if err := os.WriteFile(ignorePath, []byte("scripts/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureChezmoiIgnore(sourceDir, []string{"scripts/"}); err != nil {
		t.Fatalf("EnsureChezmoiIgnore() error = %v", err)
	}

	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(string(data), "scripts/")
	if count != 1 {
		t.Errorf("scripts/ appears %d times, want 1", count)
	}
}
