//go:build windows
package engine

import (
	"os"
	"os/exec"
)

// Window specific kill process
func killProcess(process *os.Process) error {
	slog.Info("Killing process", "pid", process.Pid)
	// F = force kill | T = kill child processes in case users program spawned its own processes | PID = process id
	err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid)).Run()
	return err
}
