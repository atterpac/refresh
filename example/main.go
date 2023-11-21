package main

import (
	hotato "github.com/atterpac/hotato/engine"
)

func main() {
	ignore := hotato.Ignore{
		File:      map[string]bool{"ignore.go": true},
		Dir:       map[string]bool{"ignoreme": true},
		Extension: map[string]bool{".txt": true},
	}
	config := hotato.Config{
		RootPath:    "./test",
		PreExec:     "go mod tidy",
		ExecCommand: "go run main.go",
		LogLevel:    "debug",
		Ignore:      ignore,
		Debounce:    1000,
	}
	watch := hotato.NewEngineFromConfig(config)

	watch.Start()
	<-make(chan struct{})
}
