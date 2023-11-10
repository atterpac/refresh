package watcher

import (
	"fmt"
	"os/exec"
)
//TODO: Pipe stdout from process into the watch engine or the logs
func Reload(conf WatchEngine) int {
	killProcess(conf.Pid)
	Pid, err := startProcess(conf.ExecCommand, conf.RootPath)
	if err != nil {
		fmt.Println("Error starting process")
		return 0
	}
	return Pid
}

func killProcess(pid int) {
	// TODO: pkill PID so process can be restarted with new build version
}

func startProcess(args []string, dir string) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	err := cmd.Start()
	if err != nil {
		fmt.Println(cmd.Err)
		return 0, err
	}
	return cmd.Process.Pid, nil
}
