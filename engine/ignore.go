package engine

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

type Ignore struct {
	Pattern   map[string]bool `toml:"pattern"`
	Dir       map[string]bool `toml:"dir"`
	File      map[string]bool `toml:"file"`
	Extension map[string]bool `toml:"extension"`
}

// Runs all ignore checks to decide if reload should happen
func (i *Ignore) checkIgnore(path string) bool {
	basePath := filepath.Base(path)
	if isTmp(basePath) {
		return true
	}
	if mapHasItems(i.Dir) && (patternMatch(path, i.Dir) || isIgnoreDir(path, i.Dir)) {
		return true
	}
	if mapHasItems(i.File) && (patternMatch(basePath, i.File) || i.File[basePath]) {
		return true
	}
	if mapHasItems(i.Extension) && (patternMatch(path, i.Extension) || i.Extension[filepath.Ext(path)]) {
		return true
	}
	return false
}

func mapHasItems(m map[string]bool) bool {
	return len(m) >= 0
}

// Checks if filepath ends in tilde returns true if it does
func isTmp(path string) bool {
	return len(path) > 0 && path[len(path)-1] == '~'
}

// Checks if path contains any directories in the ignore directory config
func isIgnoreDir(path string, Dirmap map[string]bool) bool {
	dirs := strings.Split(path, string(filepath.Separator))
	for _, dir := range dirs {
		if Dirmap[dir] {
			slog.Debug(fmt.Sprintf("Matched: %s with %s", path, dir))
			return true
		}
	}
	return false
}

func patternMatch(path string, PatternMap map[string]bool) bool {
	for pattern := range PatternMap {
		if patternCompare(path, pattern) {
			slog.Debug(fmt.Sprintf("Matched: %s with %s", path, pattern))
			return true
		}
	}
	return false
}

func patternCompare(path, pattern string) bool {
	if pattern[0:1] == `!` {
		return !patternCompare(path, pattern[1:])
	}
	parts := strings.Split(pattern, "*")
	// Match the first part before the wildcard
	i := 0
	for _, part := range parts[0] {
		index := strings.IndexRune(path[i:], part)
		if index == -1 {

			return false
		}
		i += index + 1
	}
	// Match the second part after the wildcard
	j := len(parts[1]) - 1
	for _, part := range parts[1] {
		found := false
		for ; i <= len(path)-1; i++ {
			if rune(path[i]) == part {
				found = true
				break
			}
		}
		if !found {
			return false
		}
		j--
	}

	return j < 0
}

// Custom Unmarshal to stuff data into maps
func (i *Ignore) UnmarshalTOML(data interface{}) error {
	m, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected a map")
	}

	for key, value := range m {
		switch key {
		case "dir", "file", "extension":
			strArray, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("%s should be an array", key)
			}

			stringMap := make(map[string]bool)
			for _, str := range strArray {
				stringMap[str.(string)] = true
			}

			switch key {
			case "pattern":
				i.Pattern = stringMap
			case "dir":
				i.Dir = stringMap
			case "file":
				i.File = stringMap
			case "extension":
				i.Extension = stringMap
			}
		}
	}
	return nil
}
