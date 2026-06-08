package process

import (
	"io"
	"time"
)

// ProcessState is the lifecycle state of a single process instance. It is the
// value an upstream consumer (e.g. a TUI) renders in a status column.
type ProcessState string

const (
	// StatePending is the initial state of a configured process that has not yet
	// been started in the current run.
	StatePending ProcessState = "pending"
	// StateRunning means the process has been started and has not yet exited.
	StateRunning ProcessState = "running"
	// StateExited means the process finished on its own with a zero exit code.
	StateExited ProcessState = "exited"
	// StateFailed means the process finished on its own with a non-zero exit code.
	StateFailed ProcessState = "failed"
	// StateKilled means the process was terminated by refresh (a reload restarting
	// a primary, or shutdown), rather than exiting on its own.
	StateKilled ProcessState = "killed"
)

// ProcessInfo is an immutable snapshot of a process's identity and current
// runtime state. It carries no live handles, so a consumer may hold and compare
// snapshots freely across goroutines.
type ProcessInfo struct {
	// Name is the stable identifier for the process, suitable as a key for a
	// per-process log pane. It defaults to the command string when not set.
	Name string
	// Exec is the configured command string.
	Exec string
	// Type is the process's execute type (background, once, blocking, primary).
	Type ExecuteType
	// State is the lifecycle state at the moment the snapshot was taken.
	State ProcessState
	// PID is the operating-system process id, or 0 when not running.
	PID int
	// StartedAt is when the current instance was started; zero if never started.
	StartedAt time.Time
	// ExitCode is the exit code of the last completed run, or -1 when the process
	// was killed or has not yet exited.
	ExitCode int
}

// ProcessEvent is delivered to an OnEvent hook every time a process changes
// state. It pairs the new snapshot with the time of the transition and, for
// failures, the underlying error.
type ProcessEvent struct {
	Info ProcessInfo
	Time time.Time
	// Err is set when a process failed or could not be started; nil otherwise.
	Err error
}

// OutputFunc resolves the writer that a process's stdout or stream output is
// wired to. stream is either "stdout" or "stderr". Returning nil falls back to
// the process's own os.Stdout/os.Stderr, preserving the default terminal
// behavior. A TUI returns a per-process buffer here to capture each process's
// output separately.
type OutputFunc func(info ProcessInfo, stream string) io.Writer

// EventFunc receives process lifecycle events. It is called synchronously from
// the goroutine that drives the transition, so it must not block; a consumer
// that needs to do real work should hand the event off to its own channel.
type EventFunc func(ProcessEvent)

const noExitYet = -1

// info builds a snapshot from a process's current fields. Callers must hold
// pm.mu (read or write) so the runtime fields are read consistently.
func (p *Process) info() ProcessInfo {
	name := p.Name
	if name == "" {
		name = p.Exec
	}
	return ProcessInfo{
		Name:      name,
		Exec:      p.Exec,
		Type:      p.Type,
		State:     p.state,
		PID:       p.pid,
		StartedAt: p.startedAt,
		ExitCode:  p.exitCode,
	}
}
