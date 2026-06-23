// Package git provides operations for inspecting git repository state.
package git

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// FileStatus categorizes changed files by their git status.
type FileStatus struct {
	Modified  []string
	Untracked []string
	Staged    []string
}

// runGit executes a git command in the specified directory.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

// GetBranch returns the current branch name.
func GetBranch(ctx context.Context, dir string) (string, error) {
	out, err := runGit(ctx, dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		// detached HEAD
		out, err = runGit(ctx, dir, "rev-parse", "--short", "HEAD")
		if err != nil {
			return "HEAD", nil
		}
		return strings.TrimSpace(out), nil
	}
	return branch, nil
}

// GetStatus parses git status and returns categorized file lists.
func GetStatus(ctx context.Context, dir string) (FileStatus, error) {
	out, err := runGit(ctx, dir, "status", "--porcelain=v1")
	if err != nil {
		return FileStatus{}, err
	}

	var fs FileStatus
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
func GetUpstream(ctx context.Context, dir string) (ahead, behind int, noUpstream bool, err error) {
	out, err := runGit(ctx, dir, "rev-list", "--left-right", "--count", "@{u}...HEAD")
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

// GetDefaultBranch returns the remote default ref (e.g. "origin/main")
// resolved from origin/HEAD, or an error if no origin/HEAD is set.
func GetDefaultBranch(ctx context.Context, dir string) (string, error) {
	out, err := runGit(ctx, dir, "rev-parse", "--abbrev-ref", "origin/HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetDivergence returns how far HEAD is ahead of and behind base, computed as
// `git rev-list --left-right --count base...HEAD` (left=base→behind,
// right=HEAD→ahead), matching the convention used by GetUpstream.
func GetDivergence(ctx context.Context, dir, base string) (ahead, behind int, err error) {
	out, err := runGit(ctx, dir, "rev-list", "--left-right", "--count", base+"...HEAD")
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", out)
	}
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind, nil
}

// GetWorktree reports the absolute shared git directory and whether dir is a
// linked worktree. The common dir (`git rev-parse --git-common-dir`) is shared
// by every checkout of a repository, so it serves as a grouping key; the
// primary checkout is the one whose git dir equals the common dir, while linked
// worktrees have a git dir under <common>/worktrees/<name>.
//
// Verified against git 2.54: for the primary checkout both paths are
// "<repo>/.git"; for a linked worktree the git dir is
// "<repo>/.git/worktrees/<name>" while the common dir stays "<repo>/.git".
func GetWorktree(ctx context.Context, dir string) (commonDir string, isLinked bool, err error) {
	out, err := runGit(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir", "--git-dir")
	if err != nil {
		return "", false, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		// Older git without --path-format prints a single value; treat as primary.
		return strings.TrimSpace(out), false, nil
	}
	common := strings.TrimSpace(lines[0])
	gitDir := strings.TrimSpace(lines[1])
	return common, common != gitDir, nil
}

// GetRepoName extracts the repository name from remote URL or directory.
func GetRepoName(ctx context.Context, dir string) string {
	// try origin remote first
	out, err := runGit(ctx, dir, "remote", "get-url", "origin")
	if err == nil {
		if name := parseRemoteURL(strings.TrimSpace(out)); name != "" {
			return name
		}
	}

	// try first available remote
	out, err = runGit(ctx, dir, "remote")
	if err == nil {
		remotes := strings.Fields(out)
		for _, remote := range remotes {
			out, err = runGit(ctx, dir, "remote", "get-url", remote)
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

// Fetch updates remote-tracking refs from origin and prunes deleted ones.
func Fetch(ctx context.Context, dir string) error {
	_, err := runGit(ctx, dir, "fetch", "--prune", "origin")
	return err
}

// LocalBranches returns the short names of all local branches.
func LocalBranches(ctx context.Context, dir string) ([]string, error) {
	out, err := runGit(ctx, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

// RemoteBranches returns the short names of branches on origin, stripped of the
// "origin/" prefix and excluding the symbolic origin/HEAD.
func RemoteBranches(ctx context.Context, dir string) ([]string, error) {
	out, err := runGit(ctx, dir, "for-each-ref", "--format=%(refname:short)", "refs/remotes/origin/")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range nonEmptyLines(out) {
		name := strings.TrimPrefix(line, "origin/")
		if name == "HEAD" {
			continue
		}
		branches = append(branches, name)
	}
	return branches, nil
}

// WorktreeBranches maps each branch checked out in a worktree to its path,
// parsed from `git worktree list --porcelain`.
func WorktreeBranches(ctx context.Context, dir string) (map[string]string, error) {
	out, err := runGit(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreePorcelain(out), nil
}

// parseWorktreePorcelain extracts branch→path pairs from the porcelain stream,
// where each worktree block is a "worktree <path>" line optionally followed by
// a "branch refs/heads/<name>" line.
func parseWorktreePorcelain(out string) map[string]string {
	branches := make(map[string]string)
	var path string
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			branches[strings.TrimPrefix(ref, "refs/heads/")] = path
		}
	}
	return branches
}

// nonEmptyLines splits s on newlines and returns the trimmed, non-empty lines.
func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
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
