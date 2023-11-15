package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/process"
)

// TODO: Pipe stdout from process into the watch engine or the logs
func Reload(engine Engine) *process.Process {
	// If there is a process already running kill it and run postexec command
	if engine.isRunning() {
		ok := killProcess(engine.Process)
		if !ok {
			engine.Log.Fatal("Error releasing process: %s")
			return nil
		}
		// Post Exec
		err := RunFromString(engine.Config.PostExec)
		if err != nil {
			engine.Log.Fatal(fmt.Sprintf("Error running post-exec command: %s", err.Error()))
			os.Exit(1)
		}
	}
	// Pre-Process Exec
	err := RunFromString(engine.Config.PreExec)
	if err != nil {
		engine.Log.Fatal(fmt.Sprintf("Error running pre-exec command: %s", err.Error()))
		os.Exit(1)
	}
	// Start Exec Process
	process, err := startProcess(generateExec(engine.Config.ExecCommand), engine.Config.RootPath)
	if err != nil {
		engine.Log.Fatal(fmt.Sprintf("Error starting process: %s", err.Error()))
		os.Exit(1)
	}
	return process
}

// Kill spawned child process
func killProcess(process *process.Process) bool {
	// Windows requires special handling due to calls happening in "user mode" vs "kernel mode"
	// User mode doesnt allow for killing process so the work around currently is running taskkill command in cmd 
	if runtime.GOOS == "windows" {
		err := killWindows(int(process.Pid))
		if err != nil {
			fmt.Println("Error killing process", err.Error())
			return false
		}
		return true
	}
	// Kill process on other OS's
	err := process.Kill()
	if err != nil {
		fmt.Println("Error killing process", err.Error())
		return false
	}
	return true
}

// Start process with exec command and a root path to call it in
func startProcess(args []string, dir string) (*process.Process, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Println(cmd.Err)
		return nil, err
	}
	process, err := process.NewProcess(int32(cmd.Process.Pid))
	if err != nil {
		fmt.Println("Error getting process", err.Error())
		return nil, err
	}
	return process, nil
}

// Takes a string and runs it as a command by sliceing the string on spaces and passing it to exec
func RunFromString(cmdString string) error {
	if cmdString == "" {
		return nil
	}
	commandSlice := generateExec(cmdString)
	cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
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
