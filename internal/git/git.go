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

// State is the branch, file, and upstream status of a repository, read in a
// single `git status` invocation.
type State struct {
	Branch     string
	Files      FileStatus
	Ahead      int
	Behind     int
	NoUpstream bool
}

// GetState reads branch name, upstream divergence, and file status in one
// `git status --porcelain=v2 --branch` call, replacing the three separate
// subprocesses (branch / rev-list @{u} / status) it used to take per repo.
func GetState(ctx context.Context, dir string) (State, error) {
	out, err := runGit(ctx, dir, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return State{}, err
	}
	return parseStatusV2(out), nil
}

// parseStatusV2 parses `git status --porcelain=v2 --branch` output. Header
// lines carry branch and upstream state; entry lines ('1' ordinary, '2'
// renamed/copied, 'u' unmerged, '?' untracked) carry per-file state. A repo
// with no upstream simply omits the branch.upstream / branch.ab headers, which
// is why NoUpstream defaults true until one is seen.
func parseStatusV2(out string) State {
	st := State{NoUpstream: true}
	var oid string

	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.oid "):
			oid = strings.TrimPrefix(line, "# branch.oid ")
		case strings.HasPrefix(line, "# branch.head "):
			st.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream "):
			st.NoUpstream = false
		case strings.HasPrefix(line, "# branch.ab "):
			st.NoUpstream = false
			fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(fields) == 2 {
				st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
				st.Behind, _ = strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
			}
		case strings.HasPrefix(line, "# "):
			// other header (e.g. branch.oid == (initial)); ignore
		case strings.HasPrefix(line, "? "):
			st.Files.Untracked = append(st.Files.Untracked, line[2:])
		case strings.HasPrefix(line, "! "):
			// ignored file; skip
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			parseV2Changed(line, &st.Files)
		case strings.HasPrefix(line, "u "):
			// unmerged (conflict) — a worktree change worth flagging as modified
			if p := v2Path(line, 10); p != "" {
				st.Files.Modified = append(st.Files.Modified, p)
			}
		}
	}

	// A detached checkout reports "(detached)"; fall back to a short oid to
	// preserve the previous behaviour of showing an abbreviated commit.
	if st.Branch == "(detached)" {
		st.Branch = shortOID(oid)
	}
	return st
}

// parseV2Changed classifies an ordinary ('1') or renamed/copied ('2') entry.
// The two-character <XY> field holds index (staged) and worktree status; '.'
// means unmodified. This mirrors the porcelain v1 classification exactly.
func parseV2Changed(line string, fs *FileStatus) {
	nFields := 9 // ordinary entry: 8 fixed fields then the path
	if strings.HasPrefix(line, "2 ") {
		nFields = 10 // renamed/copied entry inserts an <Xscore> field
	}
	parts := strings.SplitN(line, " ", nFields)
	if len(parts) < nFields || len(parts[1]) != 2 {
		return
	}
	xy := parts[1]
	// A rename path is "<new>\t<orig>"; the new path comes first.
	path := strings.SplitN(parts[nFields-1], "\t", 2)[0]

	if strings.IndexByte("AMDRC", xy[0]) >= 0 {
		fs.Staged = append(fs.Staged, path)
	}
	if xy[1] == 'M' || xy[1] == 'D' {
		fs.Modified = append(fs.Modified, path)
	}
}

// v2Path returns the pathname field of a space-delimited porcelain v2 entry
// that has nFields columns, stripping any "\t<orig>" rename suffix.
func v2Path(line string, nFields int) string {
	parts := strings.SplitN(line, " ", nFields)
	if len(parts) < nFields {
		return ""
	}
	return strings.SplitN(parts[nFields-1], "\t", 2)[0]
}

// shortOID abbreviates a commit id to 8 characters (or fewer if shorter).
func shortOID(oid string) string {
	if len(oid) > 8 {
		return oid[:8]
	}
	return oid
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
// right=HEAD→ahead).
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
