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
		Callback:    HotatoCallback,
		Slog: nil,
	}
	watch := hotato.NewEngineFromConfig(config)


	watch.Start()
	<-make(chan struct{})
}

func HotatoCallback(e *hotato.EventCallback) hotato.EventHandle {
	switch e.Type {
	case hotato.Create:
		return hotato.EventIgnore
	case hotato.Write:
		if e.Path == "test/monitored/ignore.go" {
			return hotato.EventBypass
		}
		return hotato.EventContinue
	case hotato.Remove:
		return hotato.EventContinue
	// Other Hotato Event Types...
	}
	return hotato.EventContinue
}
