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
				for _, pr := range pm.Processes {
					if pr.Type != Background {
						// check if pid is running
						if pr.pid != 0 {
							if _, err := os.FindProcess(pr.pid); err == nil {
								if cancel, exists := pm.Cancels[pr.Exec]; exists {
									cancel()
									delete(pm.Ctxs, pr.Exec)
									delete(pm.Cancels, pr.Exec)
								}

								time.Sleep(100 * time.Millisecond)

								if err := taskKill(pr.pid); err != nil {
									slog.Debug("Failed to kill process", "exec", pr.Exec, "pid", pr.pid, "err", err)
								}
							}
						}
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
			if err != nil {
				slog.Error("Getting stdout pipe", "exec", p.Exec, "err", err)
				pm.RestoreRootDirectory()
				cancel()
				return
			}

			subProcessCtx, subProcessCancel := context.WithCancel(ctx)
			go printSubProcess(subProcessCtx, p.logPipe)

			err = cmd.Run()
			subProcessCancel()

			if err != nil {
				slog.Error("Running Command", "exec", p.Exec, "err", err)
				pm.RestoreRootDirectory()
				cancel()
				return
			}
		} else {
			cmd.Stderr = os.Stderr
			p.logPipe, err = cmd.StdoutPipe()
			if err != nil {
				slog.Error("Getting Stdout Pipe", "exec", p.Exec, "err", err)
				pm.RestoreRootDirectory()
				cancel()
				return
			}

			err = cmd.Start()
			if err != nil {
				slog.Error("Starting command", "exec", p.Exec, "err", err)
				pm.RestoreRootDirectory()
				cancel()
				return
			}

			if cmd.Process != nil {
				p.pid = cmd.Process.Pid
				processCtx, processCancel := context.WithCancel(ctx)
				pm.Ctxs[p.Exec] = processCtx
				pm.Cancels[p.Exec] = processCancel

				subProcessCtx, subProcessCancel := context.WithCancel(processCtx)
				go printSubProcess(subProcessCtx, p.logPipe)

				go func(exec string, pid int, subCancel context.CancelFunc) {
					defer subCancel()

					select {
					case <-processCtx.Done():
						if err := taskKill(pid); err != nil {
							slog.Debug("Failed to kill process after context done", "exec", exec, "pid", pid, "err", err)
						}
					case <-ctx.Done():
						if err := taskKill(pid); err != nil {
							slog.Debug("Failed to kill process after parent context done", "exec", exec, "pid", pid, "err", err)
						}
					default:
						err := cmd.Wait()
						if err != nil {
							slog.Error("Process exited with error", "exec", exec, "err", err)
							cancel()
						}

						pm.mu.Lock()
						delete(pm.Ctxs, exec)
						delete(pm.Cancels, exec)
						pm.mu.Unlock()
					}
				}(p.Exec, p.pid, subProcessCancel)
			}
		}

		if err != nil {
			slog.Error("Executing command", "command", p.Exec, "err", err)
			cancel()
		}
	}

	pm.FirstRun = false
}

// Window specific kill process
func (pm *ProcessManager) KillProcesses() {
	// slog.Debug("Killing Processes")
	for _, p := range pm.Processes {
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
