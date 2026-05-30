# Kitchen-sink example

Exercises **every** execute type in a single config, plus a reload callback and
ignore rules. Doubles as the project's end-to-end integration test
([`main_test.go`](./main_test.go)).

| Type         | Demo command writes to     | Behavior verified                          |
| ------------ | -------------------------- | ------------------------------------------ |
| `once`       | `artifacts/once.log`       | Runs exactly once, never re-runs on reload |
| `background` | `artifacts/background.log` | Starts once, survives reloads              |
| `blocking`   | `artifacts/blocking.log`   | Re-runs every cycle, finishes before primary restarts |
| `primary`    | `artifacts/primary.log`    | Killed and restarted on every reload       |

## Run

```sh
go run .
```

Then edit [`watched/trigger.go`](./watched/trigger.go) and save — the blocking
step re-runs and the primary restarts, while `once`/`background` stay put. The
reload callback logs each detected change. `Ctrl-C` stops everything cleanly.

> Uses a POSIX shell (`sh`), so run on Linux or macOS.

## As an integration test

`main_test.go` builds the same config against a temp directory, starts the real
engine, triggers a filesystem reload, and asserts each type's semantics from the
`artifacts/` markers — then shuts down and confirms no processes leak:

```sh
go test ./examples/kitchen-sink/
```

Runtime output goes to `artifacts/` (gitignored).
