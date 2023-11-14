package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//TODO: Pipe stdout from process into the watch engine or the logs
func Reload(engine Engine) *os.Process {
	ok := releaseProcess(engine.Process)
	if !ok {
		engine.Log.Fatal("Error releasing process")
		os.Exit(1)
	}

	cmd := generateExec(engine.Config.ExecCommand)
	process, err := startProcess(cmd, engine.Config.RootPath)
	if err != nil {
		fmt.Println("Error starting process")
		engine.Log.Fatal(err.Error())
		os.Exit(1)
	}
	return process
}


func releaseProcess(process *os.Process) bool {
	if process != nil {	
		err := process.Kill()
		if err != nil {
			fmt.Println("Error killing process", err.Error())
			return false
		}
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

func generateExec(cmd string) []string {
	// String split on spaces
	return strings.Split(cmd, " ")
}
