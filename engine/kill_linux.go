//go:build linux

package engine

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Kill spawned child process
func killProcess(process *os.Process) bool {
	slog.Info("Killing process", "pid", process.Pid)
	// Windows requires special handling due to calls happening in "user mode" vs "kernel mode"
	// User mode doesnt allow for killing process so the work around currently is running taskkill command in cmd
	pgid, err := syscall.Getpgid(process.Pid)
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

func setPGID(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
