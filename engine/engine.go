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
	log            *dynamicLogger
}

// initLogger builds the engine's logger from the configured level (and an
// optional caller-supplied logger) and installs it as the slog default so that
// every package — including process — routes through the same controllable
// handler. Called by all constructors.
func (engine *Engine) initLogger() {
	engine.log = newDynamicLogger(engine.Config.LogLevel, engine.Config.Slog)
	engine.Config.Slog = engine.log.logger
	slog.SetDefault(engine.log.logger)
}

// SetLogLevel changes the active log level at runtime. Accepts
// "debug", "info", "warn", "error", or "mute". Safe to call from any goroutine.
func (engine *Engine) SetLogLevel(level string) {
	if engine.log == nil {
		engine.Config.LogLevel = level
		return
	}
	engine.log.SetLevel(level)
}

// DisableLogs mutes all engine output without discarding the configured level,
// so EnableLogs restores the previous verbosity.
func (engine *Engine) DisableLogs() {
	if engine.log != nil {
		engine.log.Disable()
	}
}

// EnableLogs resumes output after DisableLogs.
func (engine *Engine) EnableLogs() {
	if engine.log != nil {
		engine.log.Enable()
	}
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

// SetLogger replaces the engine's logger with a caller-supplied one. The logger
// is still wrapped so SetLogLevel/DisableLogs/EnableLogs continue to work.
func (engine *Engine) SetLogger(logger *slog.Logger) {
	engine.log = newDynamicLogger(engine.Config.LogLevel, logger)
	engine.Config.Slog = engine.log.logger
	slog.SetDefault(engine.log.logger)
}

// Deprecated: NewEngine predates the Config-based constructors and does not wire
// up the full lifecycle. Use NewEngineFromConfig instead.
func NewEngine(rootPath, execCommand, logLevel string, execList []string, ignore Ignore, debounce int, chunkSize string) (*Engine, error) {
	return NewEngineFromConfig(Config{
		RootPath: rootPath,
		ExecList: execList,
		LogLevel: logLevel,
		Ignore:   ignore,
		Debounce: debounce,
	})
}

func NewEngineFromConfig(options Config) (*Engine, error) {
	engine := &Engine{}
	engine.Config = options
	engine.initLogger()
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	err := engine.verifyConfig()
	if err != nil {
		return nil, err
	}
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	_ = engine.ProcessManager.SetRootDirectory(engine.Config.RootPath)
	return engine, nil
}

func NewEngineFromTOML(confPath string) (*Engine, error) {
	engine := &Engine{}
	if _, err := engine.readConfigFile(confPath); err != nil {
		return nil, err
	}
	engine.initLogger()
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	if err := engine.verifyConfig(); err != nil {
		return nil, err
	}
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	_ = engine.ProcessManager.SetRootDirectory(engine.Config.RootPath)
	return engine, nil
}

func NewEngineFromYAML(confPath string) (*Engine, error) {
	engine := &Engine{}
	if _, err := engine.readConfigYaml(confPath); err != nil {
		return nil, err
	}
	engine.initLogger()
	engine.Config.ignoreMap = convertToIgnoreMap(engine.Config.Ignore)
	if err := engine.verifyConfig(); err != nil {
		return nil, err
	}
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	_ = engine.ProcessManager.SetRootDirectory(engine.Config.RootPath)
	return engine, nil
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
