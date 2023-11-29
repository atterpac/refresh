package engine

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/rjeczalik/notify"
	"github.com/shirou/gopsutil/process"
)

type Engine struct {
	Process        *process.Process
	Chan           chan notify.EventInfo
	Active         bool
	Config         Config `toml:"config"`
	ProcessLogFile *os.File
	ProcessLogPipe io.ReadCloser
}

func (engine *Engine) Start() {
	if engine.Config.Slog == nil {
		engine.Config.Slog = newLogger(engine.Config.LogLevel)
		engine.Config.externalSlog = false
	} else {
		engine.Config.externalSlog = true
	}
	slog.SetDefault(engine.Config.Slog)
	slog.Info("Refresh Start")
	if engine.Config.Ignore.IgnoreGit {
		engine.Config.ignoreMap.git = readGitIgnore(engine.Config.RootPath)
	}
	err := runFromString(engine.Config.BackgroundExec, false)
	if err != nil {
		slog.Error(fmt.Sprintf("Running Background Process: %s", err.Error()))
		os.Exit(1)
	}
	engine.watch()
}

func (engine *Engine) Stop() {
	if runtime.GOOS == "windows" {
		err := killWindows(int(engine.Process.Pid))
		if err != nil {
			slog.Error("Could not kill windows process")
		}
	} else {
		killProcess(engine.Process)
	}
	notify.Stop(engine.Chan)
}

// This is out of date
func NewEngine(rootPath, execCommand, logLevel string, execList []string, ignore Ignore, debounce int, chunkSize string) *Engine {
	engine := &Engine{}
	engine.Config = Config{
		RootPath: rootPath,
		ExecList: execList,
		LogLevel: logLevel,
		Ignore:   ignore,
		Debounce: debounce,
	}
	engine.verifyConfig()
	return engine
}

func NewEngineFromConfig(options Config) *Engine {
	engine := &Engine{}
	engine.Config = options
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.verifyConfig()
	return engine
}

func NewEngineFromTOML(confPath string) *Engine {
	engine := Engine{}
	engine.readConfigFile(confPath)
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.verifyConfig()
	return &engine
}
