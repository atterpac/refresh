//go:build linux || darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atterpac/refresh/engine"
)

// waitFor polls cond until it is true or a generous deadline elapses, so the
// test tolerates filesystem-notification and process-startup latency without
// fixed sleeps.
func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

// lines counts the non-empty lines in a file, or 0 if it doesn't exist yet.
func lines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n := 0
	for l := range strings.SplitSeq(string(data), "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

// writeGo writes a .go file under watched/ with a changing constant so each call
// produces a content-modification event (reliable on both linux and darwin,
// unlike file creation).
func writeGo(t *testing.T, root, name string, version int) {
	t.Helper()
	src := fmt.Sprintf("package watched\n\nconst V = %d\n", version)
	if err := os.WriteFile(filepath.Join(root, "watched", name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestKitchenSinkIntegration drives the full engine lifecycle (Start, real
// filesystem-triggered reloads, then Stop) and verifies the contract of every
// execute type, ChangeDir, the reload callback (including veto), and the ignore
// rules.
func TestKitchenSinkIntegration(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "watched"))
	mustMkdir(t, filepath.Join(root, "sub")) // ChangeDir target must exist

	// Pre-create the watched files so later writes are modifications.
	writeGo(t, root, "trigger.go", 1)
	writeGo(t, root, "bypass_me.go", 1)    // callback will veto changes to this
	writeGo(t, root, "thing_ignore.go", 1) // matches the *_ignore.go ignore rule

	var callbacks atomic.Int32
	cfg := buildConfig(root)
	cfg.LogLevel = "mute" // keep test output clean
	cfg.Callback = func(e *engine.EventCallback) engine.EventHandle {
		callbacks.Add(1)
		if strings.Contains(e.Path, "bypass_me") {
			return engine.EventBypass
		}
		return engine.EventContinue
	}

	eng, err := engine.NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Start blocks until shutdown; run it in the background and ensure a clean
	// stop (which also kills the background/primary sleeps — no leaks).
	done := make(chan error, 1)
	go func() { done <- eng.Start() }()
	t.Cleanup(func() {
		eng.Stop()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("engine did not shut down within timeout")
		}
	})

	art := func(name string) string { return filepath.Join(root, "artifacts", name) }

	// --- Initial cycle: every type should have produced its marker. ---
	if !waitFor(func() bool { return lines(art("primary.log")) == 1 }) {
		t.Fatalf("primary did not start (primary.log = %d lines)", lines(art("primary.log")))
	}
	for _, f := range []string{"once.log", "background.log", "blocking.log"} {
		if !waitFor(func() bool { return lines(art(f)) >= 1 }) {
			t.Fatalf("%s was not written on startup", f)
		}
	}

	// --- #2 ChangeDir: the marker must land in sub/, not at the root. ---
	if _, err := os.Stat(filepath.Join(root, "sub", "marker.txt")); err != nil {
		t.Errorf("ChangeDir execute did not run in sub/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "marker.txt")); err == nil {
		t.Error("ChangeDir was ignored: marker.txt landed at the root")
	}

	// Let the watcher settle before editing, so changes are observed.
	time.Sleep(200 * time.Millisecond)

	// --- #1 Callback veto: a change the callback bypasses must NOT reload. ---
	cbBefore := callbacks.Load()
	writeGo(t, root, "bypass_me.go", 2)
	time.Sleep(900 * time.Millisecond) // longer than debounce + restart
	if got := lines(art("primary.log")); got != 1 {
		t.Errorf("callback EventBypass did not prevent reload (primary.log = %d)", got)
	}
	if callbacks.Load() <= cbBefore {
		t.Error("callback was not invoked for the bypassed change")
	}

	// --- #4 Ignore rule: a change matching *_ignore.go must NOT reload. ---
	cbBefore = callbacks.Load()
	writeGo(t, root, "thing_ignore.go", 2)
	time.Sleep(900 * time.Millisecond)
	if got := lines(art("primary.log")); got != 1 {
		t.Errorf("ignore rule did not prevent reload (primary.log = %d)", got)
	}
	if callbacks.Load() <= cbBefore {
		t.Error("ignored change was never observed (test would pass trivially)")
	}

	// --- A normal change must reload: primary restarts, blocking re-runs. ---
	writeGo(t, root, "trigger.go", 2)
	if !waitFor(func() bool { return lines(art("primary.log")) >= 2 }) {
		t.Fatal("primary did not restart on a watched change")
	}
	if !waitFor(func() bool { return lines(art("blocking.log")) >= 2 }) {
		t.Fatal("blocking step did not re-run on reload")
	}

	// Allow any (erroneous) extra work to surface before checking run-once
	// semantics held.
	time.Sleep(300 * time.Millisecond)

	if got := lines(art("once.log")); got != 1 {
		t.Errorf("once execute ran %d times, want exactly 1", got)
	}
	if got := lines(art("background.log")); got != 1 {
		t.Errorf("background execute restarted (%d lines), want 1 (it should survive reloads)", got)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
