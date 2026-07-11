package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/indexzero/skiffs/report"
)

func TestWritePruneTable(t *testing.T) {
	color.NoColor = true // assert on plain text, not ANSI escapes

	r := report.PruneResult{
		SafeToDelete: []report.BranchSafety{
			{Name: "feat/done", Safe: true},
			{Name: "feat/held", Safe: true, WorktreePath: "/wt/held"},
		},
		NeedsReview: []report.BranchSafety{
			{Name: "feat/wip"},
		},
	}

	var buf bytes.Buffer
	writePruneTable(&buf, "/repo", r)
	out := buf.String()

	// The deletable branch appears and drives the suggested command.
	if !strings.Contains(out, "git branch -D feat/done") {
		t.Errorf("missing delete command for feat/done:\n%s", out)
	}
	// The worktree-held branch is surfaced with its path...
	if !strings.Contains(out, "feat/held") || !strings.Contains(out, "/wt/held") {
		t.Errorf("worktree-held branch not surfaced:\n%s", out)
	}
	// ...but excluded from the delete command.
	if strings.Contains(out, "feat/held fix") || strings.Contains(out, "-D feat/done feat/held") {
		t.Errorf("worktree-held branch should not be in delete command:\n%s", out)
	}
	// The needs-review branch appears.
	if !strings.Contains(out, "feat/wip") {
		t.Errorf("missing needs-review branch:\n%s", out)
	}
}

func TestWritePruneTable_Empty(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	writePruneTable(&buf, "/repo", report.PruneResult{})
	out := buf.String()

	// Empty sections render "(none)" and no delete command is suggested.
	if !strings.Contains(out, "(none)") {
		t.Errorf("expected (none) for empty sections:\n%s", out)
	}
	if strings.Contains(out, "git branch -D") {
		t.Errorf("no delete command expected when nothing is safe:\n%s", out)
	}
}
