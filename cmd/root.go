// Package cmd provides the CLI commands for skiffscan.
package cmd

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/indexzero/skiffscan/git"
	"github.com/indexzero/skiffscan/output"
	"github.com/indexzero/skiffscan/report"
	"github.com/indexzero/skiffscan/scanner"
	"github.com/spf13/cobra"
)

var (
	roots      []string
	filter     string
	outputFmt  string
	workers    int
	ignorePats []string
)

var rootCmd = &cobra.Command{
	Use:   "skiffscan",
	Short: "Scan directories for git repositories and report status",
	Long:  "skiffscan walks directory trees to find git repositories and reports their status: uncommitted files, unpushed commits, and unpulled changes.",
	RunE:  runScan,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringArrayVarP(&roots, "root", "r", []string{"."}, "root directories to scan")
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "dirty", "filter: all|dirty|uncommitted|unpushed|unpulled")
	rootCmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "output format: table|json")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 8, "concurrent git operations")
	rootCmd.Flags().StringArrayVarP(&ignorePats, "ignore", "i", scanner.DefaultIgnorePatterns(), "glob patterns to ignore")
}

func runScan(cmd *cobra.Command, args []string) error {
	// find repos
	s := scanner.New(ignorePats)
	repoPaths, warnings := s.FindRepos(roots)

	// check repo status concurrently
	statuses := checkRepos(repoPaths, workers)

	// filter
	filtered := filterRepos(statuses, filter)

	// build result
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

	// output
	switch outputFmt {
	case "json":
		return output.JSON(result)
	default:
		output.Table(result)
	}
	return nil
}

func checkRepos(paths []string, maxWorkers int) []report.RepoStatus {
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
				results <- checkRepo(path)
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

func checkRepo(path string) report.RepoStatus {
	name := git.GetRepoName(path)
	branch, _ := git.GetBranch(path)
	files, _ := git.GetStatus(path)
	ahead, behind, noUpstream, _ := git.GetUpstream(path)

	return report.RepoStatus{
		Path:       path,
		Name:       name,
		Branch:     branch,
		Files:      files,
		Ahead:      ahead,
		Behind:     behind,
		NoUpstream: noUpstream,
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
