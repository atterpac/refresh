//go:build windows

package process

import (
	"os/exec"
	"strconv"
)

// setProcessGroup is a no-op on Windows; process-tree termination is handled by
// taskkill /T in killProcessTree.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessTree force-kills the command and all of its child processes.
func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
