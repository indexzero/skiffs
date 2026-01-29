# Adding a TUI to SkiffScan

SkiffScan's architecture makes it straightforward to add an interactive terminal UI. The packages are importable, the data model is clean, and there's no existing TUI to work around.

## Prerequisites

```sh
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
```

## Basic Table View (~50 lines)

```go
// tui/model.go
package tui

import (
    "fmt"

    "github.com/charmbracelet/bubbles/table"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/indexzero/skiffscan/report"
)

type Model struct {
    table  table.Model
    repos  []report.RepoStatus
}

func New(repos []report.RepoStatus) Model {
    columns := []table.Column{
        {Title: "Repo", Width: 20},
        {Title: "Branch", Width: 20},
        {Title: "Status", Width: 15},
    }

    var rows []table.Row
    for _, r := range repos {
        rows = append(rows, table.Row{
            r.Name,
            r.Branch,
            formatStatus(r),
        })
    }

    t := table.New(table.WithColumns(columns), table.WithRows(rows))
    return Model{table: t, repos: repos}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    }
    var cmd tea.Cmd
    m.table, cmd = m.table.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    return m.table.View() + "\n\nPress q to quit."
}

func formatStatus(r report.RepoStatus) string {
    return fmt.Sprintf("●%d ↑%d ↓%d", r.Files.Count(), r.Ahead, r.Behind)
}
```

## Wiring Into the CLI

Add to `cmd/root.go`:

```go
import "github.com/indexzero/skiffscan/tui"

// in runScan(), add case:
case "interactive":
    p := tea.NewProgram(tui.New(filtered))
    if _, err := p.Run(); err != nil {
        return err
    }
```

## Using SkiffScan as a Library

You can also build a completely separate TUI that imports skiffscan:

```go
package main

import (
    "github.com/indexzero/skiffscan/git"
    "github.com/indexzero/skiffscan/report"
    "github.com/indexzero/skiffscan/scanner"
    tea "github.com/charmbracelet/bubbletea"
)

func main() {
    // Use skiffscan's scanner
    s := scanner.New(nil)
    paths, _ := s.FindRepos([]string{"~/Code"})

    // Check each repo
    var repos []report.RepoStatus
    for _, p := range paths {
        name := git.GetRepoName(p)
        branch, _ := git.GetBranch(p)
        files, _ := git.GetStatus(p)
        ahead, behind, noUpstream, _ := git.GetUpstream(p)

        repos = append(repos, report.RepoStatus{
            Path:       p,
            Name:       name,
            Branch:     branch,
            Files:      files,
            Ahead:      ahead,
            Behind:     behind,
            NoUpstream: noUpstream,
        })
    }

    // Build your own TUI
    p := tea.NewProgram(newModel(repos))
    p.Run()
}
```

## Next Steps

Once you have the basic table, you might add:

1. **Detail view** - Show file lists when selecting a repo
2. **Filtering** - Text input to filter repos by name
3. **Git operations** - Push/pull/fetch from the TUI
4. **Diff view** - Show changes since last scan

Each feature is additive. The `report.RepoStatus` struct has everything you need:
- `Files.Modified`, `Files.Untracked`, `Files.Staged` for detail views
- `Ahead`, `Behind`, `NoUpstream` for sync status
- `Path` for running git commands
