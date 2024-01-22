package engine

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/rjeczalik/notify"
)

type Engine struct {
	ProcessTree    Process
	Chan           chan notify.EventInfo
	Active         bool
	Config         Config `toml:"config" yaml:"config"`
	ProcessLogFile *os.File
	ProcessLogPipe io.ReadCloser
}

func (engine *Engine) Start() error {
	engine.Config.Slog = newLogger(engine.Config.LogLevel)
	engine.Config.externalSlog = false
	slog.SetDefault(engine.Config.Slog)
	slog.Info("Refresh Start")
	if engine.Config.Ignore.IgnoreGit {
		engine.Config.ignoreMap.git = readGitIgnore(engine.Config.RootPath)
	}
	engine.Config.BackgroundStruct.process = backgroundExec(engine.Config.BackgroundStruct.Cmd)
	if engine.Config.BackgroundCallback != nil {
		ok := engine.Config.BackgroundCallback()
		if !ok {
			slog.Error("Background Callback Failed")
			return errors.New("Background Callback Failed")
		}
	}
	waitTime := time.Duration(engine.Config.BackgroundStruct.DelayNext) * time.Millisecond
	time.Sleep(waitTime)
	go engine.reloadProcess()
	trapChan := make(chan error)
	go engine.sigTrap(trapChan)
	go engine.watch()
	return <-trapChan
}

func (engine *Engine) sigTrap(ch chan error) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		sig := <-signalChan
		slog.Warn("Graceful Exit Requested", "signal", sig)
		engine.Stop()
		ch <- errors.New("Graceful Exit Requested")
	}()
}

func (engine *Engine) Stop() {
	engine.killProcess(engine.ProcessTree)
	localProcess := Process{Process: engine.Config.BackgroundStruct.process}
	engine.killProcess(localProcess)
	notify.Stop(engine.Chan)
}

func (engine *Engine) SetLogger(logger *slog.Logger) {
	engine.Config.Slog = logger
	engine.Config.externalSlog = true
}

// This is out of date
func NewEngine(rootPath, execCommand, logLevel string, execList []string, ignore Ignore, debounce int, chunkSize string) (*Engine, error) {
	engine := &Engine{}
	engine.Config = Config{
		RootPath: rootPath,
		ExecList: execList,
		LogLevel: logLevel,
		Ignore:   ignore,
		Debounce: debounce,
	}
	err := engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func NewEngineFromConfig(options Config) (*Engine, error) {
	engine := &Engine{}
	engine.Config = options
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	err := engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func NewEngineFromTOML(confPath string) (*Engine, error) {
	engine := Engine{}
	_, err := engine.readConfigFile(confPath)
	if err != nil {
		return nil, err
	}
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.Config.externalSlog = false
	err = engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	return &engine, nil
}

func NewEngineFromYAML(confPath string) (*Engine, error) {
	engine := Engine{}
	_, err := engine.readConfigYaml(confPath)
	if err != nil {
		return nil, err
	}
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.Config.externalSlog = false
	err = engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	return &engine, nil
}

func (engine *Engine) AttachBackgroundCallback(callback func() bool) *Engine {
	engine.Config.BackgroundCallback = callback
	return engine
}

