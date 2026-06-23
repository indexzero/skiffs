package git

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseWorktreePorcelain(t *testing.T) {
	// Format per `git worktree list --porcelain`: the primary checkout and
	// each linked worktree are "worktree <path>" blocks; a detached worktree
	// has no "branch" line and is therefore excluded.
	out := strings.Join([]string{
		"worktree /repo",
		"HEAD abc123",
		"branch refs/heads/main",
		"",
		"worktree /repo-feat",
		"HEAD def456",
		"branch refs/heads/feat/x",
		"",
		"worktree /repo-detached",
		"HEAD 789aaa",
		"detached",
		"",
	}, "\n")

	got := parseWorktreePorcelain(out)
	want := map[string]string{
		"main":   "/repo",
		"feat/x": "/repo-feat",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseWorktreePorcelain() = %v, want %v", got, want)
	}
}

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		// HTTPS URLs
		{"https://github.com/user/repo.git", "repo"},
		{"https://github.com/user/repo", "repo"},

		// SSH URLs (scp-style)
		{"git@github.com:user/repo.git", "repo"},
		{"git@github.com:user/repo", "repo"},

		// SSH URLs (full)
		{"ssh://git@github.com/user/repo.git", "repo"},

		// GitLab/other hosts
		{"git@gitlab.com:org/project.git", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := parseRemoteURL(tt.url)
			if got != tt.want {
				t.Errorf("parseRemoteURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
