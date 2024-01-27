//go:build darwin
package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	Process *os.Process
}

func (engine *Engine) killProcess(process Process) bool {
	osProcess := process.Process
	slog.Debug("Killing process", "pid", osProcess.Pid)
	pgid, err := syscall.Getpgid(osProcess.Pid)
	if err != nil {
		slog.Error("Getting process group id", "err", err.Error())
		return false
	}
	err = syscall.Kill(-pgid, syscall.SIGKILL)
	if err != nil {
		slog.Error("Killing process", "err", err.Error())
		return false
	}
	return true
}

func (engine *Engine) spawnNewProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func (engine *Engine) setNewProcessGroup(cmd *exec.Cmd) {
	// Mac doesn't need to spawn a new process group after its been started
}
