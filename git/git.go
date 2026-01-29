// Package git provides operations for inspecting git repository state.
package git

import (
	"bytes"
	"net/url"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/indexzero/skiffscan/report"
)

// runGit executes a git command in the specified directory.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

// GetBranch returns the current branch name.
func GetBranch(dir string) (string, error) {
	out, err := runGit(dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		// detached HEAD
		out, err = runGit(dir, "rev-parse", "--short", "HEAD")
		if err != nil {
			return "HEAD", nil
		}
		return strings.TrimSpace(out), nil
	}
	return branch, nil
}

// GetStatus parses git status and returns categorized file lists.
func GetStatus(dir string) (report.FileStatus, error) {
	out, err := runGit(dir, "status", "--porcelain=v1")
	if err != nil {
		return report.FileStatus{}, err
	}

	var fs report.FileStatus
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		xy := line[:2]
		file := strings.TrimSpace(line[3:])

		// X is index status, Y is worktree status
		x, y := xy[0], xy[1]

		switch {
		case x == '?' && y == '?':
			fs.Untracked = append(fs.Untracked, file)
		case x == 'A' || x == 'M' || x == 'D' || x == 'R' || x == 'C':
			// staged changes
			fs.Staged = append(fs.Staged, file)
			// check if also modified in worktree
			if y == 'M' || y == 'D' {
				fs.Modified = append(fs.Modified, file)
			}
		case y == 'M' || y == 'D':
			fs.Modified = append(fs.Modified, file)
		}
	}
	return fs, nil
}

// GetUpstream returns ahead/behind counts relative to upstream.
// Returns noUpstream=true if no upstream is configured.
func GetUpstream(dir string) (ahead, behind int, noUpstream bool, err error) {
	out, err := runGit(dir, "rev-list", "--left-right", "--count", "@{u}...HEAD")
	if err != nil {
		// no upstream configured
		return 0, 0, true, nil
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, true, nil
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind, false, nil
}

// GetRepoName extracts the repository name from remote URL or directory.
func GetRepoName(dir string) string {
	// try origin remote first
	out, err := runGit(dir, "remote", "get-url", "origin")
	if err == nil {
		if name := parseRemoteURL(strings.TrimSpace(out)); name != "" {
			return name
		}
	}

	// try first available remote
	out, err = runGit(dir, "remote")
	if err == nil {
		remotes := strings.Fields(out)
		for _, remote := range remotes {
			out, err = runGit(dir, "remote", "get-url", remote)
			if err == nil {
				if name := parseRemoteURL(strings.TrimSpace(out)); name != "" {
					return name
				}
			}
		}
	}

	// fallback to directory name
	return filepath.Base(dir)
}

// parseRemoteURL extracts repo name from git remote URL.
func parseRemoteURL(remote string) string {
	// handle scp-style: git@github.com:user/repo.git
	if strings.Contains(remote, ":") && strings.Contains(remote, "@") && !strings.Contains(remote, "://") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) == 2 {
			remote = "ssh://" + parts[0] + "/" + parts[1]
		}
	}

	if u, err := url.Parse(remote); err == nil && u.Path != "" {
		name := path.Base(u.Path)
		return strings.TrimSuffix(name, ".git")
	}
	return ""
}
