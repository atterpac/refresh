//go:build windows
package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"log/slog"
	"time"
	"syscall"

	"github.com/alexbrainman/ps"
)

type Process struct {
	Process *os.Process
	JobObject *ps.JobObject
}

func (engine *Engine) startPrimaryProcess(runString string) (*os.Process, error) {
	var err error
	slog.Debug("Starting Primary")
	cmd := generateExec(runString)
	//If an external slog is provided do not pipe stdout to the engine
	if !engine.Config.externalSlog {
		cmd.Stderr = os.Stderr
		engine.ProcessLogPipe, err = cmd.StdoutPipe()
		if err != nil {
			slog.Error("Getting log pipe", "err", err.Error())
			return nil, err
		}
	}
	err = cmd.Start()
	engine.createJobObject(cmd)
	if err != nil {
		slog.Error("Starting Primary", "err", err.Error())
		return nil, err
	}
	slog.Debug("Starting log pipe")
	go printSubProcess(engine.ProcessLogPipe)
	if err != nil {
		slog.Error("Starting Primary", "err", err.Error())
		return nil, err
	}
	return cmd.Process, nil
}

func (engine *Engine) startBackgroundProcess(runString string) *os.Process {
	cmd := generateExec(runString)
	var out, err bytes.Buffer
	// Let Process run in background
	cmd.Stdout = &out
	cmd.Stderr = &err
	processErr := cmd.Start()
	if processErr != nil {
		slog.Error("Background Execute failed", "err", err)
		return nil
	}
	engine.createJobObject(cmd)
	process := cmd.Process
	slog.Debug("Complete Exec Command", "cmd", runString)
	return process
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
	slog.Info("Killing Windows Job Object")
	err := engine.ProcessTree.JobObject.Terminate(1)
	time.Sleep(500 * time.Millisecond)
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
//
// func (engine *Engine) spawnNewProcessGroup(cmd *exec.Cmd) {
// 	// Windows needs to spawn a new process group after its been started
// }

func (engine *Engine) createJobObject(cmd *exec.Cmd) {
	var err error
	if cmd.Process == nil {
		slog.Error("Process is nil")
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
	err = engine.ProcessTree.JobObject.AddProcess(handle)
	if err != nil {
		slog.Error(fmt.Sprintf("Adding process to job object: %s", err.Error()))
	}
	syscall.CloseHandle(handle)
}

