package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TODO: Pipe stdout from process into the watch engine or the logs
func Reload(engine Engine) *os.Process {
	// If there is a process already running kill it and run postexec command
	if engine.isRunning() {
		ok := releaseProcess(engine.Process)
		if !ok {
			engine.Log.Fatal("Error releasing process: %s")
			os.Exit(1)
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
	process, err := startProcess(generateExec(engine.Config.ExecCommand), engine.Config.RootPath)
	if err != nil {
		engine.Log.Fatal(fmt.Sprintf("Error starting process: %s", err.Error()))
		os.Exit(1)
	}
	return process
}

func releaseProcess(process *os.Process) bool {
	err := process.Kill()
	if err != nil {
		fmt.Println("Error killing process", err.Error())
		return false
	}
	return true
}

func startProcess(args []string, dir string) (*os.Process, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Println(cmd.Err)
		return nil, err
	}
	return cmd.Process, nil
}

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
func generateExec(cmd string) []string {
	// String split on spaces
	return strings.Split(cmd, " ")
}

func (engine *Engine) isRunning() bool {
	return engine.Process != nil
}
