package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepos_FindsGitDirs(t *testing.T) {
	root := t.TempDir()

	// create repo1/.git
	repo1 := filepath.Join(root, "repo1")
	if err := os.MkdirAll(filepath.Join(repo1, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// create nested repo2/.git
	repo2 := filepath.Join(root, "sub", "repo2")
	if err := os.MkdirAll(filepath.Join(repo2, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	s := New(nil)
	repos, warnings := s.FindRepos([]string{root})

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d: %v", len(repos), repos)
	}
}

func TestFindRepos_RespectsIgnorePatterns(t *testing.T) {
	root := t.TempDir()

	// repo1 should be found
	repo1 := filepath.Join(root, "repo1")
	if err := os.MkdirAll(filepath.Join(repo1, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// repo in node_modules should be ignored
	ignored := filepath.Join(root, "node_modules", "pkg")
	if err := os.MkdirAll(filepath.Join(ignored, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	s := New([]string{"**/node_modules/**"})
	repos, _ := s.FindRepos([]string{root})

	if len(repos) != 1 {
		t.Errorf("expected 1 repo, got %d: %v", len(repos), repos)
	}
	if len(repos) > 0 && repos[0] != repo1 {
		t.Errorf("expected %s, got %s", repo1, repos[0])
	}
}

func TestIsGitRepo(t *testing.T) {
	root := t.TempDir()

	// directory with .git dir
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// directory without .git
	notRepo := filepath.Join(root, "notrepo")
	if err := os.MkdirAll(notRepo, 0755); err != nil {
		t.Fatal(err)
	}

	if !isGitRepo(repo) {
		t.Error("expected repo to be detected as git repo")
	}
	if isGitRepo(notRepo) {
		t.Error("expected notrepo to not be detected as git repo")
	}
}
