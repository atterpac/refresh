package engine

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Ignore struct {
	Dir       []string `toml:"dir"`
	File      []string `toml:"file"`
	Extension []string `toml:"extension"`
	IgnoreGit bool     `toml:"git"`
}

type ignoreMap struct {
	dir       map[string]struct{}
	file      map[string]struct{}
	extension map[string]struct{}
	git       map[string]struct{}
}

// Runs all ignore checks to decide if reload should happen
func (i *ignoreMap) checkIgnore(path string) bool {
	basePath := filepath.Base(path)
	if isTmp(basePath) {
		return true
	}
	slog.Debug(fmt.Sprintf("Checking if %s is in %v", path, i.dir))
	if mapHasItems(i.dir) && (patternMatch(path, i.dir) || isIgnoreDir(path, i.dir)) {
		return true
	}
	_, ok := i.file[basePath]
	slog.Debug(fmt.Sprintf("Checking if %s is in %v", basePath, i.file))
	if mapHasItems(i.file) && patternMatch(basePath, i.file) || ok {
		return true
	}
	// Check if file extension is in ignore list
	_, ok = i.extension[filepath.Ext(path)]
	slog.Debug(fmt.Sprintf("Checking if %s is in %v", filepath.Ext(path), i.extension))
	if mapHasItems(i.extension) && patternMatch(path, i.extension) || ok {
		return true
	}
	if mapHasItems(i.git) && patternMatch(path, i.git) {
		return true
	}
	return false
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

func patternMatch(path string, PatternMap map[string]struct{}) bool {
	for pattern := range PatternMap {
		if patternCompare(path, pattern) {
			slog.Debug(fmt.Sprintf("Matched: %s with %s", path, pattern))
			return true
		}
	}
	return false
}

// Custom Unmarshal to stuff data into maps
func (i *ignoreMap) UnmarshalTOML(data interface{}) error {
	m, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected a map")
	}
	for key, value := range m {
		switch key {
		case "dir", "file", "extension":
			strArray, ok := value.([]string)
			if !ok {
				return fmt.Errorf("%s should be an array", key)
			}
			stringMap := make(map[string]struct{})
			for _, str := range strArray {
				stringMap[str] = struct{}{}
			}
			switch key {
			case "dir":
				i.dir = stringMap
			case "file":
				i.file = stringMap
			case "extension":
				i.extension = stringMap
			}
		}
	}
	return nil
}

func convertToIgnoreMap(ignore Ignore) ignoreMap {
	return ignoreMap{
		file:      convertToMap(ignore.File),
		dir:       convertToMap(ignore.Dir),
		extension: convertToMap(ignore.Extension),
	}
}

func convertToMap(slice []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, v := range slice {
		m[v] = struct{}{}
	}
	return m
}

func readGitIgnore(path string) map[string]struct{} {
	file, err := os.Open(path + "/.gitignore")
	if err != nil {
		return nil
	}
	defer file.Close()
	slog.Debug("Reading .gitignore")
	scanner := bufio.NewScanner(file)
	var linesMap = make(map[string]struct{})
	for scanner.Scan() {
		// Check if line is a comment
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		// Check if line is empty
		if len(scanner.Text()) == 0 {
			continue
		}

		line := scanner.Text()
		// Check if line does not start with '*'
		if !strings.HasPrefix(line, "*") {
			// Add asterisk to the beginning of line
			line = "*" + line
		}
		// Add to the map
		linesMap[line] = struct{}{}
	}
	slog.Debug(fmt.Sprintf("Read %v lines from .gitignore", linesMap))
	return linesMap
}

// patternCompare reports whether name matches the shell file name pattern.
// Unfortunately filepath.Match doesnt work for this use case
// Comparision laid out here: https://go.dev/play/p/Ega9qgD4Qz thanks to https://gitlab.com/hackandsla.sh/letterbox
func patternCompare(pattern, name string) (matched bool) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string
		star, chunk, pattern = scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string.
			return true
		}
		// Look for match at current position.
		t, ok := matchChunk(chunk, name)
		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
			name = t
			continue
		}
		if star {
			// Look for match skipping i+1 bytes.
			for i := 0; i < len(name); i++ {
				t, ok := matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
						continue
					}
					name = t
					continue Pattern
				}
			}
		}
		return false
	}
	return len(name) == 0
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	inrange := false
	var i int
Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if !inrange {
				break Scan
			}
		}
	}
	return star, pattern[0:i], pattern[i:]
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func matchChunk(chunk, s string) (rest string, ok bool) {
	for len(chunk) > 0 {
		if len(s) == 0 {
			return
		}
		switch chunk[0] {
		case '?':
			_, n := utf8.DecodeRuneInString(s)
			s = s[n:]
			chunk = chunk[1:]
		default:
			if chunk[0] != s[0] {
				return
			}
			s = s[1:]
			chunk = chunk[1:]
		}
	}
	return s, true
}
