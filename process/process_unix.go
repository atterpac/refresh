//go:build linux || darwin

package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"
)

func (pm *ProcessManager) StartProcess(ctx context.Context, cancel context.CancelFunc) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.Processes) == 0 {
		return
	}
	for _, p := range pm.Processes {
		slog.Debug("Starting Process", "exec", p.Exec)
		if p.Exec == "KILL_STALE" {
			continue
		}
		if !pm.FirstRun && p.Background {
			continue
		}
		cmd := generateExec(p.Exec)
		p.cmd = cmd
		if p.Primary {
			// Ensure previous processes are killed if this isnt the first run
			if !pm.FirstRun {
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
						// Remove contexts
						delete(pm.Ctxs, pr.Exec)
						delete(pm.Cancels, pr.Exec)
						// Wait for the process to terminate
						select {
						case <-ctx.Done():
							slog.Debug("Process terminated", "exec", pr.Exec)
						case <-time.After(100 * time.Millisecond):
							slog.Debug("Process not terminated... forcefully killing", "exec", pr.Exec)
						}
						// Kill any remaining child processes
						if pr.pgid != 0 {
							// slog.Debug("Killing process group", "pgid", pr.pgid)
							syscall.Kill(-pr.pgid, syscall.SIGKILL)
						}
					}
				}
				time.Sleep(200 * time.Millisecond)
			} else {
				pm.FirstRun = false
			}
		}
		var err error
		if p.Blocking {
			output, err := cmd.CombinedOutput()
			if len(output) > 0 {
				fmt.Println(string(output))
			}
			if err != nil {
				slog.Error("Running Command", "exec", p.Exec, "err", err)
				cancel()
				return
			}
		} else {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Stderr = os.Stderr
			p.logPipe, err = cmd.StdoutPipe()
			if err != nil {
				slog.Error("Getting Stdout Pipe", "exec", p.Exec, "err", err)
			}
			go printSubProcess(ctx, p.logPipe)
			err = cmd.Start()
			if cmd.Process == nil {
				slog.Error("Process is nil", "exec", p.Exec)
				continue
			}
			p.pgid, _ = syscall.Getpgid(cmd.Process.Pid)
			p.pid = cmd.Process.Pid

			processCtx, processCancel := context.WithCancel(ctx)
			pm.Ctxs[p.Exec] = processCtx
			pm.Cancels[p.Exec] = processCancel
			// slog.Debug("Stored Process Context", "exec", p.Exec)

			go func() {
				select {
				case <-processCtx.Done():
					_ = syscall.Kill(-p.pid, syscall.SIGKILL)
				case <-ctx.Done():
					slog.Debug("Context closed", "exec", p.Exec)
					_ = syscall.Kill(-p.pid, syscall.SIGKILL)
				default:
					err := cmd.Wait()
					if err != nil {
						cancel()
					}
					delete(pm.Ctxs, p.Exec)
					delete(pm.Cancels, p.Exec)
				}
			}()
		}
		if err != nil {
			slog.Error("Running Command", "exec", p.Exec, "err", err)
			cancel()
		}
	}
	pm.FirstRun = false
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
