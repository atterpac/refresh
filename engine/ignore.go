package engine

import (
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
)

type Ignore struct {
	Dir          []string `toml:"dir"               yaml:"dir"`
	File         []string `toml:"file"              yaml:"file"`
	WatchedExten []string `toml:"watched_extension" yaml:"watched_extension"`
	IgnoreGit    bool     `toml:"git"               yaml:"git"`

	// gitPatterns holds globs read from the root .gitignore when IgnoreGit is
	// set. Populated by the engine at startup; not user-configured.
	gitPatterns []string
}

// shouldIgnore reports whether a change to path should be skipped. A path is
// considered only if it matches a watched extension; it is then ignored if it
// sits in an ignored directory or matches an ignore-file or .gitignore pattern.
func (i *Ignore) shouldIgnore(path string) bool {
	if !i.isWatchedExtension(path) {
		return true
	}
	return isIgnoreDir(path, i.Dir) ||
		patternMatch(path, i.Dir) ||
		patternMatch(path, i.File) ||
		patternMatch(path, i.gitPatterns)
}

func (i *Ignore) isWatchedExtension(path string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}

	// First check for direct extension matches (e.g., ".go")
	for _, watchedExt := range i.WatchedExten {
		if watchedExt == ext || watchedExt == "*"+ext {
			return true
		}
	}

	// Then try pattern matching for more complex patterns
	return patternMatch(path, i.WatchedExten)
}

// isIgnoreDir reports whether any path component exactly matches an ignore rule.
func isIgnoreDir(path string, rules []string) bool {
	for dir := range strings.SplitSeq(path, string(filepath.Separator)) {
		if slices.Contains(rules, dir) {
			slog.Debug("ignoring directory", "dir", dir)
			return true
		}
	}
	return false
}
