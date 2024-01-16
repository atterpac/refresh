//go:build windows
package engine

import (
	"fmt"
	"os"
	"os/exec"
	"log/slog"
	"syscall"

	"github.com/alexbrainman/ps"
)

type Process struct {
	Process *os.Process
	JobObject *ps.JobObject
}

// Window specific kill process
func (engine *Engine) killProcess(process Process) bool {
	osProcess := process.Process
	slog.Info("Killing process", "pid", osProcess.Pid)
	err := engine.JobObject.Terminate(1)
	return err == nil
}


func (engine *Engine) setPGID(cmd *exec.Cmd) {
	var err error
	engine.JobObject, err = ps.CreateJobObject("refresh")
	if err != nil {
		slog.Error(fmt.Sprintf("Creating job object: %s", err.Error()))
	}
	handle := syscall.Handle(cmd.Process.Pid)
	err = engine.JobObject.AddProcess(handle)
	if err != nil {
		slog.Error(fmt.Sprintf("Adding process to job object: %s", err.Error()))
	}
}
