package watcher

import (
	"fmt"
	"gotato/log"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/shirou/gopsutil/process"
)

type Engine struct {
	Process     *process.Process
	Active      bool
	Config      Config          `toml:"config"`
	ColorScheme log.ColorScheme `toml:"color_scheme"`
	Log         log.Logger
}


type EventInfo struct {
	Name   string
	Reload bool
}

func (engine *Engine) Start() {
	engine.Log.Info(fmt.Sprintf("Starting Watcher for %s", engine.Config.Label))
	engine.Monitor()
}

func NewWatcher(rootPath, execCommand, label, logLevel string, ignore Ignore, colors log.ColorScheme, debounce int) *Engine {
	engine := Engine{}
	engine.Log = log.NewStyledLogger(engine.ColorScheme, engine.GetLogLevel())
	engine.Config = Config{
		RootPath:    rootPath,
		ExecCommand: execCommand,
		Label:       label,
		LogLevel:    logLevel,
		Ignore:      ignore,
		Debounce:    debounce,
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
	var debounceTime time.Time
	var debounceThreshold = time.Duration(engine.Config.Debounce) * time.Millisecond
	for {
		ei := <-e
		eventInfo, ok := eventMap[ei.Event()]
		if !ok {
			engine.Log.Error(fmt.Sprintf("Unknown Event: %s", ei.Event()))
			continue
		}
		if eventInfo.Reload {
			// Check if file should be ignored
			if engine.Config.Ignore.CheckIgnore(ei.Path()) {
				engine.Log.Debug(fmt.Sprintf("Ignoring %s change: %s", ei.Event().String() ,ei.Path()))
				continue
			}
			// Check if we should debounce
			if checkDebounce(debounceTime, debounceThreshold ) {
				debounceTime = time.Now()
				engine.Log.Debug(fmt.Sprintf("Debounce Timer Start: %v", debounceTime))
			} else {
				engine.Log.Debug(fmt.Sprintf("Debouncing file change: %s", ei.Path()))
				continue
			}
			// Continue with reload
			relPath := getPath(engine.Log, ei.Path())
			engine.Log.Info(fmt.Sprintf("File Modified: %s", relPath))
			engine.Log.Info("Reloading...")
			engine.Process = Reload(*engine)
		}
	}
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

func checkDebounce(debounceTime time.Time, debounceThreshold time.Duration) bool {
	return time.Now().After(debounceTime.Add(debounceThreshold)) 
}
