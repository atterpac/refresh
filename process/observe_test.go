//go:build linux || darwin

package process

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
)

// syncBuffer is a goroutine-safe bytes.Buffer; process output is written from a
// process's own wait goroutine while the test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestOutputHookCapturesPerProcessStreams verifies the Output hook routes a
// process's stdout and stderr to caller-supplied writers, keyed by process name.
func TestOutputHookCapturesPerProcessStreams(t *testing.T) {
	root := t.TempDir()
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(root); err != nil {
		t.Fatal(err)
	}

	outBuf := &syncBuffer{}
	errBuf := &syncBuffer{}
	pm.Output = func(info ProcessInfo, stream string) io.Writer {
		if info.Name != "greeter" {
			return nil // only capture the greeter; other processes keep defaults
		}
		switch stream {
		case "stdout":
			return outBuf
		case "stderr":
			return errBuf
		default:
			t.Errorf("unexpected stream %q", stream)
			return nil
		}
	}

	if err := pm.AddProcessSpec(Execute{
		Name: "greeter",
		Cmd:  "echo to-out && echo to-err 1>&2",
		Type: Blocking,
	}); err != nil {
		t.Fatal(err)
	}
	// A primary is required by the engine, but the manager itself is happy with a
	// lone blocking step; add a trivial primary so the cycle mirrors real use.
	if err := pm.AddProcessSpec(Execute{Name: "app", Cmd: "true", Type: Primary}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := pm.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pm.Shutdown()

	if got := outBuf.String(); got != "to-out\n" {
		t.Errorf("stdout capture = %q, want %q", got, "to-out\n")
	}
	if got := errBuf.String(); got != "to-err\n" {
		t.Errorf("stderr capture = %q, want %q", got, "to-err\n")
	}
}

// TestOutputHookNilWriterFallsBack verifies a nil return from the Output hook
// does not panic and lets the process run normally (writing to its own stdout).
func TestOutputHookNilWriterFallsBack(t *testing.T) {
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	called := false
	pm.Output = func(info ProcessInfo, stream string) io.Writer {
		called = true
		return nil // caller declines to capture this stream
	}
	if err := pm.AddProcessSpec(Execute{Cmd: "true", Type: Blocking}); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcessSpec(Execute{Cmd: "true", Type: Primary}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := pm.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	pm.Shutdown()
	if !called {
		t.Error("Output hook was never consulted")
	}
}

// collectEvents records ProcessEvents in a goroutine-safe slice.
type eventLog struct {
	mu     sync.Mutex
	events []ProcessEvent
}

func (e *eventLog) record(ev ProcessEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
}

func (e *eventLog) statesFor(name string) []ProcessState {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []ProcessState
	for _, ev := range e.events {
		if ev.Info.Name == name {
			out = append(out, ev.Info.State)
		}
	}
	return out
}

// TestLifecycleEventsEmitted verifies a primary process emits running on start
// and killed when the cycle restarts it, while a blocking step reports
// running→exited.
func TestLifecycleEventsEmitted(t *testing.T) {
	root := t.TempDir()
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(root); err != nil {
		t.Fatal(err)
	}
	log := &eventLog{}
	pm.OnEvent = log.record

	if err := pm.AddProcessSpec(Execute{Name: "build", Cmd: "true", Type: Blocking}); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcessSpec(Execute{Name: "server", Cmd: "sleep 30", Type: Primary}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := pm.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Blocking step ran to completion during Start.
	if got := log.statesFor("build"); len(got) < 2 || got[0] != StateRunning || got[len(got)-1] != StateExited {
		t.Errorf("build states = %v, want running then exited", got)
	}
	// Primary is up.
	if !waitFor(func() bool {
		s := log.statesFor("server")
		return len(s) >= 1 && s[len(s)-1] == StateRunning
	}) {
		t.Fatalf("server never reported running, states = %v", log.statesFor("server"))
	}

	// Reload restarts the primary: expect a killed then a fresh running.
	if err := pm.Reload(ctx); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !waitFor(func() bool {
		s := log.statesFor("server")
		var killed bool
		for i, st := range s {
			if st == StateKilled {
				killed = true
			}
			if killed && st == StateRunning && i > 0 {
				return true
			}
		}
		return false
	}) {
		t.Errorf("server did not report killed→running on reload, states = %v", log.statesFor("server"))
	}

	pm.Shutdown()
	if !waitFor(func() bool {
		s := log.statesFor("server")
		return len(s) > 0 && s[len(s)-1] == StateKilled
	}) {
		t.Errorf("server did not report killed on shutdown, states = %v", log.statesFor("server"))
	}
}

// TestFailedBlockingReportsExitCode verifies a non-zero blocking step surfaces a
// failed state carrying the real exit code and the error.
func TestFailedBlockingReportsExitCode(t *testing.T) {
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	log := &eventLog{}
	pm.OnEvent = log.record

	if err := pm.AddProcessSpec(Execute{Name: "lint", Cmd: "exit 3", Type: Blocking}); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcessSpec(Execute{Name: "app", Cmd: "true", Type: Primary}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := pm.Start(ctx); err == nil {
		t.Fatal("expected Start to fail on a non-zero blocking step")
	}
	pm.Shutdown()

	log.mu.Lock()
	defer log.mu.Unlock()
	var failed *ProcessEvent
	for i := range log.events {
		if log.events[i].Info.Name == "lint" && log.events[i].Info.State == StateFailed {
			failed = &log.events[i]
		}
	}
	if failed == nil {
		t.Fatalf("no failed event for lint; events = %+v", log.events)
	}
	if failed.Info.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", failed.Info.ExitCode)
	}
	if failed.Err == nil {
		t.Error("failed event Err = nil, want the command error")
	}
}

// TestSnapshotReflectsState verifies Snapshot reports the live state of each
// process and defaults the name to the command when none is set.
func TestSnapshotReflectsState(t *testing.T) {
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcessSpec(Execute{Cmd: "sleep 30", Type: Primary}); err != nil {
		t.Fatal(err)
	}

	// Before start everything is pending, and the name falls back to the command.
	pre := pm.Snapshot()
	if len(pre) != 1 || pre[0].State != StatePending {
		t.Fatalf("pre-start snapshot = %+v, want one pending process", pre)
	}
	if pre[0].Name != "sleep 30" {
		t.Errorf("default Name = %q, want the command string", pre[0].Name)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := pm.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pm.Shutdown()

	if !waitFor(func() bool {
		s := pm.Snapshot()
		return len(s) == 1 && s[0].State == StateRunning && s[0].PID > 0
	}) {
		t.Errorf("running snapshot = %+v, want running with a pid", pm.Snapshot())
	}
	if got := pm.Snapshot()[0].StartedAt; got.IsZero() {
		t.Error("StartedAt is zero for a running process")
	}
}
