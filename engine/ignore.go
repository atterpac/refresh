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
	var dir, file, ext bool = false, false, false
	basePath := filepath.Base(path)
	if MapNotEmpty(i.Dir) {
		dir = isPatternMatch(path, i.Dir)
		if !dir {
			dir = isIgnoreDir(path, i.Dir)
		}
		if dir {
			return true
		}
	}
	if MapNotEmpty(i.File) {
		file = isPatternMatch(basePath, i.File)
		if !file {
			file = i.File[basePath]
		}
		if file {
			return true
		}
	}
	if MapNotEmpty(i.Extension) {
		ext = isPatternMatch(filepath.Ext(path), i.Extension)
		if !ext {
			ext = i.Extension[filepath.Ext(path)]
		}
		if ext {
			return true
		}
	}
	slog.Debug(fmt.Sprintf("Ignore check: %v, %v, %v, %v", path, dir, file, ext))
	return dir || file || ext || isTmp(basePath)
}

func MapNotEmpty(m map[string]bool) bool {
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

func isPattern(path string) bool {
	return strings.Contains(path, "*") || strings.Contains(path, "!")
}

func isPatternMatch(path string, PatternMap map[string]bool) bool {
	for pattern := range PatternMap {
		if patternCompare(path, pattern) {
			slog.Debug(fmt.Sprintf("Matched: %s with %s", path, pattern))
			return true
		}
	}
	return false
}

func patternCompare(path, pattern string) bool {
	parts := strings.Split(pattern, "*")
	if pattern[0:1] == `!` {
		return !patternCompare(path, pattern[1:])
	}

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
