//go:build windows

package process

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func (pm *ProcessManager) StartProcess(ctx context.Context, cancel context.CancelFunc) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.Processes) == 0 {
		// slog.Warn("No Processes to Start")
		return
	}
	for _, p := range pm.Processes {
		if p.Exec == "KILL_STALE" {
			continue
		}
		if !pm.FirstRun && p.Type == Background {
			continue
		}

		cmd := generateExec(p.Exec)
		p.cmd = cmd

		if p.Type == Primary {
			if !pm.FirstRun {
				// slog.Debug("Not first run, killing processes")
				for _, pr := range pm.Processes {
					if pr.Type != Background {
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
						taskKill(pr.pid)
					}
				}
				// slog.Debug("Processes killed")
				time.Sleep(200 * time.Millisecond)
			} else {
				// slog.Debug("First run, not killing processes")
				pm.FirstRun = false
			}
			// Log buffers
		}
		pm.ChangeExecuteDirectory(p.Dir)
		defer pm.RestoreRootDirectory()
		var err error
		if p.Type == Blocking || p.Type == Once {
			if p.Type == Once && !pm.FirstRun {
				continue
			}
			cmd.Stderr = os.Stderr
			p.logPipe, err = cmd.StdoutPipe()
			go printSubProcess(ctx, p.logPipe)
			err = cmd.Run()
			if err != nil {
				slog.Error("Running Command", "exec", p.Exec, "err", err)
				cancel()
				return
			}
		} else {
			cmd.Stderr = os.Stderr
			p.logPipe, err = cmd.StdoutPipe()
			go printSubProcess(ctx, p.logPipe)
			err = cmd.Start()
			if err != nil {
				slog.Error("Getting Stdout Pipe", "exec", p.Exec, "err", err)
			}
			if cmd.Process != nil {
				p.pid = cmd.Process.Pid
				processCtx, processCancel := context.WithCancel(ctx)
				pm.Ctxs[p.Exec] = processCtx
				pm.Cancels[p.Exec] = processCancel

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
						err := cmd.Wait()
						if err != nil {
							cancel()
						}
						// slog.Debug("Process Done", "exec", p.Exec)
						delete(pm.Ctxs, p.Exec)
						delete(pm.Cancels, p.Exec)
					}
				}()
			}
		}

		if err != nil {
			slog.Error("Running Command", "exec", p.Exec, "err", err)
			cancel()
		}
	}

	pm.FirstRun = false
}

// Window specific kill process
func (pm *ProcessManager) KillProcesses() {
	// slog.Debug("Killing Processes")
	for _, p := range pm.Processes {
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
