package main

import (
	"revolver/config"
	"revolver/watcher"

	"github.com/charmbracelet/log"
)

func main() {
	log.Info("Starting Watcher")
	watch := watcher.Watcher{
		Active: true,
		Config: config.Config{
			Label:       "Golang",
			RootPath:    "../testProject",
			IgnoreList:  []string{"newfile.go"},
			ExecCommand: []string{"go", "run", "main.go"},
		},
	}

	watch.Start()

	<-make(chan struct{})
}
