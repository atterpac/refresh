//go:build linux || darwin || windows
// +build linux darwin windows

package engine

import (
	"context"
	// "log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/atterpac/refresh/process"
	"github.com/rjeczalik/notify"
)

type EventManager struct {
	engine            *Engine
	lastEventTime     time.Time
	debounceThreshold time.Duration
	debounceTimer     *time.Timer
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewEventManager(engine *Engine, debounce int) *EventManager {
	ctx, cancel := context.WithCancel(context.Background())
	em := &EventManager{
		engine:            engine,
		debounceThreshold: time.Duration(debounce) * time.Millisecond,
		ctx:               ctx,
		cancel:            cancel,
	}
	return em
}

func (em *EventManager) HandleEvent(ei notify.EventInfo) {
	eventInfo, ok := EventMap[ei.Event()]
	if !ok {
		// slog.Error("Unknown event", "event", ei.Event())
		return
	}

	if em.engine.Config.Callback != nil {
		event := CallbackMap[ei.Event()]
		handle := em.engine.Config.Callback(&EventCallback{
			Type: event,
			Path: getPath(ei.Path()),
			Time: time.Now(),
		})
		switch handle {
		case EventContinue:
			// Continue
		case EventBypass:
			// slog.Debug("Bypassing event", "event", ei.Event(), "path", ei.Path())
			return
		case EventIgnore:
			// slog.Debug("Ignoring event", "event", ei.Event(), "path", ei.Path())
			return
		default:
		}
	}

	if eventInfo.Reload {
		newCtx, newCancel := context.WithCancel(context.Background())
		em.engine.ctx = newCtx
		em.engine.cancel = newCancel
		if em.engine.Config.Ignore.shouldIgnore(ei.Path()) {
			return
		}
		// slog.Debug("Event", "event", ei.Event(), "path", ei.Path(), "time", time.Now())
		currentTime := time.Now()
		if currentTime.Sub(em.lastEventTime) >= em.debounceThreshold {
			// slog.Debug("Setting debounce timer", "event", ei.Event(), "path", ei.Path(), "time", time.Now())
			// slog.Info("File modified...Refreshing", "file", getPath(ei.Path()))

			// Find the specific process associated with the file change event
			for _, p := range em.engine.ProcessManager.Processes {
				if p.Type == process.Primary {
					// Kill the specific process by canceling its context
					if cancel, ok := em.engine.ProcessManager.Cancels[p.Exec]; ok {
						cancel()
						delete(em.engine.ProcessManager.Ctxs, p.Exec)
						delete(em.engine.ProcessManager.Cancels, p.Exec)
					}
					break
				}
			}

			// Start a new instance of the process
			go em.engine.ProcessManager.StartProcess(em.engine.ctx, em.engine.cancel)
			go func() {
				<-em.engine.ctx.Done()
				if em.engine.ctx.Err() == context.Canceled {
					if !em.engine.ProcessManager.FirstRun {
						// slog.Error("Could not refresh processes due to build errors")
						newCtx, newCancel := context.WithCancel(context.Background())
						em.engine.ctx = newCtx
						em.engine.cancel = newCancel
						return
					}
					em.engine.Stop()
				}
			}()

			em.lastEventTime = currentTime
		} else {
			// slog.Debug("Debouncing event", "event", ei.Event(), "path", ei.Path(), "time", time.Now())
		}
	}
}

func (engine *Engine) watch(eventManager *EventManager) {
	// slog.Info("Watching", "path", engine.Config.RootPath)
	engine.Chan = make(chan notify.EventInfo, 1)
	defer notify.Stop(engine.Chan)

	if err := notify.Watch(engine.Config.RootPath+"/...", engine.Chan, notify.All); err != nil {
		// slog.Error("Watch Error", "err", err.Error())
		return
	}

	for {
		select {
		case ei := <-engine.Chan:
			eventManager.HandleEvent(ei)
		}
	}

}

func getPath(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		// slog.Error("Getting working directory")
		return ""
	}
	relPath, err := filepath.Rel(wd, path)
	if err != nil {
		// slog.Error("Getting relative path")
		return ""
	}
	return relPath
}
