package watcher

import (
	"fmt"
	"gotato/log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
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
	Label       string   `toml:"label"`
	RootPath    string   `toml:"root_path"`
	ExecCommand string   `toml:"exec_command"`
	IgnoreList  []string `toml:"ignore"`
	LogLevel    string   `toml:"log_level"`
}

func NewWatcherFromConfig(confPath string) *Engine {
	conf := Engine{}
	conf.readConfigFile(confPath)
	return &conf
}

func (engine *Engine) Start() {
	styles := setColorScheme(engine.ColorScheme)
	engine.readConfigFile("./gotato.toml")
	engine.Log = log.NewStyledLogger(styles, engine.GetLogLevel())
	engine.Log.Info(fmt.Sprintf("Color Scheme %s", engine.ColorScheme))
	engine.Log.Info(fmt.Sprintf("Starting Watcher for %s", engine.Config.Label))
	engine.Monitor()
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
	if _, err := toml.DecodeFile(path, &engine); err != nil {
		fmt.Println(err)
	}
	fmt.Println("Config", engine.Config)
	return engine
}

type EventInfo struct {
	Name   string
	Reload bool
}

func setColorScheme(scheme log.ColorScheme) log.LogStyles {
	styles := log.LogStyles{}
	styles.Debug = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Debug))
	styles.Info = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Info))
	styles.Warn = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Warn))
	styles.Error = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Error))
	styles.Fatal = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Fatal)).Bold(true)
	return styles
}

// Top level function that takes in a WatchEngine and starts a goroutine with its out fsnotify.Watcher and ruleset
func (engine *Engine) Monitor() {
	// Start Exec Command
	if len(engine.Config.ExecCommand) == 0 {
		engine.Log.Fatal("No Exec Command Provided")
	}
	engine.Process = Reload(*engine)
	// Create Channel for Events
	e := make(chan notify.EventInfo, 1)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch(engine.Config.RootPath, e, notify.All); err != nil {
		engine.Log.Error("Error creating watcher", err.Error())
	}
	defer notify.Stop(e)
	watchEvents(engine, e)
}

func containsIgnore(ignore []string, path string) bool {
	for _, ignorePath := range ignore {
		if path == ignorePath || filepath.Base(path) == ignorePath {
			return true
		}
	}
	return false
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

		engine.Log.Info(fmt.Sprintf("Event: %s", eventInfo.Name))
		if eventInfo.Reload {
			engine.Log.Info("Reloading")
			engine.Process = Reload(*engine)
		}
	}
}
