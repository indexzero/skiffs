package report

import "testing"

func TestAnalyzeBranches(t *testing.T) {
	local := []string{"main", "feat/merged", "feat/wip", "fix/closed", "feat/in-wt"}
	remote := []string{"main"} // main still on remote -> skipped
	mergedClosedPR := map[string]bool{
		"feat/merged": true,
		"fix/closed":  true,
		"feat/in-wt":  true,
	}
	worktrees := map[string]string{
		"feat/in-wt": "/checkouts/in-wt",
	}

	got := AnalyzeBranches(local, remote, mergedClosedPR, worktrees)

	// AreSafe: branches not on remote that match a merged/closed PR, sorted.
	wantSafe := []BranchSafety{
		{Name: "feat/in-wt", WorktreePath: "/checkouts/in-wt", Safe: true},
		{Name: "feat/merged", Safe: true},
		{Name: "fix/closed", Safe: true},
	}
	if !branchSlicesEqual(got.AreSafe, wantSafe) {
		t.Errorf("AreSafe = %+v, want %+v", got.AreSafe, wantSafe)
	}

	// MaybeSafe: not on remote, no matching PR.
	wantMaybe := []BranchSafety{{Name: "feat/wip"}}
	if !branchSlicesEqual(got.MaybeSafe, wantMaybe) {
		t.Errorf("MaybeSafe = %+v, want %+v", got.MaybeSafe, wantMaybe)
	}
}

func TestAnalyzeBranches_AllOnRemote(t *testing.T) {
	got := AnalyzeBranches(
		[]string{"main", "develop"},
		[]string{"main", "develop"},
		map[string]bool{},
		map[string]string{},
	)
	if len(got.AreSafe) != 0 || len(got.MaybeSafe) != 0 {
		t.Errorf("expected nothing prunable, got %+v", got)
	}
}

func branchSlicesEqual(a, b []BranchSafety) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
