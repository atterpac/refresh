package engine

import (
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
	if engine.Config.Ignore.IgnoreGit {
		engine.Config.ignoreMap.git = readGitIgnore(engine.Config.RootPath)
	}
	engine.watch()
}

func (engine *Engine) Stop() {
	if runtime.GOOS == "windows" {
		killWindows(int(engine.Process.Pid))
	} else {
		killProcess(engine.Process)
	}
	notify.Stop(engine.Chan)
}

func NewEngine(rootPath, execCommand, logLevel string, ignore Ignore, debounce int, chunkSize string) *Engine {
	engine := &Engine{}
	engine.Config = Config{
		RootPath:    rootPath,
		ExecCommand: execCommand,
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
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.verifyConfig()
	return engine
}

func NewEngineFromTOML(confPath string) *Engine {
	engine := Engine{}
	engine.readConfigFile(confPath)
	engine.verifyConfig()
	return &engine
}



