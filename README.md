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

JSON output includes full file lists for each status category.

## License

MIT
