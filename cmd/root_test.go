package cmd

import (
	"testing"

	"github.com/indexzero/skiffs/report"
)

func TestFilterRepos(t *testing.T) {
	repos := []report.RepoStatus{
		{Name: "clean", Files: report.FileStatus{}},
		{Name: "uncommitted", Files: report.FileStatus{Modified: []string{"a.go"}}},
		{Name: "ahead", Ahead: 2},
		{Name: "behind", Behind: 1},
		{Name: "dirty", Files: report.FileStatus{Untracked: []string{"b.go"}}, Ahead: 1},
	}

	tests := []struct {
		filter string
		want   int
	}{
		{"all", 5},
		{"dirty", 4},
		{"uncommitted", 2},
		{"unpushed", 2},
		{"unpulled", 1},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			got := filterRepos(repos, tt.filter)
			if len(got) != tt.want {
				t.Errorf("filterRepos(_, %q) returned %d repos, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestCountDirty(t *testing.T) {
	repos := []report.RepoStatus{
		{Name: "clean"},
		{Name: "uncommitted", Files: report.FileStatus{Modified: []string{"a"}}},
		{Name: "ahead", Ahead: 1},
		{Name: "also-clean"},
	}

	got := countDirty(repos)
	want := 2
	if got != want {
		t.Errorf("countDirty() = %d, want %d", got, want)
	}
}
