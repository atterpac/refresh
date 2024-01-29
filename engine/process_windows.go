//go:build windows

package engine

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/alexbrainman/ps"
	"github.com/pkg/errors"
)

type Process struct {
	Process *os.Process
	pid     int
	Output  bytes.Buffer
	Error   bytes.Buffer
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
	process.Process = cmd.Process
	process.pid = cmd.Process.Pid
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
	process.Output = cmd.Stdout
	process.Error = cmd.Stderr
	processErr := cmd.Start()
	if processErr != nil {
		slog.Error("Background Execute failed", "err", err)
		return Process{}, processErr
	}
	process.Process = cmd.Process
	process.pid = cmd.Process.Pid
	slog.Debug("Complete Exec Command", "cmd", runString)
	return process, nil
}

// Window specific kill process
func (engine *Engine) killProcess(process Process) bool {
	slog.Info("Killing PID", "pid", process.pid)
	err := taskKill(process.pid)
	return err == nil
}

func taskKill(pid int) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	err := kill.Run()
	if err != nil {
		slog.Error("Error killing process", "pid", pid, "err", err.Error())
		return err
	}
	slog.Debug("Process successfull killed", "pid", pid)
	return nil
}
