// Package scanner finds git repositories in directory trees.
package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Scanner finds git repositories.
type Scanner struct {
	ignorePatterns []string
}

// New creates a scanner with the given ignore patterns.
func New(ignorePatterns []string) *Scanner {
	// normalize patterns
	var patterns []string
	for _, p := range ignorePatterns {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		p = filepath.ToSlash(p)
		patterns = append(patterns, p)
	}
	return &Scanner{ignorePatterns: patterns}
}

// FindRepos walks the given roots and returns paths to git repositories.
func (s *Scanner) FindRepos(ctx context.Context, roots []string) ([]string, []string) {
	var repos []string
	var warnings []string
	visited := make(map[string]bool)

	for _, root := range roots {
		if ctx.Err() != nil {
			break
		}

		root = os.ExpandEnv(root)
		root, err := filepath.Abs(root)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if err != nil {
				warnings = append(warnings, err.Error())
				return nil
			}

			if !d.IsDir() {
				return nil
			}

			if visited[path] {
				return fs.SkipDir
			}
			visited[path] = true

			if s.shouldIgnore(path) {
				return fs.SkipDir
			}

			if isGitRepo(path) {
				repos = append(repos, path)
				return fs.SkipDir
			}
			return nil
		})
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			warnings = append(warnings, err.Error())
		}
	}

	return repos, warnings
}

// isGitRepo checks if the directory is a git repository.
func isGitRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return true
	}
	// .git file (worktree or submodule)
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "gitdir:")
}

// shouldIgnore checks if the path matches any ignore pattern.
func (s *Scanner) shouldIgnore(path string) bool {
	normalizedPath := filepath.ToSlash(path)
	for _, pattern := range s.ignorePatterns {
		matched, _ := doublestar.PathMatch(pattern, normalizedPath)
		if matched {
			return true
		}
		// also try matching against basename for simple patterns
		if !strings.Contains(pattern, "/") {
			matched, _ = doublestar.PathMatch("**/"+pattern+"/**", normalizedPath)
			if matched {
				return true
			}
		}
	}
	return false
}

// DefaultIgnorePatterns returns sensible defaults for common non-repo directories.
func DefaultIgnorePatterns() []string {
	return []string{
		"**/node_modules/**",
		"**/vendor/**",
		"**/.venv/**",
		"**/venv/**",
		"**/.cache/**",
		"**/.local/**",
		"**/build/**",
		"**/dist/**",
		"**/.idea/**",
		"**/.vscode/**",
	}
}
