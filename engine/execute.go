package engine

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

type Execute struct {
	Cmd        string
	IsBlocking bool
	IsPrimary  bool // Only one primary command can be run at a time
	process    *os.Process
}

var KILL_STALE = Execute{
	Cmd:        "KILL_STALE",
	IsBlocking: true,
	IsPrimary:  false,
}

var REFRESH_EXEC = "REFRESH"
var KILL_EXEC = "KILL_STALE"

func (ex *Execute) execute(engine *Engine) error {
	var err error
	if ex.Cmd == "" {
		return nil
	}
	if ex.IsPrimary {
		slog.Debug("Reloading Process")
		engine.Process, err = engine.startPrimary(ex.Cmd)
		if err != nil {
			slog.Error(fmt.Sprintf("Starting Run command: %s", err.Error()))
			os.Exit(1)
		}
		slog.Debug("Sucessfull refresh")
		return nil
	}
	switch ex.Cmd {
	case "":
		return nil
	case "KILL_STALE":
		slog.Debug("No process found to kill")
		if engine.isRunning() {
			slog.Debug("Killing Stale Version")
			ok := killProcess(engine.Process)
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
		slog.Debug(fmt.Sprintf("Running Exec Command: %s", ex.Cmd))
		commandSlice := generateExec(ex.Cmd)
		cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return err
		}
		ex.process = cmd.Process
		if ex.IsBlocking {
			err = cmd.Wait()
			if err != nil {
				return err
			}
		}
		slog.Debug(fmt.Sprintf("Complete Exec Command: %s", ex.Cmd))
	}
	return nil
}

func (engine *Engine) execFromList() error {
	var nextPrimary bool
	var err error
	if engine.Config.ExecList == nil {
		return nil
	}
	for _, exe := range engine.Config.ExecList {
		if nextPrimary {
			slog.Debug("Reloading Process")
			engine.Process, err = engine.startPrimary(exe)
			if err != nil {
				slog.Error(fmt.Sprintf("Starting Run command: %s", err.Error()))
				os.Exit(1)
			}
			slog.Debug("Sucessfull refresh")
			nextPrimary = false
			return nil
		}
		switch exe {
		case "":
			return nil
		case "REFRESH":
			nextPrimary = true
		case "KILL_STALE":
			slog.Debug("No process found to kill")
			if engine.isRunning() {
				slog.Debug("Killing Stale Version")
				ok := killProcess(engine.Process)
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
			slog.Debug(fmt.Sprintf("Running Exec Command: %s", exe))
			commandSlice := generateExec(exe)
			cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				return err
			}
			err = cmd.Wait()
			if err != nil {
				slog.Error("Running Execute: %s", exe)
			}
			slog.Debug(fmt.Sprintf("Complete Exec Command: %s", exe))
		}
		return nil
	}
	return nil
}

func execFromString(runString string) error {
	if runString == "" {
		return nil
	}
	commandSlice := generateExec(runString)
	cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Takes a string and splits it on spaces to create a slice of strings
func generateExec(cmd string) []string {
	return strings.Split(cmd, " ")
}
