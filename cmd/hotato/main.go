package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	hotato "github.com/atterpac/hotato/engine"
)

func main() {
	var version string = "0.0.26"

	var rootPath string
	var preExec string
	var execCommand string
	var postExec string
	var logLevel string
	var configPath string
	var debounce string
	var watch *hotato.Engine
	var versFlag bool

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string

	flag.StringVar(&rootPath, "p", "./", "Root path to watch")
	flag.StringVar(&execCommand, "e", "", "Command to execute on changes")
	flag.StringVar(&preExec, "be", "", "Command to execute before the exec command is ran")
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
		watch = hotato.NewEngineFromTOML(configPath)
	} else {
		ignore := hotato.Ignore{
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
		config := hotato.Config{
			RootPath:    rootPath,
			PreExec:     preExec,
			PostExec:    postExec,
			ExecCommand: execCommand,
			LogLevel:    logLevel,
			Ignore:      ignore,
			Debounce:    debounceThreshold,
		}
		watch = hotato.NewEngineFromConfig(config)
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
      ___           ___                       ___                       ___     
     /__/\         /  /\          ___        /  /\          ___        /  /\    
     \  \:\       /  /::\        /  /\      /  /::\        /  /\      /  /::\   
      \__\:\     /  /:/\:\      /  /:/     /  /:/\:\      /  /:/     /  /:/\:\  
  ___ /  /::\   /  /:/  \:\    /  /:/     /  /:/~/::\    /  /:/     /  /:/  \:\ 
 /__/\  /:/\:\ /__/:/ \__\:\  /  /::\    /__/:/ /:/\:\  /  /::\    /__/:/ \__\:\
 \  \:\/:/__\/ \  \:\ /  /:/ /__/:/\:\   \  \:\/:/__\/ /__/:/\:\   \  \:\ /  /:/
  \  \::/       \  \:\  /:/  \__\/  \:\   \  \::/      \__\/  \:\   \  \:\  /:/ 
   \  \:\        \  \:\/:/        \  \:\   \  \:\           \  \:\   \  \:\/:/  
    \  \:\        \  \::/          \__\/    \  \:\           \__\/    \  \::/   
     \__\/         \__\/                     \__\/                     \__\/    v%s 
`, ver)
}
