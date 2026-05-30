//go:build !linux && !darwin && !windows

package process

import "os/exec"

// shellInvocation falls back to /bin/sh on platforms without a known shell.
func shellInvocation(command string) (string, []string) {
	return "/bin/sh", []string{"-c", command}
}

// setProcessGroup is a no-op on platforms without process-group support.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessTree falls back to killing only the direct process.
func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
