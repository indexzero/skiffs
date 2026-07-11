package git

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseStatusV2(t *testing.T) {
	// Captured verbatim from `git status --porcelain=v2 --branch` on a repo with
	// a staged add+modify, a worktree-only modification, a staged rename, a
	// staged add, and an untracked directory — with no upstream configured.
	out := strings.Join([]string{
		"# branch.oid 20ca136ec8ed22f1a93d930e6a793b92d7e0fc3f",
		"# branch.head main",
		"1 AM N... 000000 100644 100644 0000000000000000000000000000000000000000 0a0a4e34 both.txt",
		"1 .M N... 100644 100644 100644 5626abf0 5626abf0 committed.txt",
		"2 R. N... 100644 100644 100644 4286f428 4286f428 R100 renamed.txt\ttorename.txt",
		"1 A. N... 000000 100644 100644 0000000000000000000000000000000000000000 91d55237 staged_new.txt",
		"? sub/",
		"",
	}, "\n")

	st := parseStatusV2(out)

	if st.Branch != "main" {
		t.Errorf("Branch = %q, want main", st.Branch)
	}
	if !st.NoUpstream {
		t.Errorf("NoUpstream = false, want true (no branch.upstream header)")
	}
	assertSet(t, "Staged", st.Files.Staged, []string{"both.txt", "renamed.txt", "staged_new.txt"})
	assertSet(t, "Modified", st.Files.Modified, []string{"both.txt", "committed.txt"})
	assertSet(t, "Untracked", st.Files.Untracked, []string{"sub/"})
}

func TestParseStatusV2_Upstream(t *testing.T) {
	out := strings.Join([]string{
		"# branch.oid abc123",
		"# branch.head feat/x",
		"# branch.upstream origin/feat/x",
		"# branch.ab +3 -2",
		"",
	}, "\n")

	st := parseStatusV2(out)
	if st.NoUpstream {
		t.Error("NoUpstream = true, want false")
	}
	if st.Ahead != 3 || st.Behind != 2 {
		t.Errorf("ahead/behind = %d/%d, want 3/2", st.Ahead, st.Behind)
	}
}

func TestParseStatusV2_Detached(t *testing.T) {
	out := strings.Join([]string{
		"# branch.oid 20ca136ec8ed22f1a93d930e6a793b92d7e0fc3f",
		"# branch.head (detached)",
		"",
	}, "\n")

	st := parseStatusV2(out)
	if st.Branch != "20ca136e" {
		t.Errorf("Branch = %q, want short oid 20ca136e", st.Branch)
	}
}

func assertSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	gm := make(map[string]bool, len(got))
	for _, g := range got {
		gm[g] = true
	}
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", label, got, want)
		return
	}
	for _, w := range want {
		if !gm[w] {
			t.Errorf("%s = %v, missing %q", label, got, w)
		}
	}
}

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
