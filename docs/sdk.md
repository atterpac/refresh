# Embedding refresh: the process SDK

`refresh` is usable as a library, not just a CLI. An upstream application can
embed the engine, **tap each process's stdout/stderr separately**, observe
process lifecycle, and poll live process state — everything needed to drive a
per-process TUI (think turbo's TUI mode) on top of the runner.

This document covers that SDK surface. For watching/config basics see the main
`README.md`.

---

## The three taps

There are three integration points, all opt-in via `engine.Config`. When you set
none of them, behavior is unchanged: process output goes straight to the
terminal.

| Tap | Config field | Direction | Use it for |
|-----|--------------|-----------|------------|
| **Output** | `Output OutputFunc` | push (bytes) | per-process log panes |
| **Events** | `OnProcessEvent EventFunc` | push (state) | status changes, restart notifications |
| **Snapshot** | `engine.Processes()` | pull (state) | rendering a status table on a tick |

The split is deliberate: stream **output is pushed** (you can't poll bytes), while
**status is pulled** on your render tick to avoid backpressure on the engine's
supervisor goroutine. Events exist for when you need to react to a transition the
moment it happens (e.g. flash a pane red on crash) rather than wait for the next
tick.

---

## 1. Capturing per-process output

Set `Config.Output`. It is called once per stream (`"stdout"` and `"stderr"`)
when a process starts, and returns the `io.Writer` that stream is wired to.

```go
import (
    "io"

    "github.com/atterpac/refresh/engine"
)

// One buffer per process name — these are what your TUI renders as panes.
panes := map[string]io.Writer{
    "build":  newPaneBuffer(),
    "server": newPaneBuffer(),
}

cfg := engine.Config{
    RootPath: "./",
    ExecStruct: []engine.Execute{
        {Name: "build",  Cmd: "go build -o ./app", Type: engine.Blocking},
        {Name: "server", Cmd: "./app",             Type: engine.Primary},
    },
    Output: func(info engine.ProcessInfo, stream string) io.Writer {
        // Route both streams of a process into its pane. Return a distinct
        // writer per stream if you want to style stderr differently.
        return panes[info.Name]
    },
}
```

Key points:

- **`info.Name` is the routing key.** Set `Name` on each `Execute`. If you leave
  it empty it defaults to the command string, but a stable explicit name is what
  you want as a pane key.
- **Returning `nil` keeps the default** (the process's own `os.Stdout` /
  `os.Stderr`). So you can capture some processes and let others print normally:

  ```go
  Output: func(info engine.ProcessInfo, stream string) io.Writer {
      if p, ok := panes[info.Name]; ok {
          return p
      }
      return nil // not a tracked process — leave it on the terminal
  }
  ```
- **You fully own the writer.** Unlike a tee, refresh does *not* also copy to the
  terminal when you return a writer — important for a TUI that owns the screen.
  If you *want* a tee, return `io.MultiWriter(os.Stdout, yourBuffer)` yourself.
- **Your writer is called from the process's own goroutine.** Make it
  goroutine-safe (guard a `bytes.Buffer` with a mutex, or write to a channel).

---

## 2. Observing lifecycle (events)

Set `Config.OnProcessEvent` to receive a `ProcessEvent` on every state change.

```go
cfg.OnProcessEvent = func(ev engine.ProcessEvent) {
    // ev.Info is a snapshot; ev.Time is when it changed; ev.Err is set on failure.
    log.Printf("%s -> %s (pid %d)", ev.Info.Name, ev.Info.State, ev.Info.PID)
    if ev.Info.State == engine.StateFailed {
        tui.FlashRed(ev.Info.Name, ev.Err)
    }
}
```

States (`engine.ProcessState`):

| State | Meaning |
|-------|---------|
| `StatePending` | configured, not yet started in this run |
| `StateRunning` | started; `PID` is live |
| `StateExited`  | finished on its own, exit code 0 |
| `StateFailed`  | finished on its own, non-zero exit (`ExitCode` + `Err` set) |
| `StateKilled`  | terminated by refresh (a reload restarting the primary, or shutdown) |

Typical sequences:

- **blocking/once step:** `Running → Exited` (or `Failed`)
- **primary:** `Running`, then on each reload `Killed → Running`, and `Killed` at shutdown
- **background:** `Running` once, `Killed` at shutdown

> `OnProcessEvent` is called **synchronously from the engine's goroutine — do not
> block in it.** If you need to do real work, hand the event to your own channel
> and return immediately.

---

## 3. Polling state (snapshot)

`engine.Processes()` returns a `[]ProcessInfo` snapshot of every configured
process and its current state. Safe to call from any goroutine, including before
the engine starts (everything reports `StatePending`). This is the natural fit
for a TUI that redraws on a tick:

```go
for range ticker.C {
    for _, p := range eng.Processes() {
        tui.Row(p.Name, p.State, p.PID, time.Since(p.StartedAt))
    }
}
```

`ProcessInfo` is a value copy with no live handles, so you can hold and diff
snapshots freely.

```go
type ProcessInfo struct {
    Name      string        // stable id / pane key (defaults to Exec)
    Exec      string        // the command string
    Type      ExecuteType   // background | once | blocking | primary
    State     ProcessState
    PID       int           // 0 when not running
    StartedAt time.Time     // zero if never started
    ExitCode  int           // last completed run; -1 if killed / never exited
}
```

---

## Driving the engine: `Run(ctx)` vs `Start()`

A TUI owns the terminal and the keyboard, so it must **not** let the engine grab
OS signals. Use `Run(ctx)` instead of `Start()`:

| | `Start()` | `Run(ctx)` |
|--|-----------|------------|
| Blocks until | SIGINT/SIGTERM | `ctx` cancelled (or `Stop()`) |
| Traps SIGINT/SIGTERM | yes | **no** |
| Traps Ctrl+Z pause (if `EnablePause`) | yes | **no** — you drive pause via your own input |
| Intended caller | the CLI | an embedding app / TUI |

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

eng, err := engine.NewEngineFromConfig(cfg)
if err != nil {
    return err
}

go func() {
    if err := eng.Run(ctx); err != nil {
        log.Printf("engine: %v", err)
    }
}()

// ... run your TUI event loop; on quit:
cancel() // engine tears down all processes cleanly, Run returns nil
```

`Run` blocks, so call it from its own goroutine. Cancelling `ctx` (or calling
`eng.Stop()`) triggers a clean shutdown: every process is killed, you get the
matching `StateKilled` events, and `Run` returns `nil`.

Because `Run` installs no signal traps, **your TUI owns pause/resume** — drive it
through the control API below.

---

## Controlling the engine: Reload / Pause / Resume

Under `Run(ctx)` the engine installs no Ctrl+Z handler, so the embedding app
drives the supervisor through four methods. All are safe to call from any
goroutine (your input handler, a button, a keybinding) and are non-blocking.

| Method | Effect |
|--------|--------|
| `Reload()` | Trigger a reload cycle (re-run blocking steps, restart the primary), exactly as a file change would. Deferred if paused. |
| `Pause()` | Suspend reload handling. File changes and `Reload` calls made while paused are remembered, not dropped. Idempotent. |
| `Resume()` | Re-enable reloads and apply any single change that arrived while paused. Idempotent. |
| `Paused() bool` | Report the current pause state (e.g. to render a "PAUSED" badge). |

```go
// Wire a TUI keymap to the control API.
switch key {
case 'r':
    eng.Reload()          // force a rebuild/restart on demand
case ' ':
    if eng.Paused() {
        eng.Resume()
    } else {
        eng.Pause()
    }
}
```

Semantics worth knowing:

- **Pause defers, it does not drop.** A reload that arrives while paused (whether
  from a file change or a `Reload()` call) is coalesced into a single pending
  reload and applied the moment you `Resume()`. Multiple changes while paused
  still result in exactly one reload on resume.
- **A no-op resume does nothing.** `Resume()` with no change pending will not
  restart the primary.
- **These work the same with `Start()`.** The CLI's Ctrl+Z toggle (when
  `EnablePause` is set) is just a wrapper around `Pause`/`Resume`; calling them
  programmatically and pressing Ctrl+Z share one pause flag. With `Run(ctx)` the
  Ctrl+Z trap is off, but the methods still work.
- **`Reload()` honors pause**: a forced reload while paused is itself deferred to
  resume, so a paused engine truly holds still.

---

## Full example

```go
package main

import (
    "bytes"
    "context"
    "io"
    "log"
    "sync"
    "time"

    "github.com/atterpac/refresh/engine"
)

// pane is a goroutine-safe buffer standing in for a TUI log pane.
type pane struct {
    mu  sync.Mutex
    buf bytes.Buffer
}

func (p *pane) Write(b []byte) (int, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    return p.buf.Write(b)
}
func (p *pane) String() string {
    p.mu.Lock()
    defer p.mu.Unlock()
    return p.buf.String()
}

func main() {
    panes := map[string]*pane{"build": {}, "server": {}}

    cfg := engine.Config{
        RootPath: "./",
        LogLevel: "mute", // silence engine chatter; you render your own UI
        ExecStruct: []engine.Execute{
            {Name: "build",  Cmd: "go build -o ./app", Type: engine.Blocking},
            {Name: "server", Cmd: "./app",             Type: engine.Primary},
        },
        Output: func(info engine.ProcessInfo, _ string) io.Writer {
            if p, ok := panes[info.Name]; ok {
                return p
            }
            return nil
        },
        OnProcessEvent: func(ev engine.ProcessEvent) {
            log.Printf("event: %s -> %s", ev.Info.Name, ev.Info.State)
        },
    }

    eng, err := engine.NewEngineFromConfig(cfg)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go func() { _ = eng.Run(ctx) }()

    // Stand-in render loop: print status + pane contents every second.
    for range time.Tick(time.Second) {
        for _, p := range eng.Processes() {
            log.Printf("[%s] %s pid=%d", p.Name, p.State, p.PID)
        }
        log.Printf("server log:\n%s", panes["server"].String())
    }
}
```

---

## Threading & safety notes

- `Output` and `OnProcessEvent` callbacks run on engine-owned goroutines. Keep
  them non-blocking and make any shared state goroutine-safe.
- `Processes()` is `RWMutex`-guarded and copies everything — call it as often as
  your render loop needs.
- An event handler may safely call `Processes()`; events are dispatched outside
  the engine's internal lock specifically to allow this.
- All of this is additive. With no hooks set, `refresh` behaves exactly as the
  CLI does today.

---

## API reference (engine package)

```go
// Config fields
Output         engine.OutputFunc                  // func(ProcessInfo, stream string) io.Writer
OnProcessEvent engine.EventFunc                   // func(ProcessEvent)

// Engine methods
func (e *Engine) Run(ctx context.Context) error   // supervise until ctx cancelled; no signal traps
func (e *Engine) Start() error                     // CLI entry; traps OS signals, blocks
func (e *Engine) Stop()                            // request graceful shutdown
func (e *Engine) Processes() []engine.ProcessInfo  // live snapshot, any goroutine
func (e *Engine) Reload()                          // force a reload cycle (deferred if paused)
func (e *Engine) Pause()                           // suspend reloads; remembers a deferred change
func (e *Engine) Resume()                          // re-enable reloads; applies a deferred change
func (e *Engine) Paused() bool                     // current pause state

// Types (re-exported from the process package)
engine.ProcessInfo
engine.ProcessEvent
engine.ProcessState
engine.OutputFunc
engine.EventFunc

// State constants
engine.StatePending engine.StateRunning engine.StateExited
engine.StateFailed  engine.StateKilled
```
