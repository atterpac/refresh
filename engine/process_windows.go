//go:build windows

package engine

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/alexbrainman/ps"
	"github.com/pkg/errors"
)

type Process struct {
	Process   *os.Process
	JobObject *ps.JobObject
}

func (engine *Engine) startPrimaryProcess(runString string) (Process, error) {
	var err error
	var process Process
	slog.Debug("Starting Primary")
	cmd := generateExec(runString)
	//If an external slog is provided do not pipe stdout to the engine
	if !engine.Config.externalSlog {
		cmd.Stderr = os.Stderr
		engine.ProcessLogPipe, err = cmd.StdoutPipe()
		if err != nil {
			slog.Error("Getting log pipe", "err", err.Error())
			return process, err
		}
	}
	err = cmd.Start()
	if err != nil {
		slog.Error("Starting Primary", "err", err.Error())
		return process, err
	}
	process.JobObject, err = createJobObject(cmd)
	process.Process = cmd.Process
	slog.Debug("Starting log pipe")
	go printSubProcess(engine.ProcessLogPipe)
	if err != nil {
		slog.Error("Starting Primary", "err", err.Error())
		return process, err
	}
	return process, nil
}

func (engine *Engine) startBackgroundProcess(runString string) (Process, error) {
	var process Process
	var err error
	cmd := generateExec(runString)
	var out, bufErr bytes.Buffer
	// Let Process run in background
	cmd.Stdout = &out
	cmd.Stderr = &bufErr
	processErr := cmd.Start()
	if processErr != nil {
		slog.Error("Background Execute failed", "err", err)
		return Process{}, processErr
	}
	process.JobObject, err = createJobObject(cmd)
	process.Process = cmd.Process
	slog.Debug("Complete Exec Command", "cmd", runString)
	return process, nil
}

const PROCESS_ALL_ACCESS = 0x1F0FFF

var (
	kernel32    = syscall.NewLazyDLL("kernel32.dll")
	handle      = kernel32.NewProc("OpenProcess")
	openProcess = kernel32.NewProc("OpenProcess")
	closeHandle = kernel32.NewProc("CloseHandle")
)

// Window specific kill process
func (engine *Engine) killProcess(process Process) bool {
	slog.Info("Killing Windows Job Object")
	err := process.JobObject.Terminate(1)
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

func createJobObject(cmd *exec.Cmd) (*ps.JobObject, error) {
	var err error
	if cmd.Process == nil {
		return nil, errors.New("Process is nil")
	}
	job, err := ps.CreateJobObject("refresh")
	if err != nil {
		slog.Error(fmt.Sprintf("Creating job object: %s", err.Error()))
	}
	handle, err := openProcessHandle(cmd.Process.Pid)
	if err != nil {
		slog.Error(fmt.Sprintf("Opening process handle: %s", err.Error()))
		return nil, err
	}
	err = job.AddProcess(handle)
	if err != nil {
		slog.Error(fmt.Sprintf("Adding process to job object: %s", err.Error()))
		return nil, err
	}
	syscall.CloseHandle(handle)
	return job, nil
}
