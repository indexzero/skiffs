package report

// RepoDelta shows what changed for a single repo between scans.
type RepoDelta struct {
	Name string `json:"name"`
	Path string `json:"path"`

	// Was holds the state in the old scan (nil if repo was added)
	Was *RepoSnapshot `json:"was,omitempty"`

	// Now holds the state in the new scan (nil if repo was removed)
	Now *RepoSnapshot `json:"now,omitempty"`

	// Delta shows the numeric changes
	Delta DeltaNumbers `json:"delta,omitempty"`
}

// RepoSnapshot is a simplified view of repo state for diffing.
type RepoSnapshot struct {
	Branch       string `json:"branch"`
	Dirty        bool   `json:"dirty"`
	Uncommitted  int    `json:"uncommitted"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
	NoUpstream   bool   `json:"noUpstream,omitempty"`
}

// DeltaNumbers shows numeric changes between scans.
type DeltaNumbers struct {
	Uncommitted int `json:"uncommitted,omitempty"`
	Ahead       int `json:"ahead,omitempty"`
	Behind      int `json:"behind,omitempty"`
}

// DiffSummary aggregates high-level changes.
type DiffSummary struct {
	ReposCleaned int `json:"reposCleaned"`
	ReposDirtied int `json:"reposDirtied"`
	ReposAdded   int `json:"reposAdded"`
	ReposRemoved int `json:"reposRemoved"`
}

// DiffResult holds the comparison between two scans.
type DiffResult struct {
	OldScan  string       `json:"oldScan"`
	NewScan  string       `json:"newScan"`
	Added    []RepoDelta  `json:"added,omitempty"`
	Removed  []RepoDelta  `json:"removed,omitempty"`
	Changed  []RepoDelta  `json:"changed,omitempty"`
	Summary  DiffSummary  `json:"summary"`
}

// snapshotFromStatus creates a RepoSnapshot from a RepoStatus.
func snapshotFromStatus(r RepoStatus) RepoSnapshot {
	return RepoSnapshot{
		Branch:      r.Branch,
		Dirty:       r.IsDirty(),
		Uncommitted: r.Files.Count(),
		Ahead:       r.Ahead,
		Behind:      r.Behind,
		NoUpstream:  r.NoUpstream,
	}
}

// CompareScanResults computes the diff between old and new scan results.
func CompareScanResults(oldScan, newScan ScanResult, oldName, newName string) DiffResult {
	// build maps by path for quick lookup
	oldMap := make(map[string]RepoStatus)
	newMap := make(map[string]RepoStatus)

	for _, r := range oldScan.Repos {
		oldMap[r.Path] = r
	}
	for _, r := range newScan.Repos {
		newMap[r.Path] = r
	}

	var added, removed, changed []RepoDelta
	var cleaned, dirtied int

	// find added and changed repos
	for path, newRepo := range newMap {
		if oldRepo, exists := oldMap[path]; exists {
			// repo exists in both - check if changed
			oldSnap := snapshotFromStatus(oldRepo)
			newSnap := snapshotFromStatus(newRepo)

			if hasChanged(oldSnap, newSnap) {
				delta := RepoDelta{
					Name: newRepo.Name,
					Path: path,
					Was:  &oldSnap,
					Now:  &newSnap,
					Delta: DeltaNumbers{
						Uncommitted: newSnap.Uncommitted - oldSnap.Uncommitted,
						Ahead:       newSnap.Ahead - oldSnap.Ahead,
						Behind:      newSnap.Behind - oldSnap.Behind,
					},
				}
				changed = append(changed, delta)

				// track clean/dirty transitions
				if oldSnap.Dirty && !newSnap.Dirty {
					cleaned++
				} else if !oldSnap.Dirty && newSnap.Dirty {
					dirtied++
				}
			}
		} else {
			// repo added
			snap := snapshotFromStatus(newRepo)
			added = append(added, RepoDelta{
				Name: newRepo.Name,
				Path: path,
				Now:  &snap,
			})
			if snap.Dirty {
				dirtied++
			}
		}
	}

	// find removed repos
	for path, oldRepo := range oldMap {
		if _, exists := newMap[path]; !exists {
			snap := snapshotFromStatus(oldRepo)
			removed = append(removed, RepoDelta{
				Name: oldRepo.Name,
				Path: path,
				Was:  &snap,
			})
			if snap.Dirty {
				cleaned++ // removing a dirty repo is like cleaning it
			}
		}
	}

	return DiffResult{
		OldScan: oldName,
		NewScan: newName,
		Added:   added,
		Removed: removed,
		Changed: changed,
		Summary: DiffSummary{
			ReposCleaned: cleaned,
			ReposDirtied: dirtied,
			ReposAdded:   len(added),
			ReposRemoved: len(removed),
		},
	}
}

// hasChanged checks if the repo state changed between snapshots.
func hasChanged(old, new RepoSnapshot) bool {
	return old.Branch != new.Branch ||
		old.Dirty != new.Dirty ||
		old.Uncommitted != new.Uncommitted ||
		old.Ahead != new.Ahead ||
		old.Behind != new.Behind
}
