package watcher

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	IsFile      bool   `toml:"-"`
	Label       string `toml:"label"`
	RootPath    string `toml:"root_path"`
	PreExec     string `toml:"pre_exec"`
	ExecCommand string `toml:"exec_command"`
	PostExec    string `toml:"post_exec"`
	Ignore      Ignore `toml:"ignore"`
	LogLevel    string `toml:"log_level"`
	Debounce    int    `toml:"debounce"`
	LogChunk    int    `toml:"log_chunk"`
}

// Reads a config.toml file and returns the engine
func (engine *Engine) readConfigFile(path string) *Engine {
	if _, err := toml.DecodeFile(path, &engine); err != nil {
		fmt.Println("Error reading config file")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	engine.Config.IsFile = true

	return engine
}

// Verify required data is present in config
func (engine *Engine) verifyConfig() {
	engine.Log.Debug("Verifying Config")
	if engine.Config.RootPath == "" {
		engine.Log.Fatal("ERROR: Root Path not set")
		os.Exit(1)
	}
	if engine.Config.ExecCommand == "" {
		engine.Log.Fatal("ERROR: Exec Command not set")
		os.Exit(1)
	}
	if engine.Config.Label == "" {
		engine.Log.Warn("Label not set")
	}
	engine.Log.Debug("Config Verified")
}
