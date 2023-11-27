package main

import (
	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	ignore := refresh.Ignore{
		File:     []string{"ignore.go"},
		Dir:      []string{"*/ignore*": true},
		Extension: []string{".db": true},
		IgnoreGit: true,
	}
	config := refresh.Config{
		RootPath:    "./test",
		// Below is ran when a reload is triggered before killing the stale version
		PreBuild:    "go mod tidy",
		ExecBuild:   "go build -o ./bin/myapp",
		PostBuild:   "chmod +x ./bin/myapp", // Not applicable to golang but a potential use case
		// Run after killing building new version and killing the stale version
		PreRun:		 "",
		ExecRun:     "./bin/myapp",
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
