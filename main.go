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

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string


	flag.StringVar(&rootPath, "path", "", "Root path to watch")
	flag.StringVar(&ignoreList, "ignore", "", "Comma-separated list of files to ignore")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "log", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.StringVar(&ignoreDir, "id", "", "Ignore Directory list as comma-separated list")
	flag.StringVar(&ignoreFile, "if", "", "Ignore File list as comma-separated list")
	flag.StringVar(&ignoreExt, "ie", "", "Ignore Extension list as comma-separated list")
	flag.Parse()

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
		ignore := watcher.Ignore{
			File:      strings.Split(ignoreFile, ","),
			Dir:       strings.Split(ignoreDir, ","),
			Extension: strings.Split(ignoreExt, ","),
		}
		// Root | Exec | Label | LogLevel | IgnoreList | Colors as log.ColorScheme
		watch = watcher.NewWatcher(rootPath, execCommand, "", logLevel, ignore, colors)
	}

	tui.Banner("Gotato v0.0.1")
	watch.Start()
	<-make(chan struct{})
}
