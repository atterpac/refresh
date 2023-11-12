package main

import (
	"flag"
	"revolver/log"
	"revolver/watcher"
	"strings"
)

func main() {
	var rootPath string
	var ignoreList string
	var execCommand string
	var logLevel string

	flag.StringVar(&rootPath, "path", ".", "Root path to watch")
	flag.StringVar(&ignoreList, "ignore", "", "Comma-separated list of files to ignore")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "log", "", "Level to set Logs")
	flag.Parse()

	ignoreListSlice := strings.Split(ignoreList, ",")

	watch := watcher.Config{
		Label:       "Golang",
		RootPath:    rootPath,
		IgnoreList:  ignoreListSlice,
		ExecCommand: strings.Fields(execCommand),
		LogLevel:    logLevel,
		ColorScheme: log.ColorScheme{
			Info:   "#ccff33",
			Debug:  "#44aaee",
			Error:  "#ff3355",
			Warn:   "#ffcc55",
			Fatal:  "#771111",
		},
	}
	// watch := watcher.Config{
	// 	Label:       "Golang",
	// 	RootPath:    "../testProject",
	// 	IgnoreList:  []string{"newfile.go"},
	// 	ExecCommand: []string{"go", "run", "main.go"},
	// }

	watch.Start()

	<-make(chan struct{})
}
