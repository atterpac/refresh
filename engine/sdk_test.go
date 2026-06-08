//go:build linux || darwin

package engine

import (
	"bytes"
	"context"
	"io"
	"sync"
	"syscall"
	"testing"
	"time"
)

// pidAlive reports whether a process with the given pid is still alive. Signal 0
// probes for existence without affecting the process.
func pidAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// safeBuffer is a goroutine-safe writer for capturing process output in tests.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

// TestRunTapsOutputAndEvents is the end-to-end SDK story: an embedding caller
// drives the engine through Run(ctx), captures each process's output via the
// Output hook, observes lifecycle through OnProcessEvent, and polls live state
// through Processes() — then cancels the context to shut everything down.
func TestRunTapsOutputAndEvents(t *testing.T) {
	root := t.TempDir()

	panes := map[string]*safeBuffer{
		"banner": {},
		"server": {},
	}

	var (
		mu     sync.Mutex
		events []ProcessEvent
	)

	cfg := Config{
		RootPath: root,
		LogLevel: "mute",
		Debounce: 100,
		Ignore:   Ignore{WatchedExten: []string{"*.go"}},
		ExecStruct: []Execute{
			{Name: "banner", Cmd: "echo hello-from-banner", Type: Blocking},
			{Name: "server", Cmd: "sleep 30", Type: Primary},
		},
		Output: func(info ProcessInfo, stream string) io.Writer {
			if b, ok := panes[info.Name]; ok {
				return b
			}
			return nil
		},
		OnProcessEvent: func(ev ProcessEvent) {
			mu.Lock()
			events = append(events, ev)
			mu.Unlock()
		},
	}

	eng, err := NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineFromConfig: %v", err)
	}

	// Before Run, every process is pending.
	for _, info := range eng.Processes() {
		if info.State != StatePending {
			t.Errorf("%s state = %q before Run, want pending", info.Name, info.State)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- eng.Run(ctx) }()

	// The blocking banner step runs once during startup; its output lands in the
	// banner pane.
	if !waitFor(func() bool { return panes["banner"].String() == "hello-from-banner\n" }) {
		t.Errorf("banner pane = %q, want %q", panes["banner"].String(), "hello-from-banner\n")
	}

	// The primary comes up running with a live pid, visible through Processes().
	if !waitFor(func() bool {
		for _, info := range eng.Processes() {
			if info.Name == "server" {
				return info.State == StateRunning && info.PID > 0
			}
		}
		return false
	}) {
		t.Fatalf("server never reported running; snapshot = %+v", eng.Processes())
	}

	// A running event for the server must have been delivered.
	if !waitFor(func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, ev := range events {
			if ev.Info.Name == "server" && ev.Info.State == StateRunning {
				return true
			}
		}
		return false
	}) {
		t.Error("no running event delivered for server")
	}

	// Cancelling the context shuts the engine down cleanly and stops the primary.
	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run returned %v, want nil on context cancel", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}

	if !waitFor(func() bool {
		for _, info := range eng.Processes() {
			if info.Name == "server" {
				return info.State == StateKilled
			}
		}
		return false
	}) {
		t.Errorf("server not killed after shutdown; snapshot = %+v", eng.Processes())
	}
}

// serverPID returns the running pid of the named process from a snapshot, or 0.
func serverPID(eng *Engine, name string) int {
	for _, info := range eng.Processes() {
		if info.Name == name {
			return info.PID
		}
	}
	return 0
}

// TestProgrammaticReloadRestartsPrimary verifies Engine.Reload restarts the
// primary process (new pid), exactly as a file change would, while driven under
// Run(ctx).
func TestProgrammaticReloadRestartsPrimary(t *testing.T) {
	cfg := Config{
		RootPath:   t.TempDir(),
		LogLevel:   "mute",
		Debounce:   100,
		Ignore:     Ignore{WatchedExten: []string{"*.go"}},
		ExecStruct: []Execute{{Name: "server", Cmd: "sleep 30", Type: Primary}},
	}
	eng, err := NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineFromConfig: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = eng.Run(ctx) }()

	if !waitFor(func() bool { return serverPID(eng, "server") > 0 }) {
		t.Fatal("server never started")
	}
	pid1 := serverPID(eng, "server")

	eng.Reload()

	if !waitFor(func() bool {
		pid := serverPID(eng, "server")
		return pid > 0 && pid != pid1
	}) {
		t.Errorf("Reload did not restart primary; pid still %d", pid1)
	}
}

// TestPauseDefersReloadUntilResume verifies that a Reload issued while paused is
// remembered and applied on Resume, and not before.
func TestPauseDefersReloadUntilResume(t *testing.T) {
	cfg := Config{
		RootPath:   t.TempDir(),
		LogLevel:   "mute",
		Debounce:   100,
		Ignore:     Ignore{WatchedExten: []string{"*.go"}},
		ExecStruct: []Execute{{Name: "server", Cmd: "sleep 30", Type: Primary}},
	}
	eng, err := NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineFromConfig: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = eng.Run(ctx) }()

	if !waitFor(func() bool { return serverPID(eng, "server") > 0 }) {
		t.Fatal("server never started")
	}
	pid1 := serverPID(eng, "server")

	eng.Pause()
	if !eng.Paused() {
		t.Fatal("Paused() = false after Pause()")
	}

	// Reload while paused defers the restart.
	eng.Reload()
	// Give the supervisor a chance to (wrongly) act on it.
	if waitFor(func() bool {
		pid := serverPID(eng, "server")
		return pid > 0 && pid != pid1
	}) {
		t.Fatalf("primary restarted while paused (pid changed from %d)", pid1)
	}

	// Resume must apply the deferred reload.
	eng.Resume()
	if eng.Paused() {
		t.Fatal("Paused() = true after Resume()")
	}
	if !waitFor(func() bool {
		pid := serverPID(eng, "server")
		return pid > 0 && pid != pid1
	}) {
		t.Errorf("deferred reload not applied on Resume; pid still %d", pid1)
	}
}

// TestPauseResumeIdempotent verifies the pause flag tolerates repeated calls and
// that a resume with no pending change does not spuriously restart the primary.
func TestPauseResumeIdempotent(t *testing.T) {
	cfg := Config{
		RootPath:   t.TempDir(),
		LogLevel:   "mute",
		Debounce:   100,
		Ignore:     Ignore{WatchedExten: []string{"*.go"}},
		ExecStruct: []Execute{{Name: "server", Cmd: "sleep 30", Type: Primary}},
	}
	eng, err := NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineFromConfig: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = eng.Run(ctx) }()

	if !waitFor(func() bool { return serverPID(eng, "server") > 0 }) {
		t.Fatal("server never started")
	}
	pid1 := serverPID(eng, "server")

	eng.Pause()
	eng.Pause() // idempotent
	if !eng.Paused() {
		t.Fatal("Paused() = false after repeated Pause()")
	}
	eng.Resume()
	eng.Resume() // idempotent; no pending change

	// No reload was issued, so the primary must be untouched.
	if waitFor(func() bool {
		pid := serverPID(eng, "server")
		return pid > 0 && pid != pid1
	}) {
		t.Errorf("primary restarted on a no-op resume (pid changed from %d)", pid1)
	}
}

// TestCancelDuringStartupKillsStartedProcesses guards the interrupt path: a
// background process is started during the initial pass, then a slower blocking
// step runs. Cancelling mid-startup must tear down the already-started
// background process.
func TestCancelDuringStartupKillsStartedProcesses(t *testing.T) {
	cfg := Config{
		RootPath: t.TempDir(),
		LogLevel: "mute",
		Debounce: 100,
		Ignore:   Ignore{WatchedExten: []string{"*.go"}},
		ExecStruct: []Execute{
			{Name: "bg", Cmd: "sleep 30", Type: Background},
			{Name: "build", Cmd: "sleep 2", Type: Blocking},
			{Name: "server", Cmd: "sleep 30", Type: Primary},
		},
	}
	eng, err := NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineFromConfig: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- eng.Run(ctx) }()

	// Wait until the background process is up and the blocking step is still
	// running (the primary has not started yet), so we cancel mid-startup.
	if !waitFor(func() bool {
		return serverPID(eng, "bg") > 0 && serverPID(eng, "server") == 0
	}) {
		t.Fatalf("background never started before blocking step; snapshot = %+v", eng.Processes())
	}
	bgPID := serverPID(eng, "bg")

	cancel()

	// Run returns cleanly on mid-startup cancel; background process must be dead.
	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run returned %v on cancel during startup, want nil (clean shutdown)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel during startup")
	}
	if !waitFor(func() bool { return !pidAlive(bgPID) }) {
		t.Errorf("background pid %d survived shutdown — orphaned", bgPID)
	}
}
