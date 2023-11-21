package main

import (
	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	ignore := refresh.Ignore{
		File:      map[string]bool{"ignore.go": true},
		Dir:       map[string]bool{"ignoreme": true},
		Extension: map[string]bool{".txt": true},
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
		// Other Hotato Event Types...
	}
	return refresh.EventContinue
}
