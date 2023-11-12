package watcher

import (
	"fmt"
	"os"
	"revolver/config"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/rjeczalik/notify"
)

type WatchEngine struct {
	Watchers []Watcher `toml:"-"`
	mu       sync.Mutex // Mutex to synchronize access to Watchers
}

type Watcher struct {
	Process *os.Process
	Active  bool
	Config  config.Config
}

func (watcher *Watcher) Start() {
	log.SetLevel(log.InfoLevel)
	log.Debug(fmt.Sprintf("Starting Watcher for %s", watcher.Config.Label))
	Monitor(watcher)
}

// Top level function that takes in a WatchEngine and starts a goroutine with its out fsnotify.Watcher and ruleset
func Monitor(conf *Watcher) {
	// Create new watcher.
	// Create Watchlist
	// err = createWatcherTree(conf)
	// if err != nil {
	// 	log.Error("Error creating watcher tree")
	// 	return
	// }
	conf.Process = Reload(*conf)
	// Start listening for events.
	e := make(chan notify.EventInfo, 1)

	if err := notify.Watch(conf.Config.RootPath+"/...", e, notify.All); err != nil {
		log.Error("Error creating watcher")
	}

	defer notify.Stop(e)
	// Initial Load
	watchEvents(conf ,e)
}

// Watches fs.Notify events based on rules inside the provided WatchEngine
func watchEvents(conf *Watcher,e chan notify.EventInfo) {
	for {
		// Log Watchlist
		switch ei := <-e; ei.Event() {
		// case notify.InCloseWrite:
		// 	log.Info(fmt.Sprintf("Event: %s", ei.Event()))
		// 	log.Info(fmt.Sprintf("Created: %s", ei.Path()))
		// 	// addWatch(ei.Path(), watcher)
		case notify.Write:
			log.Info(fmt.Sprintf("Modified: %s", ei))
			conf.Process = Reload(*conf)
		// 	// addWatch(ei.Path(), watcher) default:
		// case notify.Remove:
		// 	log.Info(fmt.Sprintf("Event: %s", ei.Event()))
		// 	log.Info(fmt.Sprintf("Removed: %s", ei.Path()))
		// 	conf.Process = Reload(*conf)
		// 	// watcher.Watcher.Remove(ei.Path())
		// case notify.Rename:
		// 	log.Info(fmt.Sprintf("Event: %s", ei.Event()))
		// 	log.Info(fmt.Sprintf("Renamed: %s", ei.Path()))
		// 	// watcher.Watcher.Remove(ei.Path())
		}
	}
}
// Generates a list of files in the root directory
// func createWatcherTree(watcher *Watcher) error {
// 	files, err := getMonitoredPaths(watcher.Config.RootPath)
// 	if err != nil {
// 		fmt.Println("Error getting monitored files\n Exiting Now...")
// 		return err
// 	}
// 	for _, file := range files {
// 		err := addWatch(file, watcher)
// 		if err != nil {
// 			fmt.Println("Error adding file to watcher: ", file)
// 		}
// 	}
// 	return nil
// }

// Adds a file to the watcher
// func addWatch(path string, watcher *Watcher) error {
// 	// Dont add ignored files to watcher
// 	log.Debug(fmt.Sprintf("Adding %s to watcher", path))
// 	if containsIgnore(watcher.Config.IgnoreList, path) {
// 		return nil
// 	}
// 	err := watcher.Watcher.Add(path)
// 	if err != nil {
// 		log.Error(fmt.Sprintf("Error adding %s to watcher", path))
// 		return err
// 	}
// 	return nil
// }

// func getMonitoredPaths(dirPath string) ([]string, error) {
// 	var acceptedDirs []string
// 	err := walkPath(dirPath, &acceptedDirs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return acceptedDirs, nil
// }

// func walkPath(dirPath string, acceptedDirs *[]string) error {
// 	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		*acceptedDirs = append(*acceptedDirs, path)
// 		return nil
// 	})
// 	return err
// }
//
// func containsIgnore(ignore []string, path string) bool {
// 	for _, ignorePath := range ignore {
// 		if path == ignorePath || filepath.Base(path) == ignorePath {
// 			return true
// 		}
// 	}
// 	return false
// }
