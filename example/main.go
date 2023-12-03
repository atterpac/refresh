package main

import (
	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	tidy := refresh.Execute{
		Cmd:        "go mod tidy",
		IsBlocking: true,
		IsPrimary:  false,
	}
	build := refresh.Execute{
		Cmd:        "go build -o ./bin/myapp",
		IsBlocking: true,
		IsPrimary:  false,
	}
	kill := refresh.KILL_STALE
	run := refresh.Execute{
		Cmd:        "./bin/myapp",
		IsBlocking: false,
		IsPrimary:  true,
	}
	ignore := refresh.Ignore{
		File:         []string{"ignore.go"},
		Dir:          []string{"*/ignore*"},
		WatchedExten: []string{"*.go", "*.mod", "*.js"},
		IgnoreGit:    true,
	}
	config := refresh.Config{
		RootPath: "./test",
		// Below is ran when a reload is triggered before killing the stale version
		Ignore:     ignore,
		Debounce:   1000,
		LogLevel:   "debug",
		ExecStruct: []refresh.Execute{tidy, build, kill, run},
		Slog:       nil,
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
