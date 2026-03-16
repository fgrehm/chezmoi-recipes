package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindGitRoot_FindsParentDir(t *testing.T) {
	setTestEnv(t)

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	recipesDir := filepath.Join(repoDir, "recipes")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := findGitRoot(recipesDir)
	if err != nil {
		t.Fatalf("findGitRoot() error = %v", err)
	}
	if got != repoDir {
		t.Errorf("findGitRoot() = %q, want %q", got, repoDir)
	}
}

func TestFindGitRoot_FindsExactDir(t *testing.T) {
	setTestEnv(t)

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := findGitRoot(repoDir)
	if err != nil {
		t.Fatalf("findGitRoot() error = %v", err)
	}
	if got != repoDir {
		t.Errorf("findGitRoot() = %q, want %q", got, repoDir)
	}
}

func TestFindGitRoot_NoGitRepo(t *testing.T) {
	setTestEnv(t)

	dir := t.TempDir()
	recipesDir := filepath.Join(dir, "nested", "recipes")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := findGitRoot(recipesDir)
	if err == nil {
		t.Fatal("expected error when no git repo found")
	}
	if !strings.Contains(err.Error(), "no git repository found") {
		t.Errorf("error should mention 'no git repository found': %v", err)
	}
}

func TestHandlePullError_Fail(t *testing.T) {
	setTestEnv(t)

	var errW bytes.Buffer
	err := handlePullError(errors.New("network error"), "fail", &errW)
	if err == nil {
		t.Fatal("expected error for --on-error=fail")
	}
	if errW.Len() != 0 {
		t.Errorf("fail mode should not write to stderr: %s", errW.String())
	}
}

func TestHandlePullError_Warn(t *testing.T) {
	setTestEnv(t)

	var errW bytes.Buffer
	err := handlePullError(errors.New("network error"), "warn", &errW)
	if err != nil {
		t.Fatalf("warn mode should return nil, got: %v", err)
	}
	if !strings.Contains(errW.String(), "warning:") {
		t.Errorf("warn mode should write warning to stderr: %s", errW.String())
	}
}

func TestHandlePullError_Ignore(t *testing.T) {
	setTestEnv(t)

	var errW bytes.Buffer
	err := handlePullError(errors.New("network error"), "ignore", &errW)
	if err != nil {
		t.Fatalf("ignore mode should return nil, got: %v", err)
	}
	if errW.Len() != 0 {
		t.Errorf("ignore mode should not write to stderr: %s", errW.String())
	}
}

func TestRunPull_NoGitRepo_FailMode(t *testing.T) {
	setTestEnv(t)

	dir := t.TempDir()
	recipesDir := filepath.Join(dir, "recipes")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var w, errW bytes.Buffer
	err := runPull(context.Background(), recipesDir, "fail", &w, &errW)
	if err == nil {
		t.Fatal("expected error when no git repo found")
	}
}

func TestRunPull_NoGitRepo_WarnMode(t *testing.T) {
	setTestEnv(t)

	dir := t.TempDir()
	recipesDir := filepath.Join(dir, "recipes")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var w, errW bytes.Buffer
	err := runPull(context.Background(), recipesDir, "warn", &w, &errW)
	if err != nil {
		t.Fatalf("warn mode should not return error, got: %v", err)
	}
	if !strings.Contains(errW.String(), "warning:") {
		t.Errorf("warn mode should write warning: %s", errW.String())
	}
}

func TestRunPull_NoGitRepo_IgnoreMode(t *testing.T) {
	setTestEnv(t)

	dir := t.TempDir()
	recipesDir := filepath.Join(dir, "recipes")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var w, errW bytes.Buffer
	err := runPull(context.Background(), recipesDir, "ignore", &w, &errW)
	if err != nil {
		t.Fatalf("ignore mode should not return error, got: %v", err)
	}
	if errW.Len() != 0 {
		t.Errorf("ignore mode should not write to stderr: %s", errW.String())
	}
}

func TestRunPull_PullsFromRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	setTestEnv(t)

	tmp := t.TempDir()
	remoteDir := filepath.Join(tmp, "remote.git")
	dotfilesDir := filepath.Join(tmp, "dotfiles")

	// Create a bare remote and clone it.
	mustGit(t, tmp, "init", "--bare", remoteDir)
	mustGit(t, tmp, "clone", remoteDir, dotfilesDir)
	mustGit(t, dotfilesDir, "config", "user.email", "test@example.com")
	mustGit(t, dotfilesDir, "config", "user.name", "Test")

	// Initial commit + push to establish the remote branch.
	mustGit(t, dotfilesDir, "commit", "--allow-empty", "-m", "init")
	mustGit(t, dotfilesDir, "push", "--set-upstream", "origin", "HEAD")

	// Push a new commit to remote via a second clone.
	secondClone := filepath.Join(tmp, "second")
	mustGit(t, tmp, "clone", remoteDir, secondClone)
	mustGit(t, secondClone, "config", "user.email", "test@example.com")
	mustGit(t, secondClone, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(secondClone, "newfile.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, secondClone, "add", ".")
	mustGit(t, secondClone, "commit", "-m", "add newfile")
	mustGit(t, secondClone, "push")

	// runPull from the recipes subdir should pull the new commit into dotfilesDir.
	recipesDir := filepath.Join(dotfilesDir, "recipes")
	if err := os.MkdirAll(recipesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var w, errW bytes.Buffer
	if err := runPull(context.Background(), recipesDir, "fail", &w, &errW); err != nil {
		t.Fatalf("runPull() error = %v; stderr: %s", err, errW.String())
	}

	if _, err := os.Stat(filepath.Join(dotfilesDir, "newfile.txt")); err != nil {
		t.Error("newfile.txt should have been pulled into dotfiles dir")
	}
}

// mustGit runs a git command in dir, failing the test on error.
func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\noutput: %s", args, err, out)
	}
}
