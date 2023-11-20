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

// Called whenever a change is detected in the filesystem
// By default we ignore file rename/remove and a bunch of other events that would likely cause breaking changes on a reload  see eventmap_[oos].go for default rules
// Callback returns two booleans reload and bypass
// reload: if true will reload the process as long as the eventMap allows it
// bypass: if true will bypass the eventMap and reload the process regardless of the eventMap instruction
type EventCallback struct {
	Name string    // rjeczalik/notify.[EVENT]
	Time time.Time // time.Now() when event was triggered
	Path string    // Full path to the modified file
}

func (engine *Engine) watch() {
	// Start Exec Command
	engine.Process = engine.reloadProcess()
	// Create Channel for Events
	engine.Chan = make(chan notify.EventInfo, 1)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch(engine.Config.RootPath+"/...", engine.Chan, notify.All); err != nil {
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
		if engine.Config.Callback != nil {
			reload, bypass := engine.Config.Callback(&EventCallback{
				Name: ei.Event().String(),
				Time: time.Now(),
				Path: ei.Path(),
			})
			if !reload {
				continue
			}
			if bypass && reload {
				engine.Process = engine.reloadProcess()
			}
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
			slog.Info(fmt.Sprintf("File Modified: %s", relPath))
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
