package main

import (
	"flag"
	"gotato/watcher"
)

func main() {
	var rootPath string
	var ignoreList string
	var execCommand string
	var logLevel string
	var confPath string

	flag.StringVar(&rootPath, "path", ".", "Root path to watch")
	flag.StringVar(&ignoreList, "ignore", "", "Comma-separated list of files to ignore")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "log", "", "Level to set Logs")
	flag.StringVar(&confPath, "f", "", "File to read config from")
	flag.Parse()


	watch := watcher.NewWatcherFromConfig(confPath)
	// watch := watcher.Config{
	// 	Label:       "Golang",
	// 	RootPath:    "../testProject",
	// 	IgnoreList:  []string{"newfile.go"},
	// 	ExecCommand: []string{"go", "run", "main.go"},
	// }

	watch.Start()

	<-make(chan struct{})
}
