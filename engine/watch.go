package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
)

// watcher translates raw filesystem notifications into debounced reload
// requests. A single timer is reset on every reload-eligible event, so a burst
// of writes (editors often emit several per save) collapses into one reload
// fired after the quiet interval — true trailing-edge debounce.
type watcher struct {
	engine   *Engine
	events   chan notify.EventInfo
	reload   chan<- struct{}
	debounce time.Duration
	root     string
	timer    *time.Timer
}

// startWatcher begins watching the resolved root directory and spawns the
// watcher goroutine. Reload requests are delivered on reload.
func (engine *Engine) startWatcher(ctx context.Context, reload chan<- struct{}) error {
	root := engine.ProcessManager.RootDir
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolving watch root: %w", err)
		}
		root = wd
	}

	events := make(chan notify.EventInfo, 16)
	if err := notify.Watch(filepath.Join(root, "..."), events, notify.All); err != nil {
		return fmt.Errorf("starting file watcher: %w", err)
	}

	w := &watcher{
		engine:   engine,
		events:   events,
		reload:   reload,
		debounce: time.Duration(engine.Config.Debounce) * time.Millisecond,
		root:     root,
	}
	go w.run(ctx)
	slog.Info("watching for changes", "root", root)
	return nil
}

func (w *watcher) run(ctx context.Context) {
	defer notify.Stop(w.events)

	// Start with a stopped, drained timer.
	w.timer = time.NewTimer(0)
	if !w.timer.Stop() {
		<-w.timer.C
	}

	for {
		select {
		case <-ctx.Done():
			w.timer.Stop()
			return
		case ei := <-w.events:
			w.handle(ei)
		case <-w.timer.C:
			w.signalReload()
		}
	}
}

// handle decides whether a single event should (eventually) trigger a reload,
// applying the platform event map, the user callback, and the ignore rules,
// then resets the debounce timer.
func (w *watcher) handle(ei notify.EventInfo) {
	info, ok := EventMap[ei.Event()]
	if !ok {
		slog.Debug("unknown event", "event", ei.Event())
		return
	}
	if !info.Reload {
		return
	}

	rel := w.relPath(ei.Path())

	if w.engine.Config.Callback != nil {
		switch w.engine.Config.Callback(&EventCallback{
			Type: CallbackMap[ei.Event()],
			Path: rel,
			Time: time.Now(),
		}) {
		case EventBypass, EventIgnore:
			return
		}
	}

	if w.engine.Config.Ignore.shouldIgnore(ei.Path()) {
		slog.Debug("ignoring change", "path", rel)
		return
	}

	slog.Debug("change detected", "path", rel, "event", info.Name)
	w.timer.Reset(w.debounce)
}

// signalReload performs a non-blocking send so a reload that is already queued
// is not duplicated; the buffered channel coalesces bursts into one reload.
func (w *watcher) signalReload() {
	select {
	case w.reload <- struct{}{}:
	default:
	}
}

func (w *watcher) relPath(path string) string {
	rel, err := filepath.Rel(w.root, path)
	if err != nil {
		return path
	}
	return rel
}
