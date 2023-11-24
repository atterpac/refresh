package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	refresh "github.com/atterpac/refresh/engine"
)

func main() {
	var version string = "0.2.0"

	var rootPath string
	var preExec string
	var preWait bool
	var execCommand string
	var postExec string
	var logLevel string
	var configPath string
	var debounce string
	var watch *refresh.Engine
	var versFlag bool

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string

	flag.StringVar(&rootPath, "p", "./", "Root path to watch")
	flag.StringVar(&execCommand, "e", "", "Command to execute on changes")
	flag.StringVar(&preExec, "be", "", "Command to execute before the exec command is ran")
	flag.BoolVar(&preWait, "w", false, "Boolean to decide if the exec should wait for the pre-exec to finish")
	flag.StringVar(&postExec, "ae", "", "Command to execute when a reload is dectected after the orginal exec completes")
	flag.StringVar(&logLevel, "l", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.StringVar(&ignoreDir, "id", "", "Ignore Directory list as comma-separated list")
	flag.StringVar(&ignoreFile, "if", "", "Ignore File list as comma-separated list")
	flag.StringVar(&ignoreExt, "ie", "", "Ignore Extension list as comma-separated list")
	flag.StringVar(&debounce, "d", "1000", "Debounce time in milliseconds")
	flag.BoolVar(&versFlag, "v", false, "Print version")
	flag.Parse()

	if versFlag {
		fmt.Println(PrintBanner(version))
		os.Exit(0)
	}

	// TODO: Make file config able to be overridden by cli
	if len(configPath) != 0 {
		watch = refresh.NewEngineFromTOML(configPath)
	} else {
		ignore := refresh.Ignore{
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
		config := refresh.Config{
			RootPath:    rootPath,
			PreWait:	 preWait, 
			PreExec:     preExec,
			PostExec:    postExec,
			ExecCommand: execCommand,
			LogLevel:    logLevel,
			Ignore:      ignore,
			Debounce:    debounceThreshold,
		}
		watch = refresh.NewEngineFromConfig(config)
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

func PrintBanner(ver string) string{
	return fmt.Sprintf(`
   ___  ___________  __________ __
  / _ \/ __/ __/ _ \/ __/ __/ // /
 / , _/ _// _// , _/ _/_\ \/ _  / 
/_/|_/___/_/ /_/|_/___/___/_//_/  
                                  v%s 
`, ver)
}
