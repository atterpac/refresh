//go:build linux || darwin

package process

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// alive reports whether a pid refers to a live process (signal 0 probes without
// actually sending anything).
func alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// waitFor polls until cond is true or the deadline elapses.
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

func TestStartReloadShutdownLifecycle(t *testing.T) {
	root := t.TempDir()
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(root); err != nil {
		t.Fatalf("SetRootDirectory: %v", err)
	}

	// A blocking step that must re-run on every cycle, and a long-lived primary
	// that must be killed and restarted on reload.
	if err := pm.AddProcess("touch marker", "blocking", ""); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcess("sleep 30", "primary", ""); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pm.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	marker := filepath.Join(root, "marker")
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("blocking step did not run on start: %v", err)
	}

	primary := pm.Processes[1]
	pid1 := primary.cmd.Process.Pid
	if !alive(pid1) {
		t.Fatalf("primary not running after start (pid %d)", pid1)
	}

	// Reload should re-run the blocking step and restart the primary with a new pid.
	os.Remove(marker)
	if err := pm.Reload(ctx); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("blocking step did not re-run on reload: %v", err)
	}

	pid2 := primary.cmd.Process.Pid
	if pid2 == pid1 {
		t.Fatalf("primary was not restarted (pid unchanged: %d)", pid1)
	}
	if !waitFor(func() bool { return !alive(pid1) }) {
		t.Errorf("old primary (pid %d) still alive after reload", pid1)
	}
	if !alive(pid2) {
		t.Fatalf("new primary (pid %d) not running after reload", pid2)
	}

	// Shutdown must terminate the running primary.
	pm.Shutdown()
	if !waitFor(func() bool { return !alive(pid2) }) {
		t.Errorf("primary (pid %d) still alive after shutdown", pid2)
	}
}

func TestShellFeaturesAreSupported(t *testing.T) {
	root := t.TempDir()
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(root); err != nil {
		t.Fatal(err)
	}
	// Redirection and && only work when the command runs through a shell rather
	// than a bare argv split.
	if err := pm.AddProcess("echo one > out.txt && echo two >> out.txt", "blocking", ""); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcess("sleep 30", "primary", ""); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := pm.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pm.Shutdown()

	data, err := os.ReadFile(filepath.Join(root, "out.txt"))
	if err != nil {
		t.Fatalf("shell command did not produce output file: %v", err)
	}
	if got := string(data); got != "one\ntwo\n" {
		t.Errorf("shell features not honored, out.txt = %q", got)
	}
}

func TestStartWithNoProcessesErrors(t *testing.T) {
	pm := NewProcessManager()
	if err := pm.Start(context.Background()); err == nil {
		t.Fatal("expected error when starting with no processes")
	}
}

func TestBlockingFailureAbortsCycle(t *testing.T) {
	root := t.TempDir()
	pm := NewProcessManager()
	if err := pm.SetRootDirectory(root); err != nil {
		t.Fatal(err)
	}
	// A blocking step that exits non-zero must abort the cycle before the primary.
	if err := pm.AddProcess("false", "blocking", ""); err != nil {
		t.Fatal(err)
	}
	if err := pm.AddProcess("sleep 30", "primary", ""); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pm.Start(ctx); err == nil {
		t.Fatal("expected Start to fail when a blocking step fails")
	}
	if primary := pm.Processes[1]; primary.cmd != nil {
		t.Error("primary should not have started after blocking failure")
	}
	pm.Shutdown()
}
