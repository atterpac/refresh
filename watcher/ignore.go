package watcher

import "path/filepath"

type Ignore struct {
	Dir       []string `toml:"dir"`
	File      []string `toml:"file"`
	Extension []string `toml:"extension"`
}

func (i *Ignore) AddDir(dir string) {
	i.Dir = append(i.Dir, dir)
}

func (i *Ignore) AddFile(file string) {
	i.File = append(i.File, file)
}

func (i *Ignore) AddExtension(ext string) {
	i.Extension = append(i.Extension, ext)
}

func (i *Ignore) CheckIgnore(path string) bool {
	for _, dir := range i.Dir {
		if dir == path {
			return true
		}
	}
	for _, file := range i.File {
		if file == path {
			return true
		}
	}
	for _, ext := range i.Extension {
		if ext == filepath.Ext(path) {
			return true
		}
	}
	return false
}

