//go:build windows
package engine

import (
	"fmt"
	"os"
	"os/exec"
	"log/slog"
)

// Window specific kill process
func killProcess(process *os.Process) bool {
	slog.Info("Killing process", "pid", process.Pid)
	// F = force kill | T = kill child processes in case users program spawned its own processes | PID = process id
	err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid)).Run()
	return err == nil
}


func setPGID(cmd *exec.Cmd) {
	slog.Debug("Windows PGID Not Implemented")
}
