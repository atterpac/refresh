package main

import (
	"time"

	refresh "github.com/atterpac/refresh/engine"
	"github.com/atterpac/refresh/process"
)

func main() {
	background := process.Execute{
		Cmd: "pwd",
	}
	tidy := process.Execute{
		Cmd:  "go mod tidy",
		Type: process.Background,
	}
	build := process.Execute{
		Cmd:  "go build -o ./bin/myapp",
		Type: process.Blocking,
	}
	kill := process.KILL_STALE
	run := process.Execute{
		Cmd:       "./myapp",
		ChangeDir: "./binn",
		Type:      process.Primary,
	}
	ignore := refresh.Ignore{
		File:         []string{"ignore.go"},
		Dir:          []string{"*/ignore*"},
		WatchedExten: []string{"*.go", "*.mod", "*.js"},
		IgnoreGit:    true,
	}
	_ = refresh.Config{
		RootPath:         "./test",
		BackgroundStruct: background,
		// Below is ran when a reload is triggered before killing the stale version
		Ignore:     ignore,
		Debounce:   1000,
		LogLevel:   "debug",
		ExecStruct: []process.Execute{tidy, build, kill, run},
		Slog:       nil,
	}

	// watch := refresh.NewEngineFromConfig(config)
	watch, err := refresh.NewEngineFromYAML("./example.yaml")
	if err != nil {
		panic(err)
	}

	watch.AttachBackgroundCallback(func() bool {
		time.Sleep(5000 * time.Millisecond)
		return true
	})
	err = watch.Start()
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
