package main

import (
	"flag"
	"gotato/log"
	"gotato/tui"
	"gotato/watcher"
	"strings"
)

func main() {
	var rootPath string
	var ignoreList string
	var execCommand string
	var logLevel string
	var configPath string
	var watch *watcher.Engine

	flag.StringVar(&rootPath, "path", "", "Root path to watch")
	flag.StringVar(&ignoreList, "ignore", "", "Comma-separated list of files to ignore")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "log", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.Parse()

	ignoreListSlice := strings.Split(ignoreList, ",")
	// TODO: Make file config able to be overridden by cli
	if len(configPath) != 0 {
		watch = watcher.NewWatcherFromConfig(configPath)
	} else {
		colors := log.ColorScheme{
			Info:  "#ccff33",
			Debug: "#44aaee",
			Error: "#ff3355",
			Warn:  "#ffcc55",
			Fatal: "#771111",
		}
		watch = watcher.NewWatcher(rootPath, execCommand, "", logLevel, ignoreListSlice, colors)
	}

	tui.Banner("Gotato v0.0.1")
	watch.Start()
	<-make(chan struct{})
}

func isFileConfig(path string) bool {
	return path != ""
}
