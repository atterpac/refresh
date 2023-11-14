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
}

func (engine *Engine) readConfigFile(path string) *Engine {
	if _, err := toml.DecodeFile(path, &engine); err != nil {
		fmt.Println("Error reading config file")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return engine
}

func (engine *Engine) verifyConfig() {
	engine.Log.Debug("Verifying Config")
	engine.Log.Debug(fmt.Sprintf("Config: %+v", engine))
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
}
