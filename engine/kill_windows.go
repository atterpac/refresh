//go:build windows
package engine

import (
	"fmt"
	"os"
	"os/exec"
	"log/slog"
	"syscall"
	"time"

	"github.com/alexbrainman/ps"
)

type Process struct {
	Process *os.Process
	JobObject *ps.JobObject
}

const PROCESS_ALL_ACCESS = 0x1F0FFF

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	handle = kernel32.NewProc("OpenProcess")
	openProcess = kernel32.NewProc("OpenProcess")
	closeHandle = kernel32.NewProc("CloseHandle")
)

// Window specific kill process
func (engine *Engine) killProcess(process Process) bool {
	osProcess := process.Process
	slog.Info("Killing process", "pid", osProcess.Pid)
	err := engine.ProcessTree.JobObject.Terminate(1)
	time.Sleep(250 * time.Millisecond)
	return err == nil
}

func openProcessHandle(pid int) (syscall.Handle, error) {
	handle, _, err := openProcess.Call(
		uintptr(PROCESS_ALL_ACCESS),
		0,
		uintptr(pid),
	)

	if handle == 0 {
		return syscall.InvalidHandle, err
	}
	return syscall.Handle(handle), nil
}

func (engine *Engine) setPGID(cmd *exec.Cmd) {
	var err error
	if cmd == nil {
		slog.Error("No command found")
		return
	}
	pid := cmd.Process.Pid
	slog.Info("Setting PGID", "pid", pid)
	engine.ProcessTree.JobObject, err = ps.CreateJobObject("refresh")
	if err != nil {
		slog.Error(fmt.Sprintf("Creating job object: %s", err.Error()))
	}
	handle, err := openProcessHandle(pid)
	if err != nil {
		slog.Error(fmt.Sprintf("Opening process handle: %s", err.Error()))
	}
	defer syscall.CloseHandle(handle)
	err = engine.ProcessTree.JobObject.AddProcess(handle)
	if err != nil {
		slog.Error(fmt.Sprintf("Adding process to job object: %s", err.Error()))
	}
}
