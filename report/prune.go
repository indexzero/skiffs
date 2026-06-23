package report

import "sort"

// BranchSafety classifies one local branch as a pruning candidate.
type BranchSafety struct {
	Name string `json:"name"`
	// WorktreePath is the worktree a branch is checked out in, if any. Such a
	// branch cannot be deleted until the worktree is removed.
	WorktreePath string `json:"worktreePath,omitempty"`
	// Safe is true when the branch matches a merged or closed PR (ARESAFE);
	// false means no matching PR was found (MAYBESAFE — review before deleting).
	Safe bool `json:"safe"`
}

// PruneResult is the categorized output of branch-prune analysis.
type PruneResult struct {
	AreSafe   []BranchSafety `json:"areSafe"`
	MaybeSafe []BranchSafety `json:"maybeSafe"`
}

// AnalyzeBranches categorizes local branches that no longer exist on the remote:
//
//   - AreSafe:   the branch name matches a merged or closed PR — safe to delete.
//   - MaybeSafe: no matching PR was found — review before deleting.
//
// Branches still present on the remote are skipped entirely. A branch checked
// out in a worktree keeps its classification but carries the worktree path, so
// the caller can surface that it cannot be deleted in place. This mirrors the
// bonsai pruning heuristic: local ∖ remote, intersected with merged/closed PRs.
func AnalyzeBranches(local, remote []string, mergedClosedPR map[string]bool, worktrees map[string]string) PruneResult {
	remoteSet := make(map[string]bool, len(remote))
	for _, b := range remote {
		remoteSet[b] = true
	}

	var result PruneResult
	for _, b := range local {
		if remoteSet[b] {
			continue
		}
		entry := BranchSafety{
			Name:         b,
			WorktreePath: worktrees[b],
			Safe:         mergedClosedPR[b],
		}
		if entry.Safe {
			result.AreSafe = append(result.AreSafe, entry)
		} else {
			result.MaybeSafe = append(result.MaybeSafe, entry)
		}
	}

	sortBySafety := func(s []BranchSafety) {
		sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	}
	sortBySafety(result.AreSafe)
	sortBySafety(result.MaybeSafe)
	return result
}
