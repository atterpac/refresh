package main

import (
	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	ignore := refresh.Ignore{
		File:      []string{"ignore.go"},
		Dir:       []string{"*/ignore*"},
		Extension: []string{".db"},
		IgnoreGit: true,
	}
	config := refresh.Config{
		RootPath: "./test",
		// Below is ran when a reload is triggered before killing the stale version
		Ignore:   ignore,
		Debounce: 1000,
		LogLevel: "info",
		Callback: RefreshCallback,
		Slog:     nil,
		ExecList: []string{"go mod tidy", "go build -o ./bin/myapp", "KILL_STALE", "REFRESH", "./bin/myapp"},
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
