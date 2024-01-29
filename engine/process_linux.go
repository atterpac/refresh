//go:build linux

package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"bytes"
)

type Process struct {
	Process *os.Process
	pgid int
	Output  bytes.Buffer
	Error   bytes.Buffer
}

// Start process with exec command and a root path to call it in
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
	attachNewProcessGroup(cmd)
	err = cmd.Start()
	if err != nil {
		slog.Error("Starting Primary", "err", err.Error())
		return process, err
	}
	slog.Debug("Starting log pipe")
	go printSubProcess(engine.ProcessLogPipe)
	if err != nil {
		slog.Error("Starting Primary", "err", err.Error())
		return process, err
	}
	process.Process = cmd.Process
	process.pgid, err = syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		slog.Error("Getting process group id", "err", err.Error())
		return process, err
	}
	return process, nil
}

func (engine *Engine) startBackgroundProcess(runString string) (Process, error) {
	var err error
	var process Process
	cmd := generateExec(runString)
	// Let Process run in background
	cmd.Stdout = &process.Output
	cmd.Stderr = &process.Error
	attachNewProcessGroup(cmd)
	cmdErr := cmd.Start()
	if cmdErr != nil {
		slog.Error("Background Execute failed", "err", cmdErr)
		return Process{}, cmdErr
	}
	slog.Debug("Complete Exec Command", "cmd", runString)
	// Get PGID
	process.pgid, err = syscall.Getpgid(cmd.Process.Pid)
	process.Process = cmd.Process
	if err != nil {
		slog.Error("Getting process group id", "err", err.Error())
		return Process{}, err
	}
	slog.Debug("Process group id", "pgid", process.pgid)
	return process, nil
}

func (engine *Engine) killProcess(process Process) bool {
	osProcess := process.Process
	if osProcess == nil {
		return false
	}
	slog.Debug("Killing process")
	err := syscall.Kill(-process.pgid, syscall.SIGKILL)
	if err != nil {
		slog.Error("Killing process", "err", err.Error())
		return false
	}
	return true
}

func attachNewProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

