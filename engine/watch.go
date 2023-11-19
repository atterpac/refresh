package engine

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
)

type EventInfo struct {
	Name   string
	Reload bool
}

func (engine *Engine) watch() {
	// Start Exec Command
	engine.Process = engine.reloadProcess()
	// Create Channel for Events
	engine.Chan = make(chan notify.EventInfo, 1)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch(engine.Config.RootPath+"/...", e, notify.All); err != nil {
		slog.Error(fmt.Sprintf("Error creating watcher: %s", err.Error()))
	}
	defer notify.Stop(engine.Chan)
	slog.Warn("Watching for file changes...")
	watchEvents(engine, engine.Chan)
}

func watchEvents(engine *Engine, e chan notify.EventInfo) {
	var debounceTime time.Time

	var debounceThreshold = time.Duration(engine.Config.Debounce) * time.Millisecond
	for {
		ei := <-e
		eventInfo, ok := eventMap[ei.Event()]
		if !ok {
			slog.Error(fmt.Sprintf("Unknown Event: %s", ei.Event()))
			continue
		}
		if eventInfo.Reload {
			// Check if file should be ignored
			if engine.Config.Ignore.checkIgnore(ei.Path()) {
				slog.Debug(fmt.Sprintf("Ignoring %s change: %s", ei.Event().String(), ei.Path()))
				continue
			}
			// Check if we should debounce
			if checkDebounce(debounceTime, debounceThreshold) {
				debounceTime = time.Now()
				slog.Debug(fmt.Sprintf("Debounce Timer Start: %v", debounceTime))
			} else {
				slog.Debug(fmt.Sprintf("Debouncing file change: %s", ei.Path()))
				continue
			}
			// Continue with reload
			relPath := getPath(ei.Path())
			slog.Warn(fmt.Sprintf("File Modified: %s", relPath))
			slog.Warn("Reloading process...")
			engine.Process = engine.reloadProcess()
		}
	}
}

func checkDebounce(debounceTime time.Time, debounceThreshold time.Duration) bool {
	return time.Now().After(debounceTime.Add(debounceThreshold))
}

func getPath(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		slog.Error(fmt.Sprintf("Error getting working directory: %s", err.Error()))
		return ""
	}
	relPath, err := stripCurrentDirectory(path, wd)
	if err != nil {
		slog.Error(fmt.Sprintf("Error stripping current directory: %s", err.Error()))
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
