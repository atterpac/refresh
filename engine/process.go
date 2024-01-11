package engine

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
)

func (engine *Engine) reloadProcess() {
	if engine.Config.ExecList == nil && engine.Config.ExecStruct == nil {
		slog.Error("No exec commands found")
		return
	}
	if engine.Config.ExecStruct != nil {
		engine.reloadFromStruct()
		return
	}
	err := engine.execFromList()
	if err != nil {
		slog.Error(fmt.Sprintf("Running from exec list: %s", err))
	}

}

func (engine *Engine) reloadFromStruct() {
	for _, ex := range engine.Config.ExecStruct {
		err := ex.run(engine)
		if err != nil {
			slog.Error("Running Execute: %s %e", ex.Cmd, err.Error())
		}
	}
}

// Start process with exec command and a root path to call it in
func (engine *Engine) startPrimary(runString string) (*os.Process, error) {
	var err error
	slog.Debug("Starting Primary")
	cmdExec := generateExec(runString)
	cmd := exec.Command(cmdExec[0], cmdExec[1:]...)
	// If an external slog is provided do not pipe stdout to the engine
	if !engine.Config.externalSlog {
		cmd.Stderr = os.Stderr
		engine.ProcessLogPipe, err = cmd.StdoutPipe()
		if err != nil {
			slog.Error(fmt.Sprintf("Getting stdout pipe: %s", err.Error()))
			return nil, err
		}
	}
	err = cmd.Start()
	slog.Debug("Starting log pipe")
	go printSubProcess(engine.ProcessLogPipe)
	if err != nil {
		fmt.Println(cmd.Err)
		return nil, err
	}
	if err != nil {
		slog.Error(fmt.Sprintf("Getting new process: %s", err.Error()))
		return nil, err
	}
	return cmd.Process, nil
}

// Kill spawned child process
func killProcess(process *os.Process) bool {
	// Windows requires special handling due to calls happening in "user mode" vs "kernel mode"
	// User mode doesnt allow for killing process so the work around currently is running taskkill command in cmd
	if runtime.GOOS == "windows" {
		err := killWindows(int(process.Pid))
		if err != nil {
			slog.Error(fmt.Sprintf("Killing process: %s", err.Error()))
			return false
		}
		return true
	}
	// Kill process on other OS's
	err := process.Kill()
	if err != nil {
		slog.Error(fmt.Sprintf("Killing process: %s", err.Error()))
		return false
	}
	return true
}

// Check if a child process is running
func (engine *Engine) isRunning() bool {
	if engine.Process == nil {
		return false
	}
	_, err := os.FindProcess(int(engine.Process.Pid))
	return err == nil
}

// Window specific kill process
func killWindows(pid int) error {
	// F = force kill | T = kill child processes in case users program spawned its own processes | PID = process id
	err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid)).Run()
	return err
}
