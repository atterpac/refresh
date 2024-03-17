//go:build !windows

package engine

import (
	"log/slog"
	"syscall"
)

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
		slog.Info("Killing PGID", "pgid", p.pgid, "exec", p.Exec)
		err := syscall.Kill(-p.pgid, syscall.SIGKILL)
		if err != nil {
			slog.Debug("Process cannot be killed", "exec", p.Exec, "err", err)
		}
	}
}
