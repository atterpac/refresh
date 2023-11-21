package engine

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	RootPath    string `toml:"root_path"`
	PreExec     string `toml:"pre_exec"`
	ExecCommand string `toml:"exec_command"`
	PostExec    string `toml:"post_exec"`
	// Ignore uses a custom unmarshaler see ignore.go
	Ignore       Ignore `toml:"ignore"`
	LogLevel     string `toml:"log_level"`
	Debounce     int    `toml:"debounce"`
	Callback     func(*EventCallback) (EventHandle)
	Slog         *slog.Logger
	ExternalSlog bool
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
	if engine.Config.ExecCommand == "" {
		slog.Error("ERROR: Exec Command not set")
		os.Exit(1)
	}
	slog.Debug("Config Verified")
}
