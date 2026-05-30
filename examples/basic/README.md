# Basic example

Embeds refresh as a library: watches this directory for `*.go` changes and, on
each change, rebuilds and restarts the small program in [`app/`](./app).

## Run

```sh
go run .
```

Then edit [`app/main.go`](./app/main.go) — refresh rebuilds `bin/app` and
restarts it. Press `Ctrl-C` to stop.

## Layout

| Path            | Purpose                                            |
| --------------- | -------------------------------------------------- |
| `main.go`       | Configures and starts the engine (library usage).  |
| `app/main.go`   | The supervised long-running process.               |
| `config.yaml`   | The same configuration as a file (`NewEngineFromYAML`). |

The built binary is written to `bin/` (gitignored).
