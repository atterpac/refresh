package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/rjeczalik/notify"
)

type Config struct {
	Process     *os.Process
	Active      bool
	Label       string   `toml:"label"`
	RootPath    string   `toml:"root_path"`
	ExecCommand []string `toml:"exec_command"`
	IgnoreList  []string `toml:"ignore_list"`
	LogLevel    string   `toml:"log_level"`
}

func (watcher *Config) Start() {
	setLogLevel(watcher)
	log.Debug(fmt.Sprintf("Starting Watcher for %s", watcher.Label))
	Monitor(watcher)
}

// Top level function that takes in a WatchEngine and starts a goroutine with its out fsnotify.Watcher and ruleset
func Monitor(conf *Config) {
	// Start Exec Command
	if len(conf.ExecCommand) == 0 {
		log.Fatal("No Exec Command Provided")
	}
	conf.Process = Reload(*conf)
	// Create Channel for Events
	e := make(chan notify.EventInfo, 1)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch(conf.RootPath+"/...", e, notify.All); err != nil {
		log.Error("Error creating watcher")
	}
	defer notify.Stop(e)
	// Initial Load
	if runtime.GOOS == "linux" {
		log.Debug("Linux Detected")
		watchLinux(conf, e)
	} else {
		watchEvents(conf, e)
	}
}

func containsIgnore(ignore []string, path string) bool {
	for _, ignorePath := range ignore {
		if path == ignorePath || filepath.Base(path) == ignorePath {
			log.Info(fmt.Sprintf("Ignoring Modification: %s", path))
			return true
		}
	}
	return false
}

// Watches fs.Notify events based on rules inside the provided WatchEngine
func watchLinux(conf *Config, e chan notify.EventInfo) {
	for {
		ei := <-e
		if containsIgnore(conf.IgnoreList, ei.Path()) {
			continue
		}
		switch ei.Event() {
		case notify.InCloseWrite:
			log.Info(fmt.Sprintf("Write: %s", ei.Path()))
			conf.Process = Reload(*conf)
		case notify.InModify:
			log.Info(fmt.Sprintf("Modified: %s", ei))
			conf.Process = Reload(*conf)
		case notify.InMovedTo:
			log.Info(fmt.Sprintf("MovedTo: %s", ei))
			conf.Process = Reload(*conf)
		case notify.InMovedFrom:
			log.Info(fmt.Sprintf("MovedFrom: %s", ei))
			conf.Process = Reload(*conf)
		case notify.InCreate:
			log.Info(fmt.Sprintf("Created: %s", ei))
			conf.Process = Reload(*conf)
		case notify.InDelete:
			log.Info(fmt.Sprintf("Deleted: %s", ei))
			conf.Process = Reload(*conf)
		// Base Events in case linux emits them
		case notify.Write:
			log.Info(fmt.Sprintf("Write: %s", ei.Path()))
			conf.Process = Reload(*conf)
		case notify.Create:
			log.Debug(fmt.Sprintf("Created: %s", ei.Path()))
		case notify.Remove:
			log.Debug(fmt.Sprintf("Removed: %s", ei.Path()))
		case notify.Rename:
			log.Debug(fmt.Sprintf("Renamed: %s", ei.Path()))
		}
	}
}

func watchEvents(conf *Config, e chan notify.EventInfo) {
	for {
		ei := <-e
		if containsIgnore(conf.IgnoreList, ei.Path()) {
			continue
		}
		switch ei.Event() {
		case notify.Write:
			log.Info(fmt.Sprintf("Write: %s", ei.Path()))
			conf.Process = Reload(*conf)
		case notify.Create:
			log.Info(fmt.Sprintf("Created: %s", ei.Path()))
		case notify.Remove:
			log.Info(fmt.Sprintf("Removed: %s", ei.Path()))
		case notify.Rename:
			log.Info(fmt.Sprintf("Renamed: %s", ei.Path()))
		}
	}
}

func setLogLevel(conf *Config) {
	switch conf.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
