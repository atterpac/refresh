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
}

// Start process with exec command and a root path to call it in
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
	attachNewProcessGroup(cmd)
	err = cmd.Start()
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
	cmd.Stdout = &out
	cmd.Stderr = &err
	attachNewProcessGroup(cmd)
	cmdErr := cmd.Start()
	if cmdErr != nil {
		slog.Error("Background Execute failed", "err", err)
		return nil
	}
	process := cmd.Process
	slog.Debug("Complete Exec Command", "cmd", runString)
	return process
}

func (engine *Engine) killProcess(process Process) bool {
	osProcess := process.Process
	if osProcess == nil {
		return false
	}
	slog.Debug("Killing process", "pid", osProcess.Pid)
	pgid, err := syscall.Getpgid(osProcess.Pid)
	if err != nil {
		slog.Error("Getting process group id", "err", err.Error())
		return false
	}
	err = syscall.Kill(-pgid, syscall.SIGKILL)
	if err != nil {
		slog.Error("Killing process", "err", err.Error())
		return false
	}
	return true
}

func attachNewProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

