package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initTempGitRepo(t *testing.T, root string) string {
	t.Helper()

	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	repo, err := git.PlainInit(repoDir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("add file: %v", err)
	}
	_, err = worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	return repoDir
}

func TestDiscoverGitReposAndHelpers(t *testing.T) {
	root := t.TempDir()
	repoDir := initTempGitRepo(t, root)
	nested := filepath.Join(root, "nested", "inner")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	nestedRepo := initTempGitRepo(t, nested)

	repos, err := discoverGitRepos(root)
	if err != nil {
		t.Fatalf("discoverGitRepos failed: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d (%v)", len(repos), repos)
	}
	if repos[0] != repoDir && repos[1] != repoDir {
		t.Fatalf("root repo not discovered: %v", repos)
	}
	if repos[0] != nestedRepo && repos[1] != nestedRepo {
		t.Fatalf("nested repo not discovered: %v", repos)
	}

	if got := relativePath(root, repoDir); got == repoDir || !strings.Contains(got, "repo") {
		t.Fatalf("unexpected relativePath: %q", got)
	}
	if got := colorize(true, "\033[31m", "text"); !strings.Contains(got, "text") || !strings.Contains(got, "\033[31m") {
		t.Fatalf("unexpected colorize output: %q", got)
	}
	if got := truncate("abcdef", 3); got != "ab…" {
		t.Fatalf("truncate overflow = %q", got)
	}
	if got := truncate("abc", 5); got != "abc" {
		t.Fatalf("truncate short = %q", got)
	}

	out := captureStdout(func() {
		printRepoStatuses([]gitRepoStatus{{Path: "repo", Branch: "main", State: "up-to-date"}})
	})
	if !strings.Contains(out, "repo") || !strings.Contains(out, "main") {
		t.Fatalf("unexpected printRepoStatuses output: %q", out)
	}
}

func TestEvaluateRepo(t *testing.T) {
	root := t.TempDir()
	repoDir := initTempGitRepo(t, root)

	got := evaluateRepo(root, repoDir, false)
	if got == nil {
		t.Fatal("expected evaluateRepo to return status")
	}
	if got.Branch == "" || got.Path == "" {
		t.Fatalf("unexpected repo status: %+v", got)
	}
}

func TestGitListCmdRunE(t *testing.T) {
	root := t.TempDir()
	_ = initTempGitRepo(t, root)

	origPath := gitListPath
	origNoFetch := gitListNoFetch
	origJSON := gitListJSON
	origTabular := gitListTabular
	defer func() {
		gitListPath = origPath
		gitListNoFetch = origNoFetch
		gitListJSON = origJSON
		gitListTabular = origTabular
	}()

	gitListPath = root
	gitListNoFetch = true
	gitListTabular = false

	out := captureStdout(func() {
		if err := gitListCmd.RunE(gitListCmd, nil); err != nil {
			t.Fatalf("git list run failed: %v", err)
		}
	})
	if !strings.Contains(out, "repo") || !strings.Contains(out, "no-upstream") {
		t.Fatalf("unexpected git list output: %q", out)
	}

	gitListJSON = true
	out = captureStdout(func() {
		if err := gitListCmd.RunE(gitListCmd, nil); err != nil {
			t.Fatalf("git list json run failed: %v", err)
		}
	})
	var rows []gitRepoStatus
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if len(rows) != 1 || rows[0].Path == "" {
		t.Fatalf("unexpected git list json rows: %+v", rows)
	}
}
