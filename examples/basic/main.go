// Command basic demonstrates embedding refresh as a library. It watches this
// directory for *.go changes and, on each change, rebuilds and restarts the
// small program under ./app.
//
// Run it from this directory:
//
//	go run .
//
// Then edit app/main.go and watch it rebuild and restart. The same setup can be
// expressed in a config file instead — see config.yaml and NewEngineFromYAML.
package main

import (
	"log"

	"github.com/atterpac/refresh/engine"
)

func main() {
	cfg := engine.Config{
		RootPath: ".",
		LogLevel: "debug",
		Debounce: 500,
		Ignore: engine.Ignore{
			WatchedExten: []string{"*.go"},
			Dir:          []string{".git", "node_modules", "vendor", "bin"},
			IgnoreGit:    true,
		},
		ExecStruct: []engine.Execute{
			// Build the watched app; blocking so the restart waits for a good binary.
			{Cmd: "go build -o ./bin/app ./app", Type: engine.Blocking},
			// The long-running primary; killed and restarted on each reload.
			{Cmd: "./bin/app", Type: engine.Primary},
		},
	}

	eng, err := engine.NewEngineFromConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := eng.Start(); err != nil {
		log.Fatal(err)
	}
}
