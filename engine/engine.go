package engine

import (
	"io"
	"log/slog"
	"os"

	"github.com/shirou/gopsutil/process"
)

type Engine struct {
	Process        *process.Process
	Active         bool
	Config         Config          `toml:"config"`
	ProcessLogFile *os.File
	ProcessLogPipe io.ReadCloser
}

func (engine *Engine) Start() {
	if engine.Config.Slog == nil {
		engine.Config.Slog = newLogger(engine.Config.LogLevel)
		engine.Config.ExternalSlog = false
	} else {
		engine.Config.ExternalSlog = true
	}
	slog.SetDefault(engine.Config.Slog)
	engine.watch()
}

func NewEngine(rootPath, execCommand, label, logLevel string, ignore Ignore, debounce int, chunkSize string) *Engine {
	engine := &Engine{}
	engine.Config = Config{
		RootPath:    rootPath,
		ExecCommand: execCommand,
		Label:       label,
		LogLevel:    logLevel,
		Ignore:      ignore,
		Debounce:    debounce,
	}
	engine.verifyConfig()
	return engine
}

func NewEngineFromConfig(options Config) *Engine {
	engine := &Engine{}
	engine.Config = options
	engine.verifyConfig()
	return engine
}

func NewEngineFromTOML(confPath string) *Engine {
	engine := Engine{}
	engine.readConfigFile(confPath)
	engine.verifyConfig()
	return &engine
}


