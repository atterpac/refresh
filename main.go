package main

import "revolver/watcher"

func main() {
	config := watcher.WatchConfig{
		RootPath:    "../testProject",
		IgnoreList:  []string{"newfile.go"},
		ExecCommand: []string{"go", "run", "main.go"},
	}

	watch := watcher.WatchEngine{
		Label: "Golang",
		State: 1,
		Config: config,
	}

	watcher.Monitor(&watch)
	<-make(chan struct{})
}
