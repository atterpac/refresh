package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type WatchEngine struct {
	Watcher  *fsnotify.Watcher
	Label    string
	Active bool
	Pid      int
	Config   WatchConfig
}

type WatchConfig struct {
	RootPath    string
	ExecCommand []string
	IgnoreList  []string
}

// Top level function that takes in a WatchEngine and starts a goroutine with its out fsnotify.Watcher and ruleset
func Monitor(conf *WatchEngine) {
	// Create new watcher.
	var err error
	conf.Watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer conf.Watcher.Close()
	// Create Watchlist
	err = createWatcherTree(conf)
	if err != nil {
		return
	}
	// Start listening for events.
	go watchEvents(conf)
	// Inital Load
	conf.Pid = Reload(*conf)
}

// Watches fs.Notify events based on rules inside the provided WatchEngine
func watchEvents(conf *WatchEngine) {
	for {
		if !conf.Active {
			return
		}
		select {
		case event, ok := <-conf.Watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				log.Println("modified file:", event.Name)
				conf.Pid = Reload(*conf)
			}
		case err, ok := <-conf.Watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

// Generates a list of files in the root directory
func createWatcherTree(engine *WatchEngine) error {
	files, err := getMonitoredPaths(engine.Config.RootPath)
	if err != nil {
		fmt.Println("Error getting monitored files\n Exiting Now...")
		return err
	}
	for _, file := range files {
		err := addWatch(file, engine)
		if err != nil {
			fmt.Println("Error adding file to watcher: ", file)
		}
	}
	return nil
}

// Adds a file to the watcher
func addWatch(path string, engine *WatchEngine) error {
	// Dont add ignored files to watcher
	if containsIgnore(engine.Config.IgnoreList, path) {
		return nil
	}
	err := engine.Watcher.Add(path)
	if err != nil {
		return err
	}
	return nil
}

func getMonitoredPaths(dirPath string) ([]string, error) {
	var acceptedDirs []string
	err := walkPath(dirPath, &acceptedDirs)
	if err != nil {
		return nil, err
	}
	return acceptedDirs, nil
}

func walkPath(dirPath string, acceptedDirs *[]string) error {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		*acceptedDirs = append(*acceptedDirs, path)
		return nil
	})
	return err
}

func containsIgnore(ignore []string, path string) bool {
	for _, ignorePath := range ignore {
		if path == ignorePath || filepath.Base(path) == ignorePath {
			return true
		}
	}
	return false
}
