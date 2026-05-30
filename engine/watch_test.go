package engine

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/atterpac/refresh/process"
)

// newWatchTestEngine builds a minimal engine whose watcher can run against a
// real temp directory, without starting any processes.
func newWatchTestEngine(t *testing.T, root string, debounceMS int) *Engine {
	t.Helper()
	e := &Engine{Config: Config{
		RootPath: root,
		Debounce: debounceMS,
		Ignore:   Ignore{WatchedExten: []string{"*.txt"}},
	}}
	e.ProcessManager = process.NewProcessManager()
	if err := e.ProcessManager.SetRootDirectory(root); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestWatcherCoalescesBurstIntoSingleReload(t *testing.T) {
	root := t.TempDir()
	e := newWatchTestEngine(t, root, 200)

	reload := make(chan struct{}, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := e.startWatcher(ctx, reload); err != nil {
		t.Fatalf("startWatcher: %v", err)
	}

	// A burst of writes well within the debounce window should collapse to one
	// reload fired after the quiet interval.
	file := filepath.Join(root, "a.txt")
	for i := range 5 {
		if err := os.WriteFile(file, []byte(strconv.Itoa(i)), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(15 * time.Millisecond)
	}

	// Wait comfortably past the debounce for the trailing-edge fire.
	time.Sleep(500 * time.Millisecond)
	cancel()

	if got := len(reload); got != 1 {
		t.Errorf("expected exactly 1 coalesced reload, got %d", got)
	}
}

func TestWatcherDetectsNestedSubdirectoryChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	e := newWatchTestEngine(t, root, 150)

	reload := make(chan struct{}, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := e.startWatcher(ctx, reload); err != nil {
		t.Fatalf("startWatcher: %v", err)
	}

	// The recursive watch must catch writes in nested directories.
	if err := os.WriteFile(filepath.Join(root, "app", "main.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	cancel()

	if got := len(reload); got != 1 {
		t.Errorf("nested change produced %d reloads, want 1", got)
	}
}

// TestWatcherDefaultConfigReloadsAnyFile guards the default/empty-filter path:
// with no WatchedExten configured (DefaultEngineConfig and the bare CLI), every
// change must still trigger a reload. A regression here previously made the
// out-of-the-box config silently watch nothing.
func TestWatcherDefaultConfigReloadsAnyFile(t *testing.T) {
	root := t.TempDir()
	e := &Engine{Config: Config{
		RootPath: root,
		Debounce: 100,
		Ignore:   Ignore{}, // zero value: no extension filter
	}}
	e.ProcessManager = process.NewProcessManager()
	if err := e.ProcessManager.SetRootDirectory(root); err != nil {
		t.Fatal(err)
	}

	reload := make(chan struct{}, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := e.startWatcher(ctx, reload); err != nil {
		t.Fatalf("startWatcher: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	cancel()

	if got := len(reload); got != 1 {
		t.Errorf("default config produced %d reloads for a .go edit, want 1", got)
	}
}

func TestWatcherIgnoresUnwatchedExtensions(t *testing.T) {
	root := t.TempDir()
	e := newWatchTestEngine(t, root, 100)

	reload := make(chan struct{}, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := e.startWatcher(ctx, reload); err != nil {
		t.Fatalf("startWatcher: %v", err)
	}

	// Only *.txt is watched; a *.log write must not trigger a reload.
	if err := os.WriteFile(filepath.Join(root, "ignore.log"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	cancel()

	if got := len(reload); got != 0 {
		t.Errorf("unwatched extension triggered %d reloads, want 0", got)
	}
}
