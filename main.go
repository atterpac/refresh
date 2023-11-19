package main

import (
	"flag"
	"fmt"
	"gotato/engine"
	"os"
	"strconv"
	"strings"
)

func main() {
	var rootPath string
	var execCommand string
	var logLevel string
	var configPath string
	var debounce string
	var label string
	var watch *engine.Engine

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string

	flag.StringVar(&rootPath, "path", "", "Root path to watch")
	flag.StringVar(&execCommand, "exec", "", "Command to execute on changes")
	flag.StringVar(&label, "l", "", "Label for sub-process")
	flag.StringVar(&logLevel, "log", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.StringVar(&ignoreDir, "id", "", "Ignore Directory list as comma-separated list")
	flag.StringVar(&ignoreFile, "if", "", "Ignore File list as comma-separated list")
	flag.StringVar(&ignoreExt, "ie", "", "Ignore Extension list as comma-separated list")
	flag.StringVar(&debounce, "d", "1000", "Debounce time in milliseconds")
	flag.Parse()

	// TODO: Make file config able to be overridden by cli
	if len(configPath) != 0 {
		watch = engine.NewEngineFromTOML(configPath)
	} else {
		ignore := engine.Ignore{
			File:      stringSliceToMap(strings.Split(ignoreFile, ",")),
			Dir:       stringSliceToMap(strings.Split(ignoreDir, ",")),
			Extension: stringSliceToMap(strings.Split(ignoreExt, ",")),
		}
		debounceThreshold, err := strconv.Atoi(debounce)
		if err != nil {
			fmt.Println("Error converting debounce to int")
			os.Exit(1)
		}
		// Debounce string to int
		config := engine.Config{
			RootPath:    rootPath,
			ExecCommand: execCommand,
			Label:       label,
			LogLevel:    logLevel,
			Ignore:      ignore,
			Debounce:    debounceThreshold,
		}
		watch = engine.NewEngineFromConfig(config)
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
