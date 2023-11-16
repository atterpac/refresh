package main

import (
	"flag"
	"fmt"
	"gotato/log"
	"gotato/tui"
	"gotato/watcher"
	"os"
	"strconv"
	"strings"
)

func main() {
	var rootPath string
	var ignoreList string
	var execCommand string
	var logLevel string
	var configPath string
	var debounce string
	var chunkSize string
	var label string
	var watch *watcher.Engine

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string


	flag.StringVar(&rootPath, "path", "", "Root path to watch")
	flag.StringVar(&ignoreList, "ignore", "", "Comma-separated list of files to ignore")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&label, "l", "", "Label for sub-process")
	flag.StringVar(&logLevel, "log", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.StringVar(&ignoreDir, "id", "", "Ignore Directory list as comma-separated list")
	flag.StringVar(&ignoreFile, "if", "", "Ignore File list as comma-separated list")
	flag.StringVar(&ignoreExt, "ie", "", "Ignore Extension list as comma-separated list")
	flag.StringVar(&chunkSize, "chunk", "10", "Chunk size for log output")
	flag.StringVar(&debounce, "d", "1000", "Debounce time in milliseconds")
	flag.Parse()

	// TODO: Make file config able to be overridden by cli
	tui.Banner("Gotato v0.0.1")
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
			File:      stringSliceToMap(strings.Split(ignoreFile, ",")),
			Dir:       stringSliceToMap(strings.Split(ignoreDir, ",")),
			Extension: stringSliceToMap(strings.Split(ignoreExt, ",")),
		}

		debounceThreshold, err := strconv.Atoi(debounce)
		if err != nil {
			fmt.Println("Error converting debounce to int")
			os.Exit(1)
		}
		// Root | Exec | Label | LogLevel | IgnoreList | Colors as log.ColorScheme | Debounce
		// Debounce string to int
		watch = watcher.NewWatcher(rootPath, execCommand, "", logLevel, ignore, colors, debounceThreshold, chunkSize)
	}

	watch.Start()
	<-make(chan struct{})
}

func stringSliceToMap(slice []string) map[string]bool {
	m := make(map[string]bool)
	for _, v := range slice {
		m[v] = true
	}
	return m
}

