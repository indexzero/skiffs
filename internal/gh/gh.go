// Package gh queries GitHub through the gh CLI.
package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type pullRequest struct {
	HeadRefName string `json:"headRefName"`
	State       string `json:"state"`
}

// Available reports whether the gh CLI is installed and authenticated.
func Available(ctx context.Context) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found: %w", err)
	}
	if err := exec.CommandContext(ctx, "gh", "auth", "status").Run(); err != nil {
		return fmt.Errorf("gh not authenticated (run 'gh auth login')")
	}
	return nil
}

// MergedClosedPRBranches returns the set of head branch names for pull requests
// in the repository at dir that are merged or closed (no longer open). It shells
// out to `gh pr list`, so gh must be installed and authenticated.
func MergedClosedPRBranches(ctx context.Context, dir string, limit int) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "list",
		"--state", "all",
		"--json", "headRefName,state",
		"--limit", strconv.Itoa(limit))
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	return parseMergedClosed(stdout.Bytes())
}

// parseMergedClosed extracts the head branch names of non-open PRs from the
// `gh pr list --json headRefName,state` payload.
func parseMergedClosed(data []byte) (map[string]bool, error) {
	var prs []pullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	branches := make(map[string]bool)
	for _, pr := range prs {
		if pr.State == "MERGED" || pr.State == "CLOSED" {
			branches[pr.HeadRefName] = true
		}
	}
	return branches, nil
}
