package engine

import (
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime"

	"github.com/rjeczalik/notify"
)

type Engine struct {
	Process        *os.Process
	Chan           chan notify.EventInfo
	Active         bool
	Config         Config `toml:"config" yaml:"config"`
	ProcessLogFile *os.File
	ProcessLogPipe io.ReadCloser
}

func (engine *Engine) Start() {
	engine.Config.Slog = newLogger(engine.Config.LogLevel)
	engine.Config.externalSlog = false
	slog.SetDefault(engine.Config.Slog)
	slog.Info("Refresh Start")
	if engine.Config.Ignore.IgnoreGit {
		engine.Config.ignoreMap.git = readGitIgnore(engine.Config.RootPath)
	}
	go backgroundExec(engine.Config.BackgroundStruct.Cmd)
	go engine.reloadProcess()
	go engine.SigTrap()
	engine.watch()
}

func (engine *Engine) SigTrap() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		slog.Warn("Graceful Exit")
		engine.Stop()
		os.Exit(0)
	}()
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

func (engine *Engine) SetLogger(logger *slog.Logger) {
	engine.Config.Slog = logger
	engine.Config.externalSlog = true
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
	engine.Config.externalSlog = false
	engine.verifyConfig()
	return &engine
}

func NewEngineFromYAML(confPath string) *Engine {
	engine := Engine{}
	engine.readConfigYaml(confPath)
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.Config.externalSlog = false
	engine.verifyConfig()
	return &engine
}
