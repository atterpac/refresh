package engine

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

type Ignore struct {
	Dir          []string `toml:"dir"`
	File         []string `toml:"file"`
	WatchedExten []string `toml:"watched_extension"`
	IgnoreGit    bool     `toml:"git"`
}

type ignoreMap struct {
	dir       map[string]struct{}
	file      map[string]struct{}
	extension map[string]struct{}
	git       map[string]struct{}
}

// Runs all ignore checks to decide if reload should happen
func (i *ignoreMap) checkIgnore(path string) bool {
	slog.Debug("Checking Ignore")
	basePath := filepath.Base(path)
	if isTmp(basePath) {
		return true
	}
	if isIgnoreDir(path, i.dir) {
		return true
	}
	dir := checkIgnoreMap(path, i.dir)
	file := checkIgnoreMap(path, i.file)
	git := checkIgnoreMap(path, i.git)
	return dir || file || git
}

func checkIgnoreMap(path string, rules map[string]struct{}) bool {
	slog.Debug(fmt.Sprintf("Checking map: %v for %s", rules, path))
	_, ok := rules[path]
	return mapHasItems(rules) && patternMatch(path, rules) || ok
}

func checkExtension(path string, rules map[string]struct{}) bool {
	slog.Debug(fmt.Sprintf("Checking Extension map: %v for %s", rules, path))
	return patternMatch(path, rules)
}

func mapHasItems(m map[string]struct{}) bool {
	return len(m) >= 0
}

// Checks if filepath ends in tilde returns true if it does
func isTmp(path string) bool {
	return len(path) > 0 && path[len(path)-1] == '~'
}

// Checks if path contains any directories in the ignore directory config
func isIgnoreDir(path string, Dirmap map[string]struct{}) bool {
	dirs := strings.Split(path, string(filepath.Separator))
	for _, dir := range dirs {
		_, ok := Dirmap[dir]
		if ok {
			slog.Debug(fmt.Sprintf("Matched: %s with %s", path, dir))
			return true
		}
	}
	return false
}

func convertToIgnoreMap(ignore Ignore) ignoreMap {
	return ignoreMap{
		file:      convertToMap(ignore.File),
		dir:       convertToMap(ignore.Dir),
		extension: convertToMap(ignore.WatchedExten),
	}
}

func convertToMap(slice []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, v := range slice {
		m[v] = struct{}{}
	}
	return m
}

