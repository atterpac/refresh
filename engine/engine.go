//go:build windows || linux || darwin
// +build windows linux darwin

package engine

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/atterpac/refresh/process"
	"github.com/rjeczalik/notify"
)

type Engine struct {
	PrimaryProcess process.Process
	BgProcess      process.Process
	Chan           chan notify.EventInfo
	Active         bool
	Config         Config `toml:"config" yaml:"config"`
	ProcessLogFile *os.File
	ProcessLogPipe io.ReadCloser
	ProcessManager *process.ProcessManager
	ctx            context.Context
	cancel         context.CancelFunc
	isPaused       bool
}

func (engine *Engine) Start() error {
	config := engine.Config
	slog.Info("Refresh Starting...")
	if config.Ignore.IgnoreGit {
		config.ignoreMap.git = readGitIgnore(config.RootPath)
	}

	waitTime := time.Duration(engine.Config.BackgroundStruct.DelayNext) * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	engine.ctx = ctx
	engine.cancel = cancel
	time.Sleep(waitTime)

	trapChan := make(chan error)
	go engine.sigTrap(trapChan)
	go engine.ProcessManager.StartProcess(engine.ctx, engine.cancel)
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.Canceled {
			if !engine.ProcessManager.FirstRun {
				slog.Error("Could not refresh processes due to execution errors")
				newCtx, newCancel := context.WithCancel(context.Background())
				engine.ctx = newCtx
				engine.cancel = newCancel
				return
			}
			engine.Stop()
			trapChan <- errors.New("An error occured while starting proceses")
		}
	}()

	eventManager := NewEventManager(engine, engine.Config.Debounce)
	go engine.watch(eventManager)
	return <-trapChan
}

func (engine *Engine) Stop() {
	engine.ProcessManager.KillProcesses()
	engine.cancel()
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
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
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
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	return engine, nil
}

func NewEngineFromTOML(confPath string) (*Engine, error) {
	engine := Engine{}
	_, err := engine.readConfigFile(confPath)
	if err != nil {
		return nil, err
	}
	config := engine.Config
	config.Slog = newLogger(config.LogLevel)
	config.externalSlog = false
	slog.SetDefault(config.Slog)
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.Config.externalSlog = false
	err = engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	return &engine, nil
}

func NewEngineFromYAML(confPath string) (*Engine, error) {
	engine := Engine{}
	_, err := engine.readConfigYaml(confPath)
	if err != nil {
		return nil, err
	}
	config := engine.Config
	config.Slog = newLogger(config.LogLevel)
	config.externalSlog = false
	slog.SetDefault(config.Slog)
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	engine.Config.externalSlog = false
	err = engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	return &engine, nil
}

func (engine *Engine) AttachBackgroundCallback(callback func() bool) *Engine {
	engine.Config.BackgroundCallback = callback
	return engine
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
