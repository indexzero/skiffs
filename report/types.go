// Package report defines data types for scan results.
package report

import "time"

// FileStatus categorizes changed files by their git status.
type FileStatus struct {
	Modified  []string `json:"modified"`
	Untracked []string `json:"untracked"`
	Staged    []string `json:"staged"`
}

// Count returns the total number of changed files.
func (f FileStatus) Count() int {
	return len(f.Modified) + len(f.Untracked) + len(f.Staged)
}

// RepoStatus describes the state of a single git repository.
type RepoStatus struct {
	Path       string     `json:"path"`
	Name       string     `json:"name"`
	Branch     string     `json:"branch"`
	Files      FileStatus `json:"files"`
	Ahead      int        `json:"ahead"`
	Behind     int        `json:"behind"`
	NoUpstream bool       `json:"noUpstream,omitempty"`
}

// IsDirty returns true if the repo has uncommitted changes or is ahead/behind.
func (r RepoStatus) IsDirty() bool {
	return r.Files.Count() > 0 || r.Ahead > 0 || r.Behind > 0
}

// Summary provides aggregate statistics for a scan.
type Summary struct {
	Total int `json:"total"`
	Dirty int `json:"dirty"`
	Clean int `json:"clean"`
}

// ScanResult holds the complete output of a scan operation.
type ScanResult struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Repos       []RepoStatus `json:"repos"`
	Summary     Summary      `json:"summary"`
	Warnings    []string     `json:"warnings,omitempty"`
}
