//go:build windows

package engine

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func (e *Engine) StartProcess(ctx context.Context) {
	pm := e.ProcessManager
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.processes) == 0 {
		// slog.Warn("No Processes to Start")
		return
	}

	// slog.Info("Starting Processes", "count", len(pm.processes))

	for _, p := range pm.processes {
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
				for _, pr := range pm.processes {
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
						delete(pm.ctxs, pr.Exec)
						delete(pm.cancels, pr.Exec)

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
							taskKill(-pr.pid)
						}
					}
				}
				// slog.Debug("Processes killed")
				time.Sleep(200 * time.Millisecond)
			} else {
				// slog.Debug("First run, not killing processes")
				firstRun = false
			}
			if !e.Config.externalSlog {
				cmd.Stderr = os.Stderr
				e.ProcessLogPipe, _ = cmd.StdoutPipe()
				go printSubProcess(ctx, e.ProcessLogPipe)
			}
		}

		var err error
		if p.Blocking {
			err = cmd.Run()
		} else {
			err = cmd.Start()
			if cmd.Process != nil {
				p.pid = cmd.Process.Pid

				processCtx, processCancel := context.WithCancel(ctx)
				pm.ctxs[p.Exec] = processCtx
				pm.cancels[p.Exec] = processCancel
				// slog.Debug("Stored Process Context", "exec", p.Exec)

				go func() {
					select {
					case <-processCtx.Done():
						// slog.Warn("Killing Process", "exec", p.Exec, "pgid", p.pgid, "pid", p.pid)
						taskKill(p.pid)
						// slog.Debug("Process Terminated", "exec", p.Exec)
					case <-ctx.Done():
						// slog.Debug("Context Done", "exec", p.Exec)
						taskKill(p.pid)
					default:
						cmd.Wait()
						// slog.Debug("Process Done", "exec", p.Exec)
						delete(pm.ctxs, p.Exec)
						delete(pm.cancels, p.Exec)
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

// Window specific kill process
func (pm *ProcessManager) KillProcesses() {
	// slog.Debug("Killing Processes")
	for _, p := range pm.processes {
		if p.pgid == 0 {
			continue
		}
		err := taskKill(p.pid)
		if err != nil {
			// slog.Error("Error killing process", "pid", p.cmd.Process.Pid, "err", err.Error())
		}
	}
}

func taskKill(pid int) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	err := kill.Run()
	if err != nil {
		// slog.Error("Error killing process", "pid", pid, "err", err.Error())
		return err
	}
	// slog.Debug("Process successfull killed", "pid", pid)
	return nil
}
