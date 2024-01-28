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
	BgProcessTree  Process
	Chan           chan notify.EventInfo
	Active         bool
	Config         Config `toml:"config" yaml:"config"`
	ProcessLogFile *os.File
	ProcessLogPipe io.ReadCloser
}

func (engine *Engine) Start() error {
	var err error
	config := engine.Config
	slog.Info("Refresh Start")
	if config.Ignore.IgnoreGit {
		config.ignoreMap.git = readGitIgnore(config.RootPath)
	}
	engine.BgProcessTree, err = engine.startBackgroundProcess(config.BackgroundStruct.Cmd)
	if err != nil {
		slog.Error("Starting Background Process", "err", err.Error())
		return err
	}
	if config.BackgroundCallback != nil {
		ok := config.BackgroundCallback()
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

func (engine *Engine) Stop() {
	engine.killProcess(engine.ProcessTree)
	engine.killProcess(engine.BgProcessTree)
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
