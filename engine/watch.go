package engine

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
)

func (engine *Engine) watch() {
	slog.Info("Watching for file changes...")
	// Create Channel for Events
	engine.Chan = make(chan notify.EventInfo, 1)
	defer notify.Stop(engine.Chan)
	// Mount watcher on route directory and subdirectories
	if err := notify.Watch("./...", engine.Chan, notify.All); err != nil {
		slog.Error(fmt.Sprintf("Creating watcher: %s", err.Error()))
	}
	watchEvents(engine, engine.Chan)
}

func watchEvents(engine *Engine, e chan notify.EventInfo) error {
	var debounceTime time.Time
	var debounceThreshold = time.Duration(engine.Config.Debounce) * time.Millisecond
	for {
		ei := <-e
		go func(ei notify.EventInfo) {
			eventInfo, ok := eventMap[ei.Event()]
			if !ok {
				slog.Error(fmt.Sprintf("Unknown Event: %s", ei.Event()))
				return
			}
			// Callback handling
			if engine.Config.Callback != nil {
				event := CallbackMap[ei.Event()]
				handle := engine.Config.Callback(&EventCallback{
					Type: event,
					Time: time.Now(),
					Path: getPath(ei.Path()),
				})
				switch handle {
				case EventContinue: // Continue with reload process as eventMap and ignore rules dictate
				case EventBypass: // Bypass all rulesets and reload process
					slog.Debug("Bypassing all rulesets and reloading process...")
					engine.reloadProcess()
					return
				case EventIgnore: // Ignore Event and continue with monitoring
					return
				default:
				}
			}
			if eventInfo.Reload {
				if engine.Config.Ignore.shouldIgnore(ei.Path()) {
					slog.Debug(fmt.Sprintf("Ignoring %s change: %s", ei.Event().String(), ei.Path()))
					return
				}
				// Check if we should debounce
				if checkDebounce(debounceTime, debounceThreshold) {
					debounceTime = time.Now()
					slog.Debug(fmt.Sprintf("Debounce Timer Start: %v", debounceTime))
				} else {
					slog.Debug(fmt.Sprintf("Debouncing file change: %s", ei.Path()))
					return
				}
				// Continue with reload
				relPath := getPath(ei.Path())
				slog.Info("File Modified...Reloading", "file", relPath)
				engine.reloadProcess()
			}
		}(ei)
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
