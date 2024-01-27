package engine

import (
	"fmt"
	"log/slog"
	"os"
)

func (engine *Engine) reloadProcess() {
	if len(engine.Config.ExecStruct) == 0 {
		slog.Error("No exec commands found")
		return
	}
	for _, ex := range engine.Config.ExecStruct {
		err := ex.run(engine)
		if err != nil {
			slog.Error("Running Execute: %s %e", ex.Cmd, err.Error())
		}
	}
}

// Check if a child process is running
func (engine *Engine) isRunning() bool {
	if engine.ProcessTree.Process == nil {
		return false
	}
	foundProcess, err := os.FindProcess(int(engine.ProcessTree.Process.Pid))
	if err != nil {
		slog.Error(fmt.Sprintf("Finding process: %s", err.Error()))
		return false
	}
	slog.Debug("Process running... attempting to kill", "pid", foundProcess.Pid)
	return err == nil
}


