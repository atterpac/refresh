package engine

import (
)

func (engine *Engine) reloadProcess() {
	engine.ProcessManager.StartProcesses()
}

// Check if a child process is running
// func (engine *Engine) isRunning() bool {
// 	if engine.PrimaryProcess.Process == nil {
// 		return false
// 	}
// 	foundProcess, err := os.FindProcess(int(engine.PrimaryProcess.Process.Pid))
// 	if err != nil {
// 		slog.Error(fmt.Sprintf("Finding process: %s", err.Error()))
// 		return false
// 	}
// 	slog.Debug("Process running... attempting to kill", "pid", foundProcess.Pid)
// 	return err == nil
// }


