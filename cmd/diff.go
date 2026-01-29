package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/indexzero/skiffs/internal/output"
	"github.com/indexzero/skiffs/report"
	"github.com/spf13/cobra"
)

var diffOutputFmt string

var diffCmd = &cobra.Command{
	Use:   "diff <old.json> <new.json>",
	Short: "Compare two scan result files",
	Long:  "Compare two JSON scan result files and show what changed between them.",
	Args:  cobra.ExactArgs(2),
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().StringVarP(&diffOutputFmt, "output", "o", "table", "output format: table|json")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	oldPath, newPath := args[0], args[1]

	oldScan, err := loadScanResult(oldPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", oldPath, err)
	}

	newScan, err := loadScanResult(newPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", newPath, err)
	}

	diff := report.CompareScanResults(oldScan, newScan, oldPath, newPath)

	switch diffOutputFmt {
	case "json":
		return outputDiffJSON(diff)
	default:
		outputDiffTable(diff)
	}
	return nil
}

func loadScanResult(path string) (report.ScanResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return report.ScanResult{}, err
	}

	var result report.ScanResult
	if err := json.Unmarshal(data, &result); err != nil {
		return report.ScanResult{}, err
	}
	return result, nil
}

func outputDiffJSON(diff report.DiffResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(diff)
}

func outputDiffTable(diff report.DiffResult) {
	fmt.Println()
	fmt.Printf("%s\n", output.Bold("Diff Report"))
	fmt.Printf("%s %s\n", output.Gray("old:"), diff.OldScan)
	fmt.Printf("%s %s\n", output.Gray("new:"), diff.NewScan)
	fmt.Println()

	// summary
	fmt.Printf("cleaned: %s  dirtied: %s  added: %s  removed: %s\n\n",
		output.Green("%d", diff.Summary.ReposCleaned),
		output.Red("%d", diff.Summary.ReposDirtied),
		output.Cyan("%d", diff.Summary.ReposAdded),
		output.Yellow("%d", diff.Summary.ReposRemoved))

	// added repos
	if len(diff.Added) > 0 {
		fmt.Printf("%s\n", output.Cyan("Added:"))
		for _, d := range diff.Added {
			fmt.Printf("  + %s", d.Name)
			if d.Now != nil && d.Now.Dirty {
				fmt.Printf(" %s", output.Red("(dirty)"))
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// removed repos
	if len(diff.Removed) > 0 {
		fmt.Printf("%s\n", output.Yellow("Removed:"))
		for _, d := range diff.Removed {
			fmt.Printf("  - %s\n", d.Name)
		}
		fmt.Println()
	}

	// changed repos
	if len(diff.Changed) > 0 {
		fmt.Printf("%s\n", output.Bold("Changed:"))
		for _, d := range diff.Changed {
			fmt.Printf("  %s\n", d.Name)
			if d.Was != nil && d.Now != nil {
				// show transitions
				if d.Was.Dirty && !d.Now.Dirty {
					fmt.Printf("    %s\n", output.Green("→ cleaned"))
				} else if !d.Was.Dirty && d.Now.Dirty {
					fmt.Printf("    %s\n", output.Red("→ dirtied"))
				}

				// show numeric deltas
				if d.Delta.Uncommitted != 0 {
					sign := "+"
					if d.Delta.Uncommitted < 0 {
						sign = ""
					}
					fmt.Printf("    uncommitted: %s%d\n", sign, d.Delta.Uncommitted)
				}
				if d.Delta.Ahead != 0 {
					sign := "+"
					if d.Delta.Ahead < 0 {
						sign = ""
					}
					fmt.Printf("    ahead: %s%d\n", sign, d.Delta.Ahead)
				}
				if d.Was.Branch != d.Now.Branch {
					fmt.Printf("    branch: %s → %s\n", d.Was.Branch, d.Now.Branch)
				}
			}
		}
	}
}
