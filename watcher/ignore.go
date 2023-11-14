package watcher

import (
	"fmt"
	"os"
)

type Ignore struct {
	Dir       map[string]bool `toml:"dir"`
	File      map[string]bool `toml:"file"`
	Extension map[string]bool `toml:"extension"`
}

func (i *Ignore) CheckIgnore(path string) bool {
	_, isDir := i.Dir[path]
	_, isFile := i.File[path]
	_, isExt := i.Extension[path]

	return (isDir && isDirectory(path)) || isFile || isExt
}

func isDirectory(path string) bool {
	pathInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return pathInfo.IsDir()
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

