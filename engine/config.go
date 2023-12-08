package engine

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	RootPath       string   `toml:"root_path"`
	BackgroundExec string   `toml:"background_exec"`
	Ignore         Ignore   `toml:"ignore"`
	ExecList       []string `toml:"exec_lifecycle"`
	ExecStruct     []Execute
	ignoreMap      ignoreMap
	LogLevel       string `toml:"log_level"`
	Debounce       int    `toml:"debounce"`
	Callback       func(*EventCallback) EventHandle
	Slog           *slog.Logger
	externalSlog   bool
}

// Reads a config.toml file and returns the engine
func (engine *Engine) readConfigFile(path string) *Engine {
	if _, err := toml.DecodeFile(path, &engine); err != nil {
		slog.Error("Error reading config file")
		slog.Error(err.Error())
		os.Exit(1)
	}
	return engine
}

// Verify required data is present in config
func (engine *Engine) verifyConfig() {
	slog.Debug("Verifying Config")
	if engine.Config.RootPath == "" {
		slog.Error("Required Root Path is not set")
		os.Exit(1)
	}
	engine.verifyExecute()
	slog.Debug("Config Verified")
	// Change directory executes are called in to match root directory
	changeWorkingDirectory(engine.Config.RootPath)
}

// Verify execute structs
func (engine *Engine) verifyExecute() {
	var primary bool
	if engine.Config.ExecList == nil && engine.Config.ExecStruct == nil {
		slog.Error("Execute list or struct's must be provided in the refresh config")
		os.Exit(1)
	}
	if engine.Config.ExecList == nil {
		for _, exe := range engine.Config.ExecStruct {
			if exe.IsPrimary {
				if primary {
					slog.Error("Only one primary function can be set")
					os.Exit(1)
				}
				primary = true
			}
		}
	} else {
		var kill bool
		var refresh bool 
		for _, exe := range engine.Config.ExecList {
			switch exe {
			case "REFRESH":
				refresh = true
			case "KILL_STALE": 
				kill = true
			default: 
				continue
			}
		}
		if !kill && !refresh {
			slog.Error("Execute List must contain `KILL_STALE` and `REFRESH`")
			os.Exit(1)
		}
		if !kill {
			slog.Error("Execute list must contain `KILL_STALE`")
			os.Exit(1)
		}
		if !refresh {
			slog.Error("Execut list must contain `REFRESH`")
			os.Exit(1)
		}
	}
}


func readGitIgnore(path string) map[string]struct{} {
	file, err := os.Open(path + "/.gitignore")
	if err != nil {
		return nil
	}
	defer file.Close()
	slog.Debug("Reading .gitignore")
	scanner := bufio.NewScanner(file)
	var linesMap = make(map[string]struct{})
	for scanner.Scan() {
		// Check if line is a comment
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		// Check if line is empty
		if len(scanner.Text()) == 0 {
			continue
		}

		line := scanner.Text()
		// Check if line does not start with '*'
		if !strings.HasPrefix(line, "*") {
			// Add asterisk to the beginning of line
			line = "*" + line
		}
		// Add to the map
		linesMap[line] = struct{}{}
	}
	slog.Debug(fmt.Sprintf("Read %v lines from .gitignore", linesMap))
	return linesMap
}

func changeWorkingDirectory(path string) {
	cleaned := strings.TrimPrefix(path, ".")
	cleaned = strings.TrimPrefix(cleaned, "/")
	cleaned = strings.TrimPrefix(cleaned, `\`) // Windows >:(
	wd, err := os.Getwd()
	if err != nil {
		slog.Error("Getting Working Directory")
	}
	err = os.Chdir(wd + "/" + cleaned)
	if err != nil {
		slog.Error("Setting new directory", "dir", path)
	}
}
