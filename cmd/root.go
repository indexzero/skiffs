// Package cmd provides the CLI commands for skiffs.
package cmd

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/indexzero/skiffs/internal/git"
	"github.com/indexzero/skiffs/internal/output"
	"github.com/indexzero/skiffs/internal/scanner"
	"github.com/indexzero/skiffs/report"
	"github.com/spf13/cobra"
)

// scanConfig holds configuration for a scan operation.
type scanConfig struct {
	roots      []string
	filter     string
	outputFmt  string
	workers    int
	ignorePats []string
}

var rootCmd = &cobra.Command{
	Use:   "skiffs",
	Short: "Scan directories for git repositories and report status",
	Long:  "skiffs walks directory trees to find git repositories and reports their status: uncommitted files, unpushed commits, and unpulled changes.",
	RunE:  runScan,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringArrayP("root", "r", []string{"."}, "root directories to scan")
	rootCmd.Flags().StringP("filter", "f", "dirty", "filter: all|dirty|uncommitted|unpushed|unpulled")
	rootCmd.Flags().StringP("output", "o", "table", "output format: table|json")
	rootCmd.Flags().IntP("workers", "w", 8, "concurrent git operations")
	rootCmd.Flags().StringArrayP("ignore", "i", scanner.DefaultIgnorePatterns, "glob patterns to ignore")
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg := scanConfig{
		roots:      mustGetStringArray(cmd, "root"),
		filter:     mustGetString(cmd, "filter"),
		outputFmt:  mustGetString(cmd, "output"),
		workers:    mustGetInt(cmd, "workers"),
		ignorePats: mustGetStringArray(cmd, "ignore"),
	}
	return doScan(cmd.Context(), cfg)
}

func mustGetString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func mustGetInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

func mustGetStringArray(cmd *cobra.Command, name string) []string {
	v, _ := cmd.Flags().GetStringArray(name)
	return v
}

func doScan(ctx context.Context, cfg scanConfig) error {
	s := scanner.New(cfg.ignorePats)
	repoPaths, warnings := s.FindRepos(ctx, cfg.roots)
	statuses := checkRepos(ctx, repoPaths, cfg.workers)
	filtered := filterRepos(statuses, cfg.filter)

	result := report.ScanResult{
		GeneratedAt: time.Now(),
		Repos:       filtered,
		Warnings:    warnings,
		Summary: report.Summary{
			Total: len(filtered),
			Dirty: countDirty(filtered),
			Clean: len(filtered) - countDirty(filtered),
		},
	}

	switch cfg.outputFmt {
	case "json":
		return output.JSON(os.Stdout, result)
	default:
		return output.Table(os.Stdout, result)
	}
}

func checkRepos(ctx context.Context, paths []string, maxWorkers int) []report.RepoStatus {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	jobs := make(chan string, len(paths))
	results := make(chan report.RepoStatus, len(paths))

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- checkRepo(ctx, path)
			}
		}()
	}

	for _, p := range paths {
		jobs <- p
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var statuses []report.RepoStatus
	for r := range results {
		statuses = append(statuses, r)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Path < statuses[j].Path
	})

	return statuses
}

type repoChecker struct {
	ctx  context.Context
	path string
	errs []string
}

func (c *repoChecker) branch() string {
	b, err := git.GetBranch(c.ctx, c.path)
	if err != nil {
		c.errs = append(c.errs, "branch: "+err.Error())
	}
	return b
}

func (c *repoChecker) status() git.FileStatus {
	s, err := git.GetStatus(c.ctx, c.path)
	if err != nil {
		c.errs = append(c.errs, "status: "+err.Error())
	}
	return s
}

func (c *repoChecker) upstream() (ahead, behind int, noUpstream bool) {
	a, b, nu, err := git.GetUpstream(c.ctx, c.path)
	if err != nil {
		c.errs = append(c.errs, "upstream: "+err.Error())
	}
	return a, b, nu
}

func (c *repoChecker) err() string {
	return strings.Join(c.errs, "; ")
}

func checkRepo(ctx context.Context, path string) report.RepoStatus {
	c := &repoChecker{ctx: ctx, path: path}
	files := c.status()
	ahead, behind, noUpstream := c.upstream()

	// Worktree linkage is best-effort: detection failures leave the fields
	// zero-valued so the repo renders as a standalone row rather than erroring.
	commonDir, isWorktree, _ := git.GetWorktree(ctx, path)

	// Divergence from the remote default branch is best-effort too: repos with
	// no origin/HEAD simply leave these zero.
	var defaultBranch string
	var aheadDefault, behindDefault int
	if base, derr := git.GetDefaultBranch(ctx, path); derr == nil {
		defaultBranch = base
		// A resolvable default that we then can't diff against (e.g. an unborn
		// HEAD) is treated as "no divergence" rather than a scan error.
		aheadDefault, behindDefault, _ = git.GetDivergence(ctx, path, base)
	}

	return report.RepoStatus{
		Path:   path,
		Name:   git.GetRepoName(ctx, path),
		Branch: c.branch(),
		Files: report.FileStatus{
			Modified:  files.Modified,
			Untracked: files.Untracked,
			Staged:    files.Staged,
		},
		Ahead:      ahead,
		Behind:     behind,
		NoUpstream: noUpstream,
		Error:         c.err(),
		CommonDir:     commonDir,
		IsWorktree:    isWorktree,
		DefaultBranch: defaultBranch,
		AheadDefault:  aheadDefault,
		BehindDefault: behindDefault,
	}
}

func filterRepos(repos []report.RepoStatus, f string) []report.RepoStatus {
	if f == "all" {
		return repos
	}

	var filtered []report.RepoStatus
	for _, r := range repos {
		switch f {
		case "dirty":
			if r.IsDirty() {
				filtered = append(filtered, r)
			}
		case "uncommitted":
			if r.Files.Count() > 0 {
				filtered = append(filtered, r)
			}
		case "unpushed":
			if r.Ahead > 0 {
				filtered = append(filtered, r)
			}
		case "unpulled":
			if r.Behind > 0 {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

func countDirty(repos []report.RepoStatus) int {
	count := 0
	for _, r := range repos {
		if r.IsDirty() {
			count++
		}
	}
	return count
}
