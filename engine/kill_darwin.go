//go:build darwin
package engine

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Process struct {
	Process *os.Process
}

// Kill spawned child process
func (engine *Engine) killProcess(process Process) bool {
	osProcess := process.Process
	slog.Info("Killing process", "pid", osProcess.Pid)
	// Windows requires special handling due to calls happening in "user mode" vs "kernel mode"
	// User mode doesnt allow for killing process so the work around currently is running taskkill command in cmd
	pgid, err := syscall.Getpgid(osProcess.Pid)
	if err != nil {
		slog.Error(fmt.Sprintf("Getting process group id: %s", err.Error()))
		return false
	}
	err = syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		slog.Error(fmt.Sprintf("Killing process: %s", err.Error()))
		return false
	}
	time.Sleep(250 * time.Millisecond)
	return true
}

func (engine *Engine) setPGID(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func removePGID(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
}
