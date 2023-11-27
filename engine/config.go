package engine

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	RootPath       string `toml:"root_path"`
	BackgroundExec string `toml:"background_exec"`
	PreBuild       string `toml:"pre_build"`
	ExecBuild      string `toml:"build"` // Runs between a reload trigger and killing the old process
	PostBuild      string `toml:"post_build"`
	PreRun         string `toml:"pre_run"`
	ExecRun        string `toml:"run_exec"` // Runs after a the old process has been stopped and after prerun
	PostRun        string `toml:"post_run"`
	CleanupExec    string `toml:"cleanup"` // Runs when an old process has been killed and before the new one starts
	PostExec       string `toml:"post_exec"`
	Ignore         Ignore `toml:"ignore"`
	LogLevel       string `toml:"log_level"`
	Debounce       int    `toml:"debounce"`
	Callback       func(*EventCallback) EventHandle
	Slog           *slog.Logger
	ExternalSlog   bool
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
		slog.Error("ERROR: Root Path not set")
		os.Exit(1)
	}
	if engine.Config.ExecRun == "" {
		slog.Error("ERROR: Exec Command not set")
		os.Exit(1)
	}
	slog.Debug("Config Verified")
}
