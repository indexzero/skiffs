// Package output provides formatters for scan results.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/indexzero/skiffscan/report"
)

const (
	nameWidth   = 24
	branchWidth = 24
)

var (
	bold    = color.New(color.Bold).SprintfFunc()
	dim     = color.New(color.Faint).SprintfFunc()
	cyan    = color.New(color.FgCyan, color.Bold).SprintfFunc()
	blue    = color.New(color.FgBlue).SprintfFunc()
	red     = color.New(color.FgRed).SprintfFunc()
	green   = color.New(color.FgGreen).SprintfFunc()
	yellow  = color.New(color.FgYellow).SprintfFunc()
	gray    = color.New(color.FgHiBlack).SprintfFunc()
	magenta = color.New(color.FgMagenta, color.Bold).SprintfFunc()
)

// Table prints the scan result as a formatted table.
func Table(r report.ScanResult) {
	// print warnings
	for _, w := range r.Warnings {
		fmt.Fprintf(os.Stderr, "%s %s\n", yellow("warning:"), w)
	}

	// header
	fmt.Println()
	fmt.Printf("%s\n", bold("Scan Report"))
	fmt.Printf("%s %s\n", dim("generated:"), gray(r.GeneratedAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("repos: %s  dirty: %s  clean: %s\n\n",
		bold("%d", r.Summary.Total),
		colorCount(r.Summary.Dirty, red),
		colorCount(r.Summary.Clean, green))

	if len(r.Repos) == 0 {
		fmt.Println(dim("no repositories found"))
		return
	}

	// table header
	fmt.Printf("%s %s %s\n",
		cyan("%-*s", nameWidth, "REPO"),
		cyan("%-*s", branchWidth, "BRANCH"),
		cyan("STATUS"))
	fmt.Println(strings.Repeat("─", nameWidth+branchWidth+20))

	// rows
	for _, repo := range r.Repos {
		printRepoRow(repo)
	}

	// details section for repos with uncommitted files
	printDetails(r.Repos)
}

func printRepoRow(r report.RepoStatus) {
	name := truncate(r.Name, nameWidth)
	branch := truncate(r.Branch, branchWidth)

	var status strings.Builder

	// uncommitted count
	count := r.Files.Count()
	if count > 0 {
		status.WriteString(red("●%d ", count))
	} else {
		status.WriteString(gray("○0 "))
	}

	// ahead
	if r.Ahead > 0 {
		status.WriteString(green("↑%d ", r.Ahead))
	} else if r.NoUpstream {
		status.WriteString(yellow("↑- "))
	} else {
		status.WriteString(gray("↑0 "))
	}

	// behind
	if r.Behind > 0 {
		status.WriteString(green("↓%d", r.Behind))
	} else if r.NoUpstream {
		status.WriteString(yellow("↓-"))
	} else {
		status.WriteString(gray("↓0"))
	}

	fmt.Printf("%-*s %s %s\n",
		nameWidth, name,
		blue("%-*s", branchWidth, branch),
		status.String())
}

func printDetails(repos []report.RepoStatus) {
	var dirty []report.RepoStatus
	for _, r := range repos {
		if r.Files.Count() > 0 {
			dirty = append(dirty, r)
		}
	}

	if len(dirty) == 0 {
		return
	}

	fmt.Printf("\n%s\n", cyan("Details:"))
	for _, r := range dirty {
		fmt.Printf("\n%s %s\n", magenta("repo:"), r.Name)
		fmt.Printf("%s %s\n", magenta("path:"), r.Path)
		for _, f := range r.Files.Modified {
			fmt.Printf("  %s\n", gray("M %s", f))
		}
		for _, f := range r.Files.Staged {
			fmt.Printf("  %s\n", gray("A %s", f))
		}
		for _, f := range r.Files.Untracked {
			fmt.Printf("  %s\n", gray("? %s", f))
		}
	}
}

// JSON prints the scan result as formatted JSON.
func JSON(r report.ScanResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func colorCount(n int, colorFn func(string, ...interface{}) string) string {
	if n > 0 {
		return colorFn("%d", n)
	}
	return fmt.Sprintf("%d", n)
}
