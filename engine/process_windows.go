//go:build windows

package engine

import (
	"log/slog"
	"os/exec"
	"strconv"
)

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
