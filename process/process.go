package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Process is a single configured command plus the runtime handles for its
// currently running instance (if any).
type Process struct {
	// Name is a stable identifier for the process, used by consumers as a key for
	// a per-process log pane. Defaults to Exec when empty.
	Name            string
	Exec            string
	Command         []string
	Env             map[string]string
	ReadyTCP        string
	ReadyHTTP       string
	ReadyTimeout    time.Duration
	ReadyInterval   time.Duration
	ShutdownTimeout time.Duration
	ExitPolicy      ExitPolicy
	Type            ExecuteType
	Dir             string
	// Delay is the pause in milliseconds inserted after this process's step
	// completes, before the next configured process starts. Zero means no pause.
	Delay int

	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan struct{}

	// Runtime state observable through ProcessInfo snapshots. Guarded by
	// ProcessManager.mu because the per-process wait goroutine writes the exit
	// state while a consumer goroutine may read a snapshot concurrently.
	state     ProcessState
	pid       int
	startedAt time.Time
	exitCode  int
}

// ProcessManager supervises the configured processes.
//
// Lifecycle methods (Start, Reload, Shutdown) are driven from a single
// goroutine — the engine's supervisor loop guarantees this — so the process
// handles (cmd/cancel/done) need no locking. The observable runtime state
// (state/pid/startedAt/exitCode), however, is also written by each process's
// wait goroutine and read by consumers via Snapshot, so it is guarded by mu.
type ProcessManager struct {
	Processes []*Process
	RootDir   string
	started   bool

	// Output, when set, resolves the writer each process's stdout/stderr is wired
	// to. nil falls back to os.Stdout/os.Stderr.
	Output OutputFunc
	// OnEvent, when set, receives a ProcessEvent on every state transition.
	OnEvent EventFunc

	mu sync.RWMutex
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
	return pm.AddProcessSpec(Execute{Cmd: exec, ChangeDir: dir, DelayNext: delay, Type: ExecuteType(typing)})
}

// AddProcessSpec appends a process from a full Execute spec, preserving its Name
// (used as the per-process identifier in snapshots and events).
func (pm *ProcessManager) AddProcessSpec(spec Execute) error {
	if err := spec.Validate(); err != nil {
		return err
	}
	execType, err := stringToExecuteType(string(spec.Type))
	if err != nil {
		return err
	}
	shutdownTimeout, _ := parseOptionalDuration(spec.ShutdownTimeout)
	process := &Process{
		Name:            spec.Name,
		Exec:            spec.Cmd,
		Command:         append([]string(nil), spec.Command...),
		Env:             cloneEnvironment(spec.Env),
		Type:            execType,
		Dir:             spec.ChangeDir,
		Delay:           spec.DelayNext,
		state:           StatePending,
		exitCode:        noExitYet,
		ShutdownTimeout: shutdownTimeout,
		ExitPolicy:      spec.ExitPolicy,
	}
	if spec.Readiness != nil {
		process.ReadyTCP = spec.Readiness.TCP
		process.ReadyHTTP = spec.Readiness.HTTP
		process.ReadyTimeout, _ = readinessDuration(spec.Readiness.Timeout, 30*time.Second)
		process.ReadyInterval, _ = readinessDuration(spec.Readiness.Interval, 100*time.Millisecond)
	}
	pm.Processes = append(pm.Processes, process)
	return nil
}

// Snapshot returns the current state of every configured process, in order. It
// is safe to call from any goroutine and copies all state, so the result is a
// stable point-in-time view a consumer (TUI) can render or diff.
func (pm *ProcessManager) Snapshot() []ProcessInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	infos := make([]ProcessInfo, 0, len(pm.Processes))
	for _, p := range pm.Processes {
		infos = append(infos, p.info())
	}
	return infos
}

// transition updates a process's observable state under lock and then delivers
// an event to OnEvent (outside the lock, so the consumer's handler can call back
// into Snapshot without deadlocking). pid and exitCode are applied only when
// non-zero / not the keep sentinel, so callers can update state alone.
func (pm *ProcessManager) transition(p *Process, state ProcessState, pid int, exitCode int, err error) {
	pm.mu.Lock()
	p.state = state
	if pid != 0 {
		p.pid = pid
	}
	if state == StateRunning {
		p.startedAt = time.Now()
	}
	if exitCode != keepExitCode {
		p.exitCode = exitCode
	}
	if state == StateExited || state == StateFailed || state == StateKilled {
		p.pid = 0
	}
	info := p.info()
	hook := pm.OnEvent
	pm.mu.Unlock()

	if hook != nil {
		hook(ProcessEvent{Info: info, Time: time.Now(), Err: err})
	}
}

// keepExitCode is passed to transition to leave the recorded exit code
// unchanged (used by transitions that aren't process completions).
const keepExitCode = -2

// stdio resolves the writer for one of a process's streams, honoring the Output
// hook and falling back to the process's own stdout/stderr.
func (pm *ProcessManager) stdio(p *Process, stream string, fallback io.Writer) io.Writer {
	pm.mu.RLock()
	out := pm.Output
	info := p.info()
	pm.mu.RUnlock()
	if out == nil {
		return fallback
	}
	if w := out(info, stream); w != nil {
		return w
	}
	return fallback
}

// GetExecutes returns the configured command strings in order.
func (pm *ProcessManager) GetExecutes() []string {
	execs := make([]string, 0, len(pm.Processes))
	for _, p := range pm.Processes {
		execs = append(execs, p.displayCommand())
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
	startupFailures := make(chan error, 1)
	return pm.runCycle(ctx, true, startupFailures)
}

// Reload re-runs blocking steps and restarts the primary process. Background and
// once processes started during Start are left running.
func (pm *ProcessManager) Reload(ctx context.Context) error {
	return pm.runCycle(ctx, false, nil)
}

func (pm *ProcessManager) runCycle(ctx context.Context, firstRun bool, startupFailures chan error) error {
	for _, p := range pm.Processes {
		select {
		case err := <-startupFailures:
			return err
		default:
		}

		// Markers used by the ExecList config form; no-ops in the struct form.
		if p.Exec == KILL_EXEC || p.Exec == REFRESH_EXEC {
			continue
		}
		switch p.Type {
		case Background:
			if !firstRun {
				continue
			}
			onExit := func(err error) {
				select {
				case startupFailures <- err:
				default:
				}
			}
			if err := pm.startAsync(ctx, p, onExit); err != nil {
				slog.Error("starting background process", "exec", p.displayCommand(), "err", err)
				return err
			}
			if err := pm.waitUntilReady(ctx, p); err != nil {
				return err
			}
		case Once:
			if !firstRun {
				continue
			}
			if err := pm.runBlocking(ctx, p, startupFailures); err != nil {
				slog.Error("once process failed", "exec", p.displayCommand(), "err", err)
				return err
			}
		case Blocking:
			if err := pm.runBlocking(ctx, p, startupFailures); err != nil {
				// On reload a failed blocking step (typically a build error)
				// aborts the cycle and leaves the current primary running, so a
				// broken build doesn't take down the last good process.
				slog.Error("blocking process failed", "exec", p.displayCommand(), "err", err)
				return err
			}
		case Primary:
			pm.stopProcess(p) // kill the previous instance (no-op on first run)
			if err := pm.startAsync(ctx, p, nil); err != nil {
				slog.Error("starting primary process", "exec", p.displayCommand(), "err", err)
				return err
			}
			if err := pm.waitUntilReady(ctx, p); err != nil {
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

func (pm *ProcessManager) waitUntilReady(ctx context.Context, p *Process) error {
	if p.ReadyTCP == "" && p.ReadyHTTP == "" {
		return nil
	}
	readyCtx, cancel := context.WithTimeout(ctx, p.ReadyTimeout)
	defer cancel()
	// Cancel an in-flight probe as soon as the process exits.
	go func(done <-chan struct{}) {
		select {
		case <-done:
			cancel()
		case <-readyCtx.Done():
		}
	}(p.done)
	ticker := time.NewTicker(p.ReadyInterval)
	defer ticker.Stop()
	dialer := net.Dialer{Timeout: p.ReadyInterval}
	client := &http.Client{
		Timeout: p.ReadyInterval,
		// A redirect is not a 200 from the configured endpoint.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	target := p.ReadyTCP
	if p.ReadyHTTP != "" {
		target = p.ReadyHTTP
	}
	for {
		ready := false
		if p.ReadyHTTP != "" {
			req, err := http.NewRequestWithContext(readyCtx, http.MethodGet, p.ReadyHTTP, nil)
			if err != nil {
				return fmt.Errorf("creating readiness request: %w", err)
			}
			req.Header.Set("Accept", "*/*")
			response, err := client.Do(req)
			if err == nil {
				ready = response.StatusCode == http.StatusOK
				_ = response.Body.Close()
			}
		} else {
			connection, err := dialer.DialContext(readyCtx, "tcp", p.ReadyTCP)
			if err == nil {
				_ = connection.Close()
				ready = true
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-p.done:
			return fmt.Errorf("process %q exited before becoming ready on %s", p.displayCommand(), target)
		default:
		}
		if readyCtx.Err() != nil {
			return fmt.Errorf("process %q did not become ready on %s within %s: %w", p.displayCommand(), target, p.ReadyTimeout, readyCtx.Err())
		}
		if ready {
			slog.Debug("process is ready", "exec", p.displayCommand(), "target", target)
			return nil
		}
		select {
		case <-readyCtx.Done():
		case <-ticker.C:
		}
	}
}

// delayNext holds for the process's configured delay_next before the cycle moves
// on, letting one step settle (e.g. a service binding a port) before the next
// starts. The wait is context-aware; it returns false if the context is
// cancelled during the pause so the caller can stop the cycle.
func (pm *ProcessManager) delayNext(ctx context.Context, p *Process) bool {
	if p.Delay <= 0 {
		return true
	}
	slog.Debug("delaying before next process", "exec", p.displayCommand(), "ms", p.Delay)
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
func (pm *ProcessManager) startAsync(ctx context.Context, p *Process, onUnexpectedExit func(error)) error {
	procCtx, cancel := context.WithCancel(ctx)
	cmd := p.command()
	cmd.Dir = pm.resolveDir(p.Dir)
	cmd.Env = processEnvironment(p.Env)
	cmd.Stdout = pm.stdio(p, "stdout", os.Stdout)
	cmd.Stderr = pm.stdio(p, "stderr", os.Stderr)
	setProcessGroup(cmd)

	slog.Debug("starting process", "exec", p.displayCommand(), "dir", cmd.Dir)
	if err := cmd.Start(); err != nil {
		cancel()
		pm.transition(p, StateFailed, 0, noExitYet, err)
		return err
	}

	done := make(chan struct{})
	p.cmd = cmd
	p.cancel = cancel
	p.done = done
	pm.transition(p, StateRunning, cmd.Process.Pid, keepExitCode, nil)

	go func() {
		defer close(done)
		waitErr := make(chan error, 1)
		go func() { waitErr <- cmd.Wait() }()
		select {
		case <-procCtx.Done():
			if err := stopProcessTree(cmd, waitErr, p.ShutdownTimeout); err != nil {
				slog.Debug("killing process tree", "exec", p.displayCommand(), "err", err)
			}
			pm.transition(p, StateKilled, 0, noExitYet, nil)
		case err := <-waitErr:
			if err != nil {
				slog.Debug("process exited", "exec", p.displayCommand(), "err", err)
				pm.transition(p, StateFailed, 0, exitCodeOf(cmd, err), err)
			} else {
				pm.transition(p, StateExited, 0, 0, nil)
			}
			if onUnexpectedExit != nil {
				if err != nil {
					onUnexpectedExit(fmt.Errorf("background process %q exited during startup: %w", p.displayCommand(), err))
				} else {
					onUnexpectedExit(fmt.Errorf("background process %q exited during startup", p.displayCommand()))
				}
			}
		}
	}()
	return nil
}

// runBlocking runs a process to completion. The command runs in its own process
// group and is bound to ctx, so a shutdown while it is running force-kills the
// whole tree (not just the direct child, which is all CommandContext would
// reach) and unblocks the wait.
func (pm *ProcessManager) runBlocking(ctx context.Context, p *Process, startupFailures <-chan error) error {
	cmd := p.command()
	cmd.Dir = pm.resolveDir(p.Dir)
	cmd.Env = processEnvironment(p.Env)
	cmd.Stdout = pm.stdio(p, "stdout", os.Stdout)
	cmd.Stderr = pm.stdio(p, "stderr", os.Stderr)
	setProcessGroup(cmd)
	slog.Debug("running blocking process", "exec", p.displayCommand(), "dir", cmd.Dir)

	if err := cmd.Start(); err != nil {
		pm.transition(p, StateFailed, 0, noExitYet, err)
		return err
	}
	pm.transition(p, StateRunning, cmd.Process.Pid, keepExitCode, nil)

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	var err error
	select {
	case <-ctx.Done():
		return pm.stopBlocking(p, cmd, waitErr, ctx.Err())
	case startupErr := <-startupFailures:
		return pm.stopBlocking(p, cmd, waitErr, startupErr)
	case err = <-waitErr:
	}
	if err != nil {
		pm.transition(p, StateFailed, 0, exitCodeOf(cmd, err), err)
		return err
	}
	pm.transition(p, StateExited, 0, 0, nil)
	return nil
}

func (pm *ProcessManager) stopBlocking(p *Process, cmd *exec.Cmd, waitErr <-chan error, cause error) error {
	if err := stopProcessTree(cmd, waitErr, p.ShutdownTimeout); err != nil {
		slog.Debug("stopping blocking process tree", "exec", p.displayCommand(), "err", err)
	}
	pm.transition(p, StateKilled, 0, noExitYet, nil)
	return cause
}

func stopProcessTree(cmd *exec.Cmd, wait <-chan error, grace time.Duration) error {
	if grace <= 0 {
		err := killProcessTree(cmd)
		<-wait
		return err
	}
	if err := terminateProcessTree(cmd); err != nil {
		forceErr := killProcessTree(cmd)
		<-wait
		if forceErr != nil {
			return errors.Join(err, forceErr)
		}
		return err
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-wait:
		return nil
	case <-timer.C:
		err := killProcessTree(cmd)
		<-wait
		return err
	}
}

// exitCodeOf extracts the process exit code from a completed command, falling
// back to -1 when the failure wasn't a normal non-zero exit (e.g. a signal).
func exitCodeOf(cmd *exec.Cmd, err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return noExitYet
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
