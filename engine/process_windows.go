//go:build windows

package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func (e *Engine) StartProcesses() {
	pm := e.ProcessManager
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.processes) == 0 {
		slog.Warn("No Processes to Start")
		return
	}
	slog.Warn("Starting Processes", "count", len(pm.processes))
	for _, p := range pm.processes {
		if p.Exec == "KILL_STALE" {
			continue
		}
		if !firstRun && p.Background {
			slog.Debug("Leaving background process running", "exec", p.Exec)
			continue
		}
		slog.Debug("Starting Process", "exec", p.Exec, "blocking", p.Blocking, "primary", p.Primary, "background", p.Background, "firstRun", firstRun)
		cmd := generateExec(p.Exec)
		p.cmd = cmd
		if p.Primary {
			if !firstRun {
				pm.KillProcesses(true)
				slog.Debug("Processes killed")
				time.Sleep(100 * time.Millisecond)
			}
			if !e.Config.externalSlog {
				cmd.Stderr = os.Stderr
				e.ProcessLogPipe, _ = cmd.StdoutPipe()
				go printSubProcess(e.ProcessLogPipe)
			}
		}
		var err error
		if p.Blocking {
			err = cmd.Run()
		} else {
			err = cmd.Start()
			go func() {
				cmd.Wait()
			}()
		}
		if err != nil {
			slog.Error("Running Command", "exec", p.Exec, "err", err)
		}
	}
}

// Window specific kill process
func (pm *ProcessManager) KillProcesses(ignoreBackground bool) {
	slog.Debug("Killing Processes")
	for _, p := range pm.processes {
		if p.Background && ignoreBackground {
			slog.Debug("Ignoring background process", "exec", p.Exec)
			continue
		}
		if p.pgid == 0 {
			continue
		}
		err := taskKill(p.cmd.Process.Pid)
		if err != nil {
			slog.Error("Error killing process", "pid", p.cmd.Process.Pid, "err", err.Error())
		}
	}
}

func taskKill(pid int) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	err := kill.Run()
	if err != nil {
		slog.Error("Error killing process", "pid", pid, "err", err.Error())
		return err
	}
	slog.Debug("Process successfull killed", "pid", pid)
	return nil
}
