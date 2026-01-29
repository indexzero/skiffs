package report

import (
	"testing"
	"time"
)

func TestCompareScanResults(t *testing.T) {
	now := time.Now()

	oldScan := ScanResult{
		GeneratedAt: now,
		Repos: []RepoStatus{
			{Path: "/a", Name: "a", Branch: "main", Files: FileStatus{Modified: []string{"x.go"}}},
			{Path: "/b", Name: "b", Branch: "main"},
		},
	}

	newScan := ScanResult{
		GeneratedAt: now,
		Repos: []RepoStatus{
			{Path: "/a", Name: "a", Branch: "main"}, // cleaned
			{Path: "/c", Name: "c", Branch: "dev"},  // added
		},
	}

	diff := CompareScanResults(oldScan, newScan, "old.json", "new.json")

	// check added
	if len(diff.Added) != 1 || diff.Added[0].Name != "c" {
		t.Errorf("expected 1 added repo 'c', got %v", diff.Added)
	}

	// check removed
	if len(diff.Removed) != 1 || diff.Removed[0].Name != "b" {
		t.Errorf("expected 1 removed repo 'b', got %v", diff.Removed)
	}

	// check changed
	if len(diff.Changed) != 1 || diff.Changed[0].Name != "a" {
		t.Errorf("expected 1 changed repo 'a', got %v", diff.Changed)
	}

	// check summary
	if diff.Summary.ReposCleaned != 1 {
		t.Errorf("expected 1 repo cleaned, got %d", diff.Summary.ReposCleaned)
	}
	if diff.Summary.ReposAdded != 1 {
		t.Errorf("expected 1 repo added, got %d", diff.Summary.ReposAdded)
	}
	if diff.Summary.ReposRemoved != 1 {
		t.Errorf("expected 1 repo removed, got %d", diff.Summary.ReposRemoved)
	}
}

func TestHasChanged(t *testing.T) {
	base := RepoSnapshot{Branch: "main", Dirty: false, Uncommitted: 0}

	tests := []struct {
		name string
		old  RepoSnapshot
		new  RepoSnapshot
		want bool
	}{
		{"no change", base, base, false},
		{"branch changed", base, RepoSnapshot{Branch: "dev", Dirty: false}, true},
		{"dirty changed", base, RepoSnapshot{Branch: "main", Dirty: true, Uncommitted: 1}, true},
		{"ahead changed", base, RepoSnapshot{Branch: "main", Ahead: 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasChanged(tt.old, tt.new); got != tt.want {
				t.Errorf("hasChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
