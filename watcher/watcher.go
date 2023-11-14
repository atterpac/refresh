package watcher

import (
	"fmt"
	"gotato/log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/rjeczalik/notify"
)

type Engine struct {
	Process     *os.Process
	Active      bool
	Config      Config          `toml:"config"`
	ColorScheme log.ColorScheme `toml:"color_scheme"`
	Log         log.Logger
	LogStyles   log.LogStyles
}

type Config struct {
	IsFile      bool     `toml:"-"`
	Path        string   `toml:"config_path"`
	Label       string   `toml:"label"`
	RootPath    string   `toml:"root_path"`
	ExecCommand string   `toml:"exec_command"`
	IgnoreList  []string `toml:"ignore"`
	LogLevel    string   `toml:"log_level"`
}

func (engine *Engine) Start() {
	engine.Log.Info(fmt.Sprintf("Starting Watcher for %s", engine.Config.Label))
	engine.Monitor()
}

func NewWatcher(rootPath, execCommand, label, logLevel string, ignoreList []string, colors log.ColorScheme) *Engine {
	engine := Engine{}
	engine.Log = log.NewStyledLogger(engine.ColorScheme, engine.GetLogLevel())
	engine.Config = Config{
		RootPath:    rootPath,
		ExecCommand: execCommand,
		Label:       label,
		LogLevel:    logLevel,
		IgnoreList:  ignoreList,
	}
	engine.verifyConfig()
	return &engine
}

func NewWatcherFromConfig(confPath string) *Engine {
	engine := Engine{}
	engine.readConfigFile(confPath)
	engine.Log = log.NewStyledLogger(engine.ColorScheme, engine.GetLogLevel())
	engine.verifyConfig()
	return &engine
}

func (engine *Engine) GetLogLevel() int {
	switch engine.Config.LogLevel {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}

func (engine *Engine) readConfigFile(path string) *Engine {
	fmt.Println("Reading Config File", engine.Config.Path)
	if _, err := toml.DecodeFile(path, &engine); err != nil {
		fmt.Println("Error reading config file")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return engine
}

func (engine *Engine) verifyConfig() {
	engine.Log.Debug("Verifying Config")
	config := engine.Config
	engine.Log.Debug(fmt.Sprintf("Config: %+v", config))
	if config.RootPath == "" {
		engine.Log.Fatal("ERROR: Root Path not set")
		os.Exit(1)
	}
	if config.ExecCommand == "" {
		engine.Log.Fatal("ERROR: Exec Command not set")
		os.Exit(1)
	}
	if config.Label == "" {
		engine.Log.Warn("Label not set")
	}
}

type EventInfo struct {
	Name   string
	Reload bool
}

// Top level function that takes in a WatchEngine and starts a goroutine with its out fsnotify.Watcher and ruleset
func (engine *Engine) Monitor() {
	// Start Exec Command
	engine.Process = Reload(*engine)
	// Create Channel for Events
	e := make(chan notify.EventInfo, 1)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch(engine.Config.RootPath+"/...", e, notify.All); err != nil {
		engine.Log.Error("Error creating watcher", err.Error())
	}
	defer notify.Stop(e)
	watchEvents(engine, e)
}

func watchEvents(engine *Engine, e chan notify.EventInfo) {
	for {
		ei := <-e
		if containsIgnore(engine.Config.IgnoreList, ei.Path()) {
			continue
		}

		eventInfo, ok := eventMap[ei.Event()]
		if !ok {
			engine.Log.Error(fmt.Sprintf("Unknown Event: %s", ei.Event()))
			continue
		}

		engine.Log.Debug(fmt.Sprintf("Event: %s | %s", eventInfo.Name, ei.Path()))
		if eventInfo.Reload {
			relPath := getPath(engine.Log, ei.Path())
			engine.Log.Info(fmt.Sprintf("File Modified: %s", relPath))
			engine.Log.Info("Reloading...")
			engine.Process = Reload(*engine)
		}
	}
}

func containsIgnore(ignore []string, path string) bool {
	for _, ignorePath := range ignore {
		if path == ignorePath || filepath.Base(path) == ignorePath {
			return true
		}
	}
	return false
}


func getPath(log log.Logger, path string) string {
	wd, err := os.Getwd()
	if err != nil {
		log.Error(fmt.Sprintf("Error getting working directory: %s", err.Error()))
		return ""
	}
	relPath, err := stripCurrentDirectory(path, wd)
	if err != nil {
		log.Error(fmt.Sprintf("Error stripping current directory: %s", err.Error()))
		return ""
	}
	return relPath
}

func stripCurrentDirectory(fullPath, currentDirectory string) (string, error) {
	relativePath, err := filepath.Rel(currentDirectory, fullPath)
	if err != nil {
		return "", err
	}

	return relativePath, nil
}
