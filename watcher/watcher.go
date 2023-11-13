package watcher

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/rjeczalik/notify"
	"gotato/log"
	"os"
	"path/filepath"
)

type Config struct {
	Process     *os.Process
	Active      bool
	Label       string   `toml:"label"`
	RootPath    string   `toml:"root_path"`
	ExecCommand []string `toml:"exec_command"`
	IgnoreList  []string `toml:"ignore_list"`
	LogLevel    string   `toml:"log_level"`
	Log         log.Logger
	ColorScheme log.ColorScheme
	LogStyles   log.LogStyles
}

func (conf *Config) Start() {
	styles := setColorScheme(conf.ColorScheme)
	conf.Log = log.NewStyledLogger(styles, conf.GetLogLevel())
	conf.Log.Info(fmt.Sprintf("Starting Watcher for %s", conf.Label))
	conf.Monitor()
}

func (conf *Config) GetLogLevel() int {
	switch conf.LogLevel {
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
func (conf *Config) Monitor() {
	// Start Exec Command
	if len(conf.ExecCommand) == 0 {
		conf.Log.Fatal("No Exec Command Provided")
	}
	conf.Process = Reload(*conf)
	// Create Channel for Events
	e := make(chan notify.EventInfo, 1)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch(conf.RootPath+"/...", e, notify.All); err != nil {
		conf.Log.Error("Error creating watcher")
	}
	defer notify.Stop(e)
	watchEvents(conf, e)
}

func containsIgnore(ignore []string, path string) bool {
	for _, ignorePath := range ignore {
		if path == ignorePath || filepath.Base(path) == ignorePath {
			return true
		}
	}
	return false
}

func watchEvents(conf *Config, e chan notify.EventInfo) {
	for {
		ei := <-e
		if containsIgnore(conf.IgnoreList, ei.Path()) {
			continue
		}

		eventInfo, ok := eventMap[ei.Event()]
		if !ok {
			conf.Log.Error(fmt.Sprintf("Unknown Event: %s", ei.Event()))
			continue
		}

		conf.Log.Info(fmt.Sprintf("Event: %s", eventInfo.Name))
		if eventInfo.Reload {
			conf.Log.Info("Reloading")
			conf.Process = Reload(*conf)
		}
	}
}
