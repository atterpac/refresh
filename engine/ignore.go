package engine

import (
	"log/slog"
	"fmt"
	"path/filepath"
	"strings"
)

type Ignore struct {
	Dir       map[string]bool `toml:"dir"`
	File      map[string]bool `toml:"file"`
	Extension map[string]bool `toml:"extension"`
}

// Runs all ignore checks to decide if reload should happen
func (i *Ignore) checkIgnore(path string) bool {
	slog.Debug("Checking ignore", "path", path)
	var dir, file, ext bool = false, false, false
	basePath := filepath.Base(path)
	if !isMapEmpty(i.Dir) {
		dir = isIgnoreDir(path, i.Dir)
	}
	if !isMapEmpty(i.File) {
		_, file = i.File[basePath]
	}
	if !isMapEmpty(i.Extension) {
		_, ext = i.Extension[path]
	}

	return dir || file || ext || isTmp(basePath)
}

func isMapEmpty(m map[string]bool) bool {
	return len(m) <= 1
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
			return true
		}
	}
	return false
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
