package engine

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/process"
)

// TODO: Pipe stdout from process into the watch engine or the logs
func (engine *Engine) reloadProcess() *process.Process {
	// If there is a process already running kill it and run postexec command
	if engine.isRunning() {
		ok := killProcess(engine.Process)
		if !ok {
			slog.Error("Releasing process: %s")
			return nil
		}
		// Post Exec
		err := runFromString(engine.Config.PostExec, false)
		if err != nil {
			slog.Error(fmt.Sprintf("Running post-exec command: %s", err.Error()))
			os.Exit(1)
		}
		if engine.ProcessLogPipe != nil {
			slog.Debug("Closing log pipe")
			engine.ProcessLogPipe.Close()
			engine.ProcessLogPipe = nil
		}
	}
	// Pre-Process Exec
	err := runFromString(engine.Config.PreExec, engine.Config.PreWait)
	if err != nil {
		slog.Error(fmt.Sprintf("Running pre-exec command: %s", err.Error()))
		os.Exit(1)
	}
	// Start Exec Process
	process, err := engine.startProcess()
	if err != nil {
		slog.Error(fmt.Sprintf("Starting process: %s", err.Error()))
		os.Exit(1)
	}

	return process
}

// Start process with exec command and a root path to call it in
func (engine *Engine) startProcess() (*process.Process, error) {
	var err error
	command := generateExec(engine.Config.ExecCommand)
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = engine.Config.RootPath
	// If an external slog is provided do not pipe stdout to the engine
	if !engine.Config.externalSlog {
		cmd.Stderr = os.Stderr
		engine.ProcessLogPipe, err = cmd.StdoutPipe()
		if err != nil {
			slog.Error(fmt.Sprintf("Getting stdout pipe: %s", err.Error()))
			return nil, err
		}
		slog.Debug("Starting log pipe")
		go printSubProcess(engine.ProcessLogPipe)
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println(cmd.Err)
		return nil, err
	}
	process, err := process.NewProcess(int32(cmd.Process.Pid))
	if err != nil {
		slog.Error(fmt.Sprintf("Getting new process: %s", err.Error()))
		return nil, err
	}
	return process, nil
}

// Kill spawned child process
func killProcess(process *process.Process) bool {
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

// Takes a string and runs it as a command by sliceing the string on spaces and passing it to exec
func runFromString(cmdString string, wait bool) error {
	if cmdString == "" {
		return nil
	}
	commandSlice := generateExec(cmdString)
	cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	if wait {
		err = cmd.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

// Takes a string and splits it on spaces to create a slice of strings
func generateExec(cmd string) []string {
	// String split on spaces
	return strings.Split(cmd, " ")
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
