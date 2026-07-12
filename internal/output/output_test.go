package output

import (
	"testing"

	"github.com/indexzero/skiffs/report"
)

func TestGroupRepos(t *testing.T) {
	repos := []report.RepoStatus{
		{Path: "/a", Name: "alpha"},                                                 // standalone
		{Path: "/proj/wt-b", Name: "proj", CommonDir: "/proj/.git", IsWorktree: true},
		{Path: "/proj", Name: "proj", CommonDir: "/proj/.git", IsWorktree: false},   // primary
		{Path: "/proj/wt-a", Name: "proj", CommonDir: "/proj/.git", IsWorktree: true},
		{Path: "/z", Name: "zeta"},                                                  // standalone
	}

	units := groupRepos(repos)

	if len(units) != 3 {
		t.Fatalf("got %d units, want 3", len(units))
	}

	// Singletons stay singletons, in first-seen order.
	if len(units[0]) != 1 || units[0][0].Name != "alpha" {
		t.Errorf("unit[0] = %+v, want single alpha", units[0])
	}
	if len(units[2]) != 1 || units[2][0].Name != "zeta" {
		t.Errorf("unit[2] = %+v, want single zeta", units[2])
	}

	// The proj group holds all three checkouts, primary first, then linked by path.
	group := units[1]
	if len(group) != 3 {
		t.Fatalf("proj group has %d rows, want 3", len(group))
	}
	if group[0].IsWorktree || group[0].Path != "/proj" {
		t.Errorf("group[0] = %+v, want primary /proj first", group[0])
	}
	if group[1].Path != "/proj/wt-a" || group[2].Path != "/proj/wt-b" {
		t.Errorf("linked worktrees out of order: %s, %s", group[1].Path, group[2].Path)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this is a longer string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
