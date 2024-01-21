package engine

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
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

func (ex *Execute) run(engine *Engine) error {
	slog.Info("Running Execute", "command", ex.Cmd)
	var err error
	var restoreDir string = ""
	if ex.Cmd == "" {
		return nil
	}
	if ex.ChangeDir != "" {
		restoreDir, err = os.Getwd()
		slog.Debug("Change Directory Set", "WD", restoreDir)
		if err != nil {
			slog.Error("Getting working directory")
		}
		changeWorkingDirectory(ex.ChangeDir)
	}
	if ex.IsPrimary {
		slog.Debug("Reloading Process")
		time.Sleep(500 * time.Millisecond)
		engine.ProcessTree.Process, err = engine.startPrimary(ex.Cmd)
		if err != nil {
			slog.Error("Starting Run command", err, "command", ex.Cmd)
			return err
		}
		if engine.ProcessTree.Process != nil {
			slog.Info("Primary Process Started", "pid", engine.ProcessTree.Process.Pid)
			if restoreDir != "" {
				slog.Info("Restoring working Dir")
				changeWorkingDirectory(restoreDir)
			}
			return nil
		}
	}
	switch ex.Cmd {
	case "":
		return nil
	case "KILL_STALE":
		if firstRun {
			firstRun = false
			return nil
		}
		if engine.isRunning() {
			slog.Debug("Killing Stale Version")
			ok := engine.killProcess(engine.ProcessTree)
			if !ok {
				slog.Error("Releasing stale process")
			}
			if engine.ProcessLogPipe != nil {
				slog.Debug("Closing log pipe")
				engine.ProcessLogPipe.Close()
				engine.ProcessLogPipe = nil
			}
		}
	default:
		err := execFromString(ex.Cmd, ex.IsBlocking)
		if err != nil {
			slog.Error("Running Execute", "command", ex.Cmd, "error", err.Error())
		}
	}
	if restoreDir != "" {
		slog.Info("Restoring working Dir")
		changeWorkingDirectory(restoreDir)
	}
	return nil
}

func backgroundExec(runString string) {
	commandSlice := generateExec(runString)
	cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	var out, err bytes.Buffer
	// Let Process run in background
	cmd.Stdout = &out
	cmd.Stderr = &err
	removePGID(cmd)
	cmd.Start()
	slog.Debug(fmt.Sprintf("Complete Exec Command: %s", runString))
}

func execFromString(runString string, block bool) error {
	commandSlice := generateExec(runString)
	cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	// Let Process run in background
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	if block {
		err := cmd.Wait()
		if err != nil {
			slog.Error("Running Execute", "command", runString)
			return err
		}
	}
	return nil
}

// Takes a string and splits it on spaces to create a slice of strings
func generateExec(cmd string) []string {
	return strings.Split(cmd, " ")
}
