package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	gotato "github.com/atterpac/gotato/engine"
)

func main() {
	var rootPath string
	var execCommand string
	var logLevel string
	var configPath string
	var debounce string
	var watch *gotato.Engine

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string

	flag.StringVar(&rootPath, "p", "", "Root path to watch")
	flag.StringVar(&execCommand, "e", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "l", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.StringVar(&ignoreDir, "id", "", "Ignore Directory list as comma-separated list")
	flag.StringVar(&ignoreFile, "if", "", "Ignore File list as comma-separated list")
	flag.StringVar(&ignoreExt, "ie", "", "Ignore Extension list as comma-separated list")
	flag.StringVar(&debounce, "d", "1000", "Debounce time in milliseconds")
	flag.Parse()

	// TODO: Make file config able to be overridden by cli
	if len(configPath) != 0 {
		watch = gotato.NewEngineFromTOML(configPath)
	} else {
		ignore := gotato.Ignore{
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
		config := gotato.Config{
			RootPath:    rootPath,
			ExecCommand: execCommand,
			LogLevel:    logLevel,
			Ignore:      ignore,
			Debounce:    debounceThreshold,
		}
		watch = gotato.NewEngineFromConfig(config)
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
