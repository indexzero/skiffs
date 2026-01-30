// Package output provides formatters for scan results.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/indexzero/skiffs/report"
)

const (
	nameWidth   = 24
	branchWidth = 24
)

// Color formatting functions for terminal output.
var (
	Bold    = color.New(color.Bold).SprintfFunc()
	Dim     = color.New(color.Faint).SprintfFunc()
	Cyan    = color.New(color.FgCyan, color.Bold).SprintfFunc()
	Blue    = color.New(color.FgBlue).SprintfFunc()
	Red     = color.New(color.FgRed).SprintfFunc()
	Green   = color.New(color.FgGreen).SprintfFunc()
	Yellow  = color.New(color.FgYellow).SprintfFunc()
	Gray    = color.New(color.FgHiBlack).SprintfFunc()
	Magenta = color.New(color.FgMagenta, color.Bold).SprintfFunc()
)

// Table writes the scan result as a formatted table.
func Table(w io.Writer, r report.ScanResult) error {
	for _, warn := range r.Warnings {
		fmt.Fprintf(os.Stderr, "%s %s\n", Yellow("warning:"), warn)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", Bold("Scan Report"))
	fmt.Fprintf(w, "%s %s\n", Dim("generated:"), Gray(r.GeneratedAt.Format("2006-01-02 15:04:05")))
	fmt.Fprintf(w, "repos: %s  dirty: %s  clean: %s\n\n",
		Bold("%d", r.Summary.Total),
		colorCount(r.Summary.Dirty, Red),
		colorCount(r.Summary.Clean, Green))

	if len(r.Repos) == 0 {
		fmt.Fprintln(w, Dim("no repositories found"))
		return nil
	}

	fmt.Fprintf(w, "%s %s %s\n",
		Cyan("%-*s", nameWidth, "REPO"),
		Cyan("%-*s", branchWidth, "BRANCH"),
		Cyan("STATUS"))
	fmt.Fprintln(w, strings.Repeat("─", nameWidth+branchWidth+20))

	for _, repo := range r.Repos {
		writeRepoRow(w, repo)
	}

	return writeDetails(w, r.Repos)
}

func writeRepoRow(w io.Writer, r report.RepoStatus) {
	name := truncate(r.Name, nameWidth)
	branch := truncate(r.Branch, branchWidth)

	var status strings.Builder

	if r.Error != "" {
		status.WriteString(Red("✗ "))
	}

	count := r.Files.Count()
	if count > 0 {
		status.WriteString(Red("●%d ", count))
	} else {
		status.WriteString(Gray("○0 "))
	}

	if r.Ahead > 0 {
		status.WriteString(Green("↑%d ", r.Ahead))
	} else if r.NoUpstream {
		status.WriteString(Yellow("↑- "))
	} else {
		status.WriteString(Gray("↑0 "))
	}

	if r.Behind > 0 {
		status.WriteString(Green("↓%d", r.Behind))
	} else if r.NoUpstream {
		status.WriteString(Yellow("↓-"))
	} else {
		status.WriteString(Gray("↓0"))
	}

	fmt.Fprintf(w, "%-*s %s %s\n",
		nameWidth, name,
		Blue("%-*s", branchWidth, branch),
		status.String())
}

func writeDetails(w io.Writer, repos []report.RepoStatus) error {
	var detailed []report.RepoStatus
	for _, r := range repos {
		if r.Files.Count() > 0 || r.Error != "" {
			detailed = append(detailed, r)
		}
	}

	if len(detailed) == 0 {
		return nil
	}

	fmt.Fprintf(w, "\n%s\n", Cyan("Details:"))
	for _, r := range detailed {
		fmt.Fprintf(w, "\n%s %s\n", Magenta("repo:"), r.Name)
		fmt.Fprintf(w, "%s %s\n", Magenta("path:"), r.Path)
		if r.Error != "" {
			fmt.Fprintf(w, "  %s\n", Red("error: %s", r.Error))
		}
		for _, f := range r.Files.Modified {
			fmt.Fprintf(w, "  %s\n", Gray("M %s", f))
		}
		for _, f := range r.Files.Staged {
			fmt.Fprintf(w, "  %s\n", Gray("A %s", f))
		}
		for _, f := range r.Files.Untracked {
			fmt.Fprintf(w, "  %s\n", Gray("? %s", f))
		}
	}
	return nil
}

// JSON writes the scan result as formatted JSON.
func JSON(w io.Writer, r report.ScanResult) error {
	enc := json.NewEncoder(w)
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

func colorCount(n int, colorFn func(string, ...any) string) string {
	if n > 0 {
		return colorFn("%d", n)
	}
	return fmt.Sprintf("%d", n)
}
