// Package output provides formatters for scan results.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
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

	for _, unit := range groupRepos(r.Repos) {
		if len(unit) == 1 {
			writeRepoRow(w, unit[0])
			continue
		}
		writeGroup(w, unit)
	}

	return writeDetails(w, r.Repos)
}

// groupRepos clusters repos that share a CommonDir (the worktrees of one
// repository) into render units, preserving the first-seen order of groups so
// output stays deterministic for a path-sorted input. Repos without a
// CommonDir are emitted as singleton units keyed by their own path. Within a
// group the primary checkout sorts first, then linked worktrees by path.
func groupRepos(repos []report.RepoStatus) [][]report.RepoStatus {
	var order []string
	groups := make(map[string][]report.RepoStatus)
	for _, r := range repos {
		key := r.CommonDir
		if key == "" {
			key = r.Path
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], r)
	}

	units := make([][]report.RepoStatus, 0, len(order))
	for _, key := range order {
		g := groups[key]
		sort.Slice(g, func(i, j int) bool {
			if g[i].IsWorktree != g[j].IsWorktree {
				return !g[i].IsWorktree // primary checkout first
			}
			return g[i].Path < g[j].Path
		})
		units = append(units, g)
	}
	return units
}

// writeGroup renders a multi-worktree repository as one header followed by an
// indented, presence-marked row per checkout, so a repo fanned across many
// worktrees reads as a single cluster instead of repeating its name per row.
func writeGroup(w io.Writer, repos []report.RepoStatus) {
	fmt.Fprintf(w, "%s %s\n",
		Magenta(repos[0].Name),
		Dim("(%d worktrees)", len(repos)))
	for _, r := range repos {
		mark := Gray("+")
		if !r.IsWorktree {
			mark = Cyan("@")
		}
		fmt.Fprintf(w, "  %s %s %s\n",
			mark,
			Blue("%-*s", branchWidth, truncate(r.Branch, branchWidth)),
			statusString(r))
	}
}

func writeRepoRow(w io.Writer, r report.RepoStatus) {
	fmt.Fprintf(w, "%-*s %s %s\n",
		nameWidth, truncate(r.Name, nameWidth),
		Blue("%-*s", branchWidth, truncate(r.Branch, branchWidth)),
		statusString(r))
}

// statusString builds the glyph-and-count status column shared by standalone
// rows and grouped worktree rows.
func statusString(r report.RepoStatus) string {
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

	// No tracking branch is a presence fact, not a sync magnitude: the
	// ahead/behind columns are undefined, so render one dim token rather than
	// painting both arrows with a yellow dash that conflates "N/A" with
	// "caution".
	if r.NoUpstream {
		status.WriteString(Dim("no upstream"))
		return status.String()
	}

	if r.Ahead > 0 {
		status.WriteString(Green("↑%d ", r.Ahead))
	} else {
		status.WriteString(Gray("↑0 "))
	}

	if r.Behind > 0 {
		status.WriteString(Green("↓%d", r.Behind))
	} else {
		status.WriteString(Gray("↓0"))
	}

	return status.String()
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
