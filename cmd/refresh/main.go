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
	var version string = "0.4.4"

	var rootPath string
	var execCommand string
	var logLevel string
	var configPath string
	var debounce string
	var watch *refresh.Engine
	var versFlag bool
	var gitIgnore bool

	// Ignore
	var ignoreDir string
	var ignoreFile string
	var ignoreExt string

	flag.StringVar(&rootPath, "p", "./", "Root path to watch")
	flag.StringVar(&execCommand, "e", "", "Command to execute on changes")
	flag.StringVar(&logLevel, "l", "info", "Level to set Logs")
	flag.StringVar(&configPath, "f", "", "File to read config from")
	flag.StringVar(&ignoreDir, "id", "", "Ignore Directory list as comma-separated list")
	flag.StringVar(&ignoreFile, "if", "", "Ignore File list as comma-separated list")
	flag.StringVar(&ignoreExt, "ie", "", "Watched Extension list as comma-separated list")
	flag.StringVar(&debounce, "d", "1000", "Debounce time in milliseconds")
	flag.BoolVar(&versFlag, "v", false, "Print version")
	flag.BoolVar(&gitIgnore, "git", false, "Read from .gitignore")
	flag.Parse()

	if versFlag {
		fmt.Println(PrintBanner(version))
		os.Exit(0)
	}

	if len(configPath) != 0 {
		watch = refresh.NewEngineFromTOML(configPath)
	} else {
		ignore := refresh.Ignore{
			File:         strings.Split(ignoreFile, ","),
			Dir:          strings.Split(ignoreDir, ","),
			WatchedExten: strings.Split(ignoreExt, ","),
			IgnoreGit:    gitIgnore,
		}
		// Debounce string to int
		debounceThreshold, err := strconv.Atoi(debounce)
		if err != nil {
			fmt.Println("Error converting debounce to int")
			os.Exit(1)
		}
		config := refresh.Config{
			RootPath: rootPath,
			ExecList: strings.Split(execCommand, ","),
			LogLevel: logLevel,
			Ignore:   ignore,
			Debounce: debounceThreshold,
		}
		watch = refresh.NewEngineFromConfig(config)
	}

	err := watch.Start()
	if err != nil {
		os.Exit(1)
	}
	<-make(chan struct{})
}

func PrintBanner(ver string) string {
	return fmt.Sprintf(`
   ___  ___________  __________ __
  / _ \/ __/ __/ _ \/ __/ __/ // /
 / , _/ _// _// , _/ _/_\ \/ _  / 
/_/|_/___/_/ /_/|_/___/___/_//_/ CLI v%s  
`, ver)
}
