package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

type Execute struct {
	Cmd        string      `toml:"cmd" yaml:"cmd"`               // Execute command
	ChangeDir  string      `toml:"dir" yaml:"dir"`               // If directory needs to be changed to call this command relative to the root path
	IsBlocking bool        `toml:"blocking" yaml:"blocking"`     // Should the following executes wait for this one to complete
	IsPrimary  bool        `toml:"primary" yaml:"primary"`       // Only one primary command can be run at a time
	DelayNext  int         `toml:"delay_next" yaml:"delay_next"` // Delay in milliseconds before running command
	process    *os.Process // Stores the Exec.Start() process
}

var KILL_STALE = Execute{
	Cmd:        "KILL_STALE",
	IsBlocking: true,
	IsPrimary:  false,
}

var REFRESH_EXEC = "REFRESH"
var KILL_EXEC = "KILL_STALE"
var firstRun = true

// func (ex *Execute) run(engine *Engine) error {
// 	slog.Info("Running Execute", "command", ex.Cmd)
// 	var err error
// 	var restoreDir string = ""
// 	if ex.Cmd == "" {
// 		return nil
// 	}
// 	if ex.ChangeDir != "" {
// 		restoreDir, err = os.Getwd()
// 		slog.Debug("Change Directory Set", "dir", restoreDir)
// 		if err != nil {
// 			slog.Error("Getting working directory")
// 		}
// 		changeWorkingDirectory(ex.ChangeDir)
// 	}
// 	if ex.IsPrimary {
// 		err := engine.kill()
// 		if err != nil {
// 			slog.Error("Killing Stale Process", "err", err.Error())
// 		}
// 		slog.Debug("Reloading Process")
// 		time.Sleep(500 * time.Millisecond)
// 		processKey, err := engine.startPrimaryProcess(ex.Cmd)
// 		if err != nil {
// 			slog.Error("Starting Run command", "err", err, "command", ex.Cmd)
// 			return err
// 		}
// 		process, ok := engine.ProcessMap[processKey]
// 		if !ok {
// 			slog.Error("Primary Process Failed to start")
// 			return errors.New("Primary Process Failed to start")
// 		}
// 		slog.Info("Primary Process Started", "pid", process.Process.Pid)
// 		if restoreDir != "" {
// 			slog.Info("Restoring working Dir")
// 			changeWorkingDirectory(restoreDir)
// 		}
// 		slog.Error("Primary Process Failed to Start")
// 	}
// 	switch ex.Cmd {
// 	case "":
// 		return nil
// 	case "KILL_STALE":
// 		slog.Debug("Kill_STALE depreciated primary will be killed prior to next run")
// 	default:
// 		ex.process, err = execFromString(ex.Cmd, ex.IsBlocking)
// 		if err != nil {
// 			slog.Error("Running Execute", "command", ex.Cmd, "error", err.Error())
// 		}
// 		slog.Debug("Complete Exec Command", "cmd", ex.Cmd)
// 	}
// 	if restoreDir != "" {
// 		slog.Info("Restoring working Dir")
// 		changeWorkingDirectory(restoreDir)
// 	}
// 	return nil
// }

func (engine *Engine) kill() error {
	if firstRun {
		firstRun = false
		return nil
	}
	engine.ProcessManager.KillProcesses(true)
	return nil
}

func execFromString(runString string, block bool) (*os.Process, error) {
	cmd := generateExec(runString)
	// Let Process run in background
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	if block {
		err := cmd.Wait()
		if err != nil {
			slog.Error("Running Execute", "command", runString)
			return nil, err
		}
	}
	return cmd.Process, nil
}

// Takes a string and splits it on spaces to create a slice of strings
func generateExec(cmd string) *exec.Cmd {
	slice := strings.Split(cmd, " ")
	cmdEx := exec.Command(slice[0], slice[1:]...)
	return cmdEx
}

func (e *Engine) generateProcess() {
	e.ProcessManager.AddProcess(e.Config.BackgroundStruct.Cmd, e.Config.BackgroundStruct.IsBlocking, e.Config.BackgroundStruct.IsPrimary, true)
	for _, ex := range e.Config.ExecStruct {
		e.ProcessManager.AddProcess(ex.Cmd, ex.IsBlocking, ex.IsPrimary, false)
	}
}

