//go:build linux || darwin

package process

import (
	"context"
	// "log/slog"
	"os"
	"syscall"
	"time"
)

var firstRun = true

func (pm *ProcessManager) StartProcess(ctx context.Context) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.Processes) == 0 {
		// slog.Warn("No Processes to Start")
		return
	}

	// slog.Info("Starting Processes", "count", len(pm.processes))

	for _, p := range pm.Processes {
		if p.Exec == "KILL_STALE" {
			continue
		}
		if !firstRun && p.Background {
			continue
		}

		cmd := generateExec(p.Exec)
		p.cmd = cmd

		if p.Primary {
			if !firstRun {
				// slog.Debug("Not first run, killing processes")
				for _, pr := range pm.Processes {
					if !pr.Background {
						// check if pid is running
						if pr.pid != 0 {
							_, err := os.FindProcess(pr.pid)
							if err != nil {
								// slog.Debug("Process not running", "exec", pr.Exec)
								continue
							}
						}
						// slog.Debug("Checking for stale process", "exec", pr.Exec)
						delete(pm.Ctxs, pr.Exec)
						delete(pm.Cancels, pr.Exec)

						// Wait for the process to terminate
						select {
						case <-ctx.Done():
							// slog.Debug("Process terminated", "exec", pr.Exec)
						case <-time.After(100 * time.Millisecond):
							// slog.Debug("Process not terminated... killing", "exec", pr.Exec)
						}

						// Kill any remaining child processes
						if pr.pgid != 0 {
							// slog.Debug("Killing process group", "pgid", pr.pgid)
							syscall.Kill(-pr.pgid, syscall.SIGKILL)
						}
					}
				}
				// slog.Debug("Processes killed")
				time.Sleep(200 * time.Millisecond)
			} else {
				// slog.Debug("First run, not killing processes")
				firstRun = false
			}
			// Log Buffers
		}

		var err error
		if p.Blocking {
			err = cmd.Run()
		} else {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			err = cmd.Start()
			if cmd.Process != nil {
				p.pgid, _ = syscall.Getpgid(cmd.Process.Pid)
				p.pid = cmd.Process.Pid

				processCtx, processCancel := context.WithCancel(ctx)
				pm.Ctxs[p.Exec] = processCtx
				pm.Cancels[p.Exec] = processCancel
				// slog.Debug("Stored Process Context", "exec", p.Exec)

				go func() {
					select {
					case <-processCtx.Done():
						// slog.Warn("Killing Process", "exec", p.Exec, "pgid", p.pgid, "pid", p.pid)
						syscall.Kill(-p.pid, syscall.SIGKILL)
						// slog.Debug("Process Terminated", "exec", p.Exec)
					case <-ctx.Done():
						// slog.Debug("Context Done", "exec", p.Exec)
						syscall.Kill(-p.pid, syscall.SIGKILL)
					default:
						cmd.Wait()
						// slog.Debug("Process Done", "exec", p.Exec)
						delete(pm.Ctxs, p.Exec)
						delete(pm.Cancels, p.Exec)
					}
				}()
			}
		}

		if err != nil {
			// slog.Error("Running Command", "exec", p.Exec, "err", err)
		}
	}
	firstRun = false
}

func (pm *ProcessManager) KillProcesses() {
	for _, p := range pm.Processes {
		// slog.Debug("Killing Process", "exec", p.Exec, "pid", p.pid)
		if p.pid != 0 {
			_, err := os.FindProcess(p.pid)
			if err != nil {
				// slog.Debug("Process not running", "exec", p.Exec)
				continue
			}
			syscall.Kill(-p.pid, syscall.SIGKILL)
		}
	}
}
