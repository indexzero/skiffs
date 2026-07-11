package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/indexzero/skiffs/internal/gh"
	"github.com/indexzero/skiffs/internal/git"
	"github.com/indexzero/skiffs/internal/output"
	"github.com/indexzero/skiffs/report"
	"github.com/spf13/cobra"
)

// prPageLimit caps how many PRs gh returns; matches bonsai's default so large
// repositories still surface every merged/closed branch.
const prPageLimit = 10000

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Find local branches that are safe to delete",
	Long: "prune analyzes the local branches of a single repository and classifies " +
		"those no longer on the remote: safe to delete (a merged or closed PR matches " +
		"the branch) and needs review (no matching PR). It is a Go port of bonsai.",
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().StringP("root", "r", ".", "repository directory to analyze")
	pruneCmd.Flags().StringP("output", "o", "table", "output format: table|json")
	pruneCmd.Flags().Bool("no-fetch", false, "skip 'git fetch --prune origin' before analyzing")
	rootCmd.AddCommand(pruneCmd)
}

func runPrune(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	dir := mustGetString(cmd, "root")
	outputFmt := mustGetString(cmd, "output")
	noFetch, _ := cmd.Flags().GetBool("no-fetch")

	if err := gh.Available(ctx); err != nil {
		return err
	}

	if !noFetch {
		if err := git.Fetch(ctx, dir); err != nil {
			return fmt.Errorf("fetch: %w", err)
		}
	}

	result, err := analyzePrune(ctx, dir)
	if err != nil {
		return err
	}

	if outputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	writePruneTable(os.Stdout, dir, result)
	return nil
}

func analyzePrune(ctx context.Context, dir string) (report.PruneResult, error) {
	local, err := git.LocalBranches(ctx, dir)
	if err != nil {
		return report.PruneResult{}, fmt.Errorf("listing local branches: %w", err)
	}
	remote, err := git.RemoteBranches(ctx, dir)
	if err != nil {
		return report.PruneResult{}, fmt.Errorf("listing remote branches: %w", err)
	}
	worktrees, err := git.WorktreeBranches(ctx, dir)
	if err != nil {
		return report.PruneResult{}, fmt.Errorf("listing worktrees: %w", err)
	}
	prBranches, err := gh.MergedClosedPRBranches(ctx, dir, prPageLimit)
	if err != nil {
		return report.PruneResult{}, fmt.Errorf("querying pull requests: %w", err)
	}
	return report.AnalyzeBranches(local, remote, prBranches, worktrees), nil
}

func writePruneTable(w *os.File, dir string, r report.PruneResult) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", output.Bold("Prune Report"))
	fmt.Fprintf(w, "%s %s\n\n", output.Gray("repo:"), dir)

	// Safe-to-delete costs little attention, so dim the rows.
	fmt.Fprintf(w, "%s %s\n",
		output.Green("Safe to delete"),
		output.Dim("(not on remote + merged/closed PR)"))
	writePruneSection(w, r.SafeToDelete, output.Dim("✓"))

	fmt.Fprintln(w)

	// Needs-review should draw the eye, so render it in yellow.
	fmt.Fprintf(w, "%s %s\n",
		output.Yellow("Needs review"),
		output.Dim("(not on remote + no matching PR)"))
	writePruneSection(w, r.NeedsReview, output.Yellow("?"))

	if deletable := deletableNames(r.SafeToDelete); len(deletable) > 0 {
		fmt.Fprintf(w, "\n%s\n", output.Bold("To delete safe branches:"))
		fmt.Fprintf(w, "  %s\n", output.Cyan("git branch -D %s", strings.Join(deletable, " ")))
	}
}

func writePruneSection(w *os.File, branches []report.BranchSafety, marker string) {
	if len(branches) == 0 {
		fmt.Fprintf(w, "  %s\n", output.Dim("(none)"))
		return
	}
	for _, b := range branches {
		if b.WorktreePath != "" {
			continue
		}
		fmt.Fprintf(w, "  %s %s\n", marker, output.Dim(b.Name))
	}
	// Worktree-held branches can't be deleted in place; call them out separately.
	var held []report.BranchSafety
	for _, b := range branches {
		if b.WorktreePath != "" {
			held = append(held, b)
		}
	}
	if len(held) > 0 {
		fmt.Fprintf(w, "  %s\n", output.Yellow("in worktrees (remove the worktree first):"))
		for _, b := range held {
			fmt.Fprintf(w, "    %s %s %s\n", marker, b.Name, output.Dim("→ %s", b.WorktreePath))
		}
	}
}

// deletableNames returns the safe-to-delete branches not pinned by a worktree,
// i.e. the ones the suggested `git branch -D` command can remove immediately.
func deletableNames(branches []report.BranchSafety) []string {
	var names []string
	for _, b := range branches {
		if b.WorktreePath == "" {
			names = append(names, b.Name)
		}
	}
	return names
}
