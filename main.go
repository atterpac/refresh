package main

import (
	"flag"
	"fmt"
	"revolver/watcher"
	"strings"

	"github.com/charmbracelet/log"
)

func main() {
	log.SetLevel(log.DebugLevel)
	var rootPath string
	var ignoreList string
	var execCommand string
	var logLevel string

	flag.StringVar(&rootPath, "path", ".", "Root path to watch")
	flag.StringVar(&ignoreList, "ignore", "", "Comma-separated list of files to ignore")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "log", "", "Level to set Logs")
	flag.Parse()


	log.Info(fmt.Sprintf("Ignore list: %s", ignoreList))
	ignoreListSlice := strings.Split(ignoreList, ",")

	watch := watcher.Config{
		Label:       "Golang",
		RootPath:    rootPath,
		IgnoreList:  ignoreListSlice,
		ExecCommand: strings.Fields(execCommand),
		LogLevel:   logLevel,
	}
	log.Info("Starting Watcher")
	// watch := watcher.Config{
	// 	Label:       "Golang",
	// 	RootPath:    "../testProject",
	// 	IgnoreList:  []string{"newfile.go"},
	// 	ExecCommand: []string{"go", "run", "main.go"},
	// }

	watch.Start()

	<-make(chan struct{})
}
