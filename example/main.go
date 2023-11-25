package main

import (
	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	ignore := refresh.Ignore{
		File:      []string{"ignore.go", "*ignore.go"},
		Dir:       []string{"*/ignore*"},
		Extension: []string{".db", ".txt"},
		IgnoreGit: true,
	}
	config := refresh.Config{
		RootPath:    "./test",
		PreExec:     "go mod tidy",
		ExecCommand: "go run main.go",
		LogLevel:    "debug",
		Ignore:      ignore,
		Debounce:    1000,
		Callback:    RefreshCallback,
		Slog:        nil,
	}
	watch := refresh.NewEngineFromConfig(config)

	watch.Start()
	<-make(chan struct{})
}

func RefreshCallback(e *refresh.EventCallback) refresh.EventHandle {
	switch e.Type {
	case refresh.Create:
		return refresh.EventIgnore
	case refresh.Write:
		if e.Path == "test/monitored/ignore.go" {
			return refresh.EventBypass
		}
		return refresh.EventContinue
	case refresh.Remove:
		return refresh.EventContinue
		// Other cases as needed ...
	}
	return refresh.EventContinue
}
