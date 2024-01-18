package main

import (
	"time"

	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	background := refresh.Execute {
		Cmd: "pwd",
	}
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
		Cmd:        "./myapp",
		ChangeDir: "./bin",
		IsBlocking: false,
		IsPrimary:  true,
	}
	ignore := refresh.Ignore{
		File:         []string{"ignore.go"},
		Dir:          []string{"*/ignore*"},
		WatchedExten: []string{"*.go", "*.mod", "*.js"},
		IgnoreGit:    true,
	}
	_ = refresh.Config{
		RootPath: "./test",
		BackgroundStruct: background,
		// Below is ran when a reload is triggered before killing the stale version
		Ignore:     ignore,
		Debounce:   1000,
		LogLevel:   "debug",
		ExecStruct: []refresh.Execute{tidy, build, kill, run},
		Slog:       nil,
	}

	// watch := refresh.NewEngineFromConfig(config)
	watch := refresh.NewEngineFromTOML("./example.toml")

	watch.AttachBackgroundCallback(func() bool {
		time.Sleep(5000 * time.Millisecond)
		return true
	})
	err := watch.Start()
	if err != nil {
		panic(err)
	}

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
