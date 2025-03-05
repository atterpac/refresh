//go:build linux || darwin

package process

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func (pm *ProcessManager) StartProcess(ctx context.Context, cancel context.CancelFunc) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Store the original directory to ensure we restore it at the end of function
	originalDir, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get current working directory", "err", err)
		// If we can't get the current directory, use our saved RootDir
		originalDir = pm.RootDir
	}

	// Ensure we always restore the original directory when this function exits
	defer func() {
		err := os.Chdir(originalDir)
		if err != nil {
			slog.Error("Failed to restore original directory", "dir", originalDir, "err", err)
		}
	}()

	if len(pm.Processes) == 0 {
		slog.Warn("No Processes to Start")
		os.Exit(1)
		return
	}
	for _, p := range pm.Processes {
		slog.Debug("Starting Process", "exec", p.Exec)
		if p.Exec == "KILL_STALE" {
			continue
		}
		if !pm.FirstRun && p.Type == Background {
			continue
		}
		cmd := generateExec(p.Exec)
		p.cmd = cmd
		if p.Type == Primary {
			// Ensure previous processes are killed if this isnt the first run
			if !pm.FirstRun {
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
		if p.Type == Blocking || p.Type == Once {
			if !pm.FirstRun && p.Type == Once {
				continue
			}
			cmd.Stderr = os.Stderr
			p.logPipe, err = cmd.StdoutPipe()
			if err != nil {
				slog.Error("Getting Stdout Pipe", "exec", p.Exec, "err", err)
			}
			go printSubProcess(ctx, p.logPipe)

			// Change to the command's directory if specified
			if p.Dir != "" {
				targetDir := p.Dir
				if !filepath.IsAbs(p.Dir) {
					// If relative path, make it relative to RootDir
					targetDir = filepath.Join(pm.RootDir, p.Dir)
				}
				currentDir, _ := os.Getwd()
				slog.Debug("Changing directory for process", "from", currentDir, "to", targetDir, "process", p.Exec)
				err = os.Chdir(targetDir)
				if err != nil {
					slog.Error("Failed to change directory", "dir", targetDir, "err", err)
					cancel()
					return
				}
			}

			err = cmd.Run()
			if err != nil {
				slog.Error("Running Command", "exec", p.Exec, "err", err)
				cancel()
				return
			}
			slog.Debug("Process completed closing context", "exec", p.Exec)
			ctx.Done()
		} else {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Stderr = os.Stderr
			p.logPipe, err = cmd.StdoutPipe()
			if err != nil {
				slog.Error("Getting Stdout Pipe", "exec", p.Exec, "err", err)
			}
			go printSubProcess(ctx, p.logPipe)

			// Change to the command's directory if specified
			if p.Dir != "" {
				targetDir := p.Dir
				if !filepath.IsAbs(p.Dir) {
					// If relative path, make it relative to RootDir
					targetDir = filepath.Join(pm.RootDir, p.Dir)
				}
				currentDir, _ := os.Getwd()
				slog.Debug("Changing directory for process", "from", currentDir, "to", targetDir, "process", p.Exec)
				err = os.Chdir(targetDir)
				if err != nil {
					slog.Error("Failed to change directory", "dir", targetDir, "err", err)
					cancel()
					continue
				}
			}

			err = cmd.Start()
			if cmd.Process == nil {
				slog.Error("Primary process not running", "exec", p.Exec)
				cancel()
				continue
			}

			p.pgid, _ = syscall.Getpgid(cmd.Process.Pid)
			p.pid = cmd.Process.Pid

			processCtx, processCancel := context.WithCancel(ctx)
			pm.Ctxs[p.Exec] = processCtx
			pm.Cancels[p.Exec] = processCancel
			// slog.Debug("Stored Process Context", "exec", p.Exec)

			go func() {
				errCh := make(chan error, 1)
				go func() {
					errCh <- cmd.Wait()
				}()
				select {
				case <-processCtx.Done():
					_ = syscall.Kill(-p.pid, syscall.SIGKILL)
				case <-ctx.Done():
					slog.Debug("Context closed", "exec", p.Exec)
					_ = syscall.Kill(-p.pid, syscall.SIGKILL)
				case err := <-errCh:
					if err != nil {
						cancel()
					}
					slog.Debug("Process Errored closing context", "exec", p.Exec)
					ctx.Done()
					delete(pm.Ctxs, p.Exec)
					delete(pm.Cancels, p.Exec)
				}
			}()
		}
		if err != nil {
			slog.Error("Running Command", "exec", p.Exec, "err", err)
			cancel()
		}

		// After each process, restore to the original directory
		err = os.Chdir(originalDir)
		if err != nil {
			slog.Error("Failed to restore directory after process", "dir", originalDir, "err", err)
		}
	}
	pm.FirstRun = false
}

func (pm *ProcessManager) KillProcesses() {
	for _, p := range pm.Processes {
		if p.pid != 0 {
			_, err := os.FindProcess(p.pid)
			if err != nil {
				continue
			}
			syscall.Kill(-p.pid, syscall.SIGKILL)
			if cancel, ok := pm.Cancels[p.Exec]; ok {
				cancel()
			}
		}
	}
}
