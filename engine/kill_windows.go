//go:build windows
package engine

import (
	"exec"
)

// Window specific kill process
func killProcess(pid int) error {
	// F = force kill | T = kill child processes in case users program spawned its own processes | PID = process id
	err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid)).Run()
	return err
}
