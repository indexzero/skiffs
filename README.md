# skiffscan

Scan directories for Git repositories and report their status.

## Install

```sh
go install github.com/indexzero/skiffscan@latest
```

## Usage

```sh
# scan current directory
skiffscan

# scan specific roots
skiffscan -r ~/Code -r ~/work

# show all repos (not just dirty)
skiffscan -f all

# output as JSON
skiffscan -o json
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
- Uncommitted files count
- Commits ahead of upstream (unpushed)
- Commits behind upstream (unpulled)

JSON output includes full file lists for each status category:
- Modified files
- Untracked files
- Staged files

## Comparing Scans Over Time

Save scans as JSON and compare them later:

```sh
# save today's scan
skiffscan -r ~/Code -o json > ~/.cache/skiffscan/$(date +%Y-%m-%d).json

# compare to yesterday
skiffscan diff ~/.cache/skiffscan/yesterday.json ~/.cache/skiffscan/today.json
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

## License

MIT
