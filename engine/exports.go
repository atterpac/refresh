package engine

import "github.com/atterpac/refresh/process"

// Re-exports of the process package's public types and values, so callers can
// configure an engine entirely through the engine package — matching the
// documented `refresh.Execute{...}` / `refresh.KILL_EXEC` usage without a second
// import.
type (
	Execute     = process.Execute
	ExecuteType = process.ExecuteType
)

var (
	Background = process.Background
	Once       = process.Once
	Blocking   = process.Blocking
	Primary    = process.Primary

	// KILL_STALE is a marker execute (struct form) indicating where a stale
	// primary should be terminated. The supervisor now restarts the primary
	// automatically, so it is accepted for backwards compatibility and is a
	// no-op in the run cycle.
	KILL_STALE = process.KILL_STALE

	// KILL_EXEC / REFRESH_EXEC are markers for the ExecList (string) form.
	// REFRESH_EXEC marks the command that follows it as the primary process.
	KILL_EXEC    = process.KILL_EXEC
	REFRESH_EXEC = process.REFRESH_EXEC
)
