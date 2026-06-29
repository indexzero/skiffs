# skiffs

Traverse directories for git repositories and report their status against upstream.

A skiff is a small boat—often one you built yourself. Your code are your skiffs and
we have more than ever to keep track of. From small, hand-built vessels to monorepos
they are all scattered across the filesystem. This tool helps you check if any are
taking on water.

> _The name started as "scdiffs" for "source control diffs," then lost some letters.
> The boat thing came later, but it stuck._

## Install

```sh
go install github.com/indexzero/skiffs@latest
```

## Usage

```sh
# scan current directory
skiffs

# scan specific roots
skiffs -r ~/Code -r ~/work

# show all repos (not just dirty)
skiffs -f all

# output as JSON
skiffs -o json
```

## Flags

```
-r, --root     Root directories to scan (repeatable, default: .)
-f, --filter   Filter: all|dirty|uncommitted|unpushed|unpulled (default: dirty)
-o, --output   Output format: table|json (default: table)
-w, --workers  Concurrent git operations (default: 8)
-i, --ignore   Glob patterns to ignore (repeatable)
```

## Output

Table output shows repo name, branch, and status indicators:
- Uncommitted files count (`●N`)
- Commits ahead of upstream, unpushed (`↑N`)
- Commits behind upstream, unpulled (`↓N`)

A branch with no tracking branch reads `no upstream` instead of the ahead/behind
columns — that's a fact about the branch, not a per-arrow warning. When a branch
has diverged from the repository's default branch (resolved from `origin/HEAD`),
a dim `[main ↑N ↓N]` suffix shows how far, even when there's no upstream to
compare against.

### Worktrees

Checkouts that share a repository (linked worktrees) are grouped under a single
header instead of repeating the repo name once per checkout:

```
yourbytes (3 worktrees)
  @ main                     ●0 ↑0 ↓0
  + typed/source-a           ●2 ↑9 ↓39  [main ↑9 ↓39]
  + typed/source-b           ○0 ↑9 ↓39  [main ↑9 ↓39]
```

`@` marks the primary checkout; `+` marks a linked worktree.

JSON output includes full file lists for each status category (modified,
untracked, staged) plus the worktree linkage (`commonDir`, `isWorktree`) and
default-branch divergence (`defaultBranch`, `aheadDefault`, `behindDefault`).

## Comparing Scans Over Time

Save scans as JSON and compare them later:

```sh
# save today's scan
skiffs -r ~/Code -o json > ~/.cache/skiffs/$(date +%Y-%m-%d).json

# compare to yesterday
skiffs diff ~/.cache/skiffs/yesterday.json ~/.cache/skiffs/today.json
```

The diff command shows:
- Added and removed repositories
- Repos that became dirty or were cleaned
- Changes in uncommitted/ahead/behind counts

Example output:
```
Diff Report
old: yesterday.json
new: today.json

cleaned: 2  dirtied: 1  added: 0  removed: 0

Changed:
  myproject
    → cleaned
  another-repo
    → dirtied
    uncommitted: +3
```

## Pruning Branches

`skiffs prune` finds local branches that are safe to delete in a single
repository. It classifies branches that no longer exist on the remote:

- **AreSafe** — a merged or closed pull request matches the branch (safe to
  delete; rendered dimmed)
- **MaybeSafe** — no matching PR was found (review before deleting; rendered in
  yellow)

Branches still on the remote are ignored, and branches checked out in a worktree
are called out separately since they can't be deleted in place.

```sh
# fetch + analyze the current repository
skiffs prune

# skip the fetch, use existing remote refs
skiffs prune --no-fetch

# analyze a specific repo, as JSON
skiffs prune -r ~/Code/myproject -o json
```

Requires the [`gh`](https://cli.github.com) CLI, authenticated (`gh auth login`),
to look up pull-request state. This is a Go port of the `bonsai` script.

```
Prune Report
repo: .

AreSafe (not on remote + merged/closed PR)
  ✓ feat/old-experiment
  ✓ fix/typo

MaybeSafe (not on remote + no matching PR — review first)
  ? feat/wip

To delete AreSafe branches:
  git branch -D feat/old-experiment fix/typo
```

## License

MIT
