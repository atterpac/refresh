// Command kitchen-sink demonstrates every refresh execute type in one config:
//
//   - once       — a setup step that runs a single time at startup
//   - background — a long-lived process that survives reloads
//   - blocking   — a build-style step that re-runs and must finish each cycle
//   - primary    — the main process, killed and restarted on every reload
//
// It also wires a reload Callback and ignore rules. Run it from this directory
// and edit watched/trigger.go to see a reload:
//
//	go run .
//
// The demo commands use a POSIX shell, so run it on Linux or macOS. The same
// buildConfig is exercised by the integration test in main_test.go.
package main

import (
	"log"
	"log/slog"

	"github.com/atterpac/refresh/engine"
)

// buildConfig assembles a Config exercising all execute types, rooted at root.
// Commands use paths relative to root — refresh runs each command with its
// working directory set accordingly, so no absolute paths are needed. Each step
// appends a line to a file under artifacts/ so its behavior is observable.
func buildConfig(root string) engine.Config {
	return engine.Config{
		RootPath: root,
		LogLevel: "info",
		Debounce: 300,
		Ignore: engine.Ignore{
			WatchedExten: []string{"*.go"},              // only react to Go changes
			Dir:          []string{".git", "artifacts"}, // never react to our own output
			File:         []string{"*_ignore.go"},
		},
		ExecStruct: []engine.Execute{
			// Runs a single time at startup, before anything else.
			{Cmd: "mkdir -p artifacts && echo setup >> artifacts/once.log", Type: engine.Once},
			// ChangeDir runs the command with its working directory set to sub/,
			// so the relative marker lands there rather than at the root.
			{Cmd: "echo here > marker.txt", ChangeDir: "sub", Type: engine.Once},
			// Started once, survives reloads, killed on shutdown.
			{Cmd: "mkdir -p artifacts && echo up >> artifacts/background.log && sleep 3600", Type: engine.Background},
			// Re-runs every cycle and must finish before the primary restarts.
			{Cmd: "mkdir -p artifacts && echo build >> artifacts/blocking.log", Type: engine.Blocking},
			// The long-lived process, killed and restarted on each reload.
			{Cmd: "mkdir -p artifacts && echo run >> artifacts/primary.log && sleep 3600", Type: engine.Primary},
		},
	}
}

func main() {
	cfg := buildConfig(".")
	cfg.Callback = func(e *engine.EventCallback) engine.EventHandle {
		slog.Info("reload callback", "path", e.Path)
		return engine.EventContinue
	}

	eng, err := engine.NewEngineFromConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := eng.Start(); err != nil {
		log.Fatal(err)
	}
}
