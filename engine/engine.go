package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/atterpac/refresh/process"
)

type Engine struct {
	Config         Config `toml:"config" yaml:"config"`
	ProcessManager *process.ProcessManager
	ctx            context.Context
	cancel         context.CancelFunc
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

// Start runs the initial process pass, begins watching the filesystem, and then
// blocks on a single supervisor loop that serializes reloads and shutdown. It
// returns nil on a clean (signal-triggered) exit, or an error if the initial
// startup fails.
func (engine *Engine) Start() error {
	slog.Info("refresh starting")

	if len(EventMap) == 0 {
		return errors.New("file watching is not supported on this platform")
	}
	if engine.Config.Ignore.IgnoreGit {
		engine.Config.Ignore.gitPatterns = readGitIgnore(engine.Config.RootPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	engine.ctx = ctx
	engine.cancel = cancel

	// Initial pass over all configured processes.
	if err := engine.ProcessManager.Start(ctx); err != nil {
		engine.ProcessManager.Shutdown()
		cancel()
		return fmt.Errorf("starting processes: %w", err)
	}

	engine.trapSignals()

	// Reload requests from the watcher arrive here; the buffer of one plus a
	// non-blocking send coalesces bursts into a single pending reload.
	reload := make(chan struct{}, 1)
	if err := engine.startWatcher(ctx, reload); err != nil {
		engine.Stop()
		engine.ProcessManager.Shutdown()
		return err
	}

	// Optional pause/resume toggle, driven by the suspend key (Ctrl+Z). The
	// buffered, non-blocking send keeps the toggle from blocking the signal
	// goroutine and coalesces rapid presses.
	toggle := make(chan struct{}, 1)
	if engine.Config.EnablePause {
		engine.trapControlSignals(toggle)
	}

	// Supervisor loop: the only goroutine that drives process lifecycle, so the
	// process manager needs no locking around its runtime state. paused and
	// pending live here for the same reason.
	paused := false
	pending := false // a reload arrived while paused

	for {
		select {
		case <-ctx.Done():
			engine.ProcessManager.Shutdown()
			slog.Info("refresh stopped")
			return nil
		case <-toggle:
			paused = !paused
			if paused {
				slog.Warn("refresh paused — file changes ignored until resumed")
				continue
			}
			slog.Warn("refresh resumed")
			if pending {
				pending = false
				slog.Info("applying change made while paused, reloading")
				if err := engine.ProcessManager.Reload(ctx); err != nil {
					slog.Error("reload failed", "err", err)
				}
			}
		case <-reload:
			if paused {
				pending = true
				continue
			}
			slog.Info("change detected, reloading")
			if err := engine.ProcessManager.Reload(ctx); err != nil {
				slog.Error("reload failed", "err", err)
			}
		}
	}
}

// Stop requests a graceful shutdown. The supervisor loop performs the actual
// process teardown when the context is cancelled.
func (engine *Engine) Stop() {
	if engine.cancel != nil {
		engine.cancel()
	}
}

// SetLogger replaces the engine's logger with a caller-supplied one. The logger
// is still wrapped so SetLogLevel/DisableLogs/EnableLogs continue to work.
func (engine *Engine) SetLogger(logger *slog.Logger) {
	engine.log = newDynamicLogger(engine.Config.LogLevel, logger)
	engine.Config.Slog = engine.log.logger
	slog.SetDefault(engine.log.logger)
}

func NewEngineFromConfig(options Config) (*Engine, error) {
	engine := &Engine{}
	engine.Config = options
	engine.initLogger()
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
	if err := engine.verifyConfig(); err != nil {
		return nil, err
	}
	engine.ProcessManager = process.NewProcessManager()
	engine.generateProcess()
	_ = engine.ProcessManager.SetRootDirectory(engine.Config.RootPath)
	return engine, nil
}

// trapSignals cancels the engine context on the first interrupt/terminate
// signal, which lets the supervisor loop tear everything down gracefully.
func (engine *Engine) trapSignals() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChan
		slog.Warn("graceful exit requested", "signal", sig)
		engine.Stop()
	}()
}
