//go:build linux || darwin

package process

import (
	"os/exec"
	"syscall"
)

// shellInvocation returns the shell and arguments used to run a command string,
// so commands may use shell features (quoting, pipes, &&, redirection).
func shellInvocation(command string) (string, []string) {
	return "/bin/sh", []string{"-c", command}
}

// setProcessGroup puts the command in its own process group so the entire tree
// (the child and anything it spawns) can be signalled together.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree force-kills the command's whole process group, falling back to
// the direct process if the group id can't be resolved.
func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
		return syscall.Kill(-pgid, syscall.SIGKILL)
	}
	return cmd.Process.Kill()
}
