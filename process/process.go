package process

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Process is a single configured command plus the runtime handles for its
// currently running instance (if any).
type Process struct {
	Exec string
	Type ExecuteType
	Dir  string
	// Delay is the pause in milliseconds inserted after this process's step
	// completes, before the next configured process starts. Zero means no pause.
	Delay int

	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan struct{}
}

// ProcessManager supervises the configured processes.
//
// All lifecycle methods (Start, Reload, Shutdown) are expected to be driven from
// a single goroutine — the engine's supervisor loop guarantees this — so the
// per-process runtime fields need no additional locking. The only concurrent
// actor is each process's own wait goroutine, which never mutates those fields.
type ProcessManager struct {
	Processes []*Process
	RootDir   string
	started   bool
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{Processes: make([]*Process, 0)}
}

func (pm *ProcessManager) AddProcess(exec, typing, dir string) error {
	return pm.AddProcessWithDelay(exec, typing, dir, 0)
}

// AddProcessWithDelay is AddProcess plus delay, the pause in milliseconds held
// after this process's step before the next one starts (the delay_next config
// field).
func (pm *ProcessManager) AddProcessWithDelay(exec, typing, dir string, delay int) error {
	execType, err := stringToExecuteType(typing)
	if err != nil {
		return err
	}
	pm.Processes = append(pm.Processes, &Process{Exec: exec, Type: execType, Dir: dir, Delay: delay})
	return nil
}

// GetExecutes returns the configured command strings in order.
func (pm *ProcessManager) GetExecutes() []string {
	execs := make([]string, 0, len(pm.Processes))
	for _, p := range pm.Processes {
		execs = append(execs, p.Exec)
	}
	return execs
}

// SetRootDirectory resolves dir to an absolute path used as the base for every
// process working directory. It no longer changes the calling process's working
// directory; each command's directory is set on its exec.Cmd instead, which
// removes the global-state race the previous os.Chdir approach suffered from.
func (pm *ProcessManager) SetRootDirectory(dir string) error {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return errors.New("unable to determine working directory")
		}
		pm.RootDir = wd
		return nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	pm.RootDir = abs
	slog.Debug("root directory set", "dir", pm.RootDir)
	return nil
}

// resolveDir returns the absolute working directory for a process, joining
// relative directories onto RootDir.
func (pm *ProcessManager) resolveDir(dir string) string {
	switch {
	case dir == "":
		return pm.RootDir
	case filepath.IsAbs(dir):
		return dir
	default:
		return filepath.Join(pm.RootDir, dir)
	}
}

// Start performs the initial pass over all configured processes: background and
// once processes run only here; blocking and primary processes run every cycle.
func (pm *ProcessManager) Start(ctx context.Context) error {
	if len(pm.Processes) == 0 {
		return errors.New("no processes configured")
	}
	return pm.runCycle(ctx, true)
}

// Reload re-runs blocking steps and restarts the primary process. Background and
// once processes started during Start are left running.
func (pm *ProcessManager) Reload(ctx context.Context) error {
	return pm.runCycle(ctx, false)
}

func (pm *ProcessManager) runCycle(ctx context.Context, firstRun bool) error {
	for _, p := range pm.Processes {
		// Markers used by the ExecList config form; no-ops in the struct form.
		if p.Exec == KILL_EXEC || p.Exec == REFRESH_EXEC {
			continue
		}
		switch p.Type {
		case Background:
			if !firstRun {
				continue
			}
			if err := pm.startAsync(ctx, p); err != nil {
				slog.Error("starting background process", "exec", p.Exec, "err", err)
				return err
			}
		case Once:
			if !firstRun {
				continue
			}
			if err := pm.runBlocking(ctx, p); err != nil {
				slog.Error("once process failed", "exec", p.Exec, "err", err)
				return err
			}
		case Blocking:
			if err := pm.runBlocking(ctx, p); err != nil {
				// On reload a failed blocking step (typically a build error)
				// aborts the cycle and leaves the current primary running, so a
				// broken build doesn't take down the last good process.
				slog.Error("blocking process failed", "exec", p.Exec, "err", err)
				return err
			}
		case Primary:
			pm.stopProcess(p) // kill the previous instance (no-op on first run)
			if err := pm.startAsync(ctx, p); err != nil {
				slog.Error("starting primary process", "exec", p.Exec, "err", err)
				return err
			}
		}
		// Reached only when the step above actually ran (skipped/aborted steps
		// continue/return before here), so the delay sits strictly between this
		// process and the next.
		if !pm.delayNext(ctx, p) {
			return nil // context cancelled mid-delay — abort the cycle quietly
		}
	}
	pm.started = true
	return nil
}

// delayNext holds for the process's configured delay_next before the cycle moves
// on, letting one step settle (e.g. a service binding a port) before the next
// starts. The wait is context-aware; it returns false if the context is
// cancelled during the pause so the caller can stop the cycle.
func (pm *ProcessManager) delayNext(ctx context.Context, p *Process) bool {
	if p.Delay <= 0 {
		return true
	}
	slog.Debug("delaying before next process", "exec", p.Exec, "ms", p.Delay)
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Duration(p.Delay) * time.Millisecond):
		return true
	}
}

// startAsync launches a long-lived process (background or primary) in its own
// process group and tracks it so it can be terminated on the next cycle or
// shutdown. The command is started in a fresh process group so the whole tree
// can be signalled, not just the direct child.
func (pm *ProcessManager) startAsync(ctx context.Context, p *Process) error {
	procCtx, cancel := context.WithCancel(ctx)
	cmd := generateExec(p.Exec)
	cmd.Dir = pm.resolveDir(p.Dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcessGroup(cmd)

	slog.Debug("starting process", "exec", p.Exec, "dir", cmd.Dir)
	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	done := make(chan struct{})
	p.cmd = cmd
	p.cancel = cancel
	p.done = done

	go func() {
		defer close(done)
		waitErr := make(chan error, 1)
		go func() { waitErr <- cmd.Wait() }()
		select {
		case <-procCtx.Done():
			if err := killProcessTree(cmd); err != nil {
				slog.Debug("killing process tree", "exec", p.Exec, "err", err)
			}
			<-waitErr // reap the process after the kill
		case err := <-waitErr:
			if err != nil {
				slog.Debug("process exited", "exec", p.Exec, "err", err)
			}
		}
	}()
	return nil
}

// runBlocking runs a process to completion. The command is bound to ctx so a
// shutdown while it is running terminates it.
func (pm *ProcessManager) runBlocking(ctx context.Context, p *Process) error {
	cmd := generateExecContext(ctx, p.Exec)
	cmd.Dir = pm.resolveDir(p.Dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	slog.Debug("running blocking process", "exec", p.Exec, "dir", cmd.Dir)
	return cmd.Run()
}

// stopProcess cancels a tracked process and waits for it to fully terminate.
// Safe to call on a process that isn't running.
func (pm *ProcessManager) stopProcess(p *Process) {
	if p.cancel != nil {
		p.cancel()
	}
	if p.done != nil {
		<-p.done
	}
	p.cmd, p.cancel, p.done = nil, nil, nil
}

// Shutdown terminates every running process and waits for them to exit.
func (pm *ProcessManager) Shutdown() {
	slog.Debug("shutting down processes")
	for _, p := range pm.Processes {
		pm.stopProcess(p)
	}
}
