package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/atterpac/refresh/process"
)

type Engine struct {
	Config         Config `toml:"config" yaml:"config"`
	ProcessManager *process.ProcessManager
	ctx            context.Context
	cancel         context.CancelFunc
	log            *dynamicLogger

	// Control plane for programmatic Reload/Pause/Resume. reloadCh carries reload
	// requests (from the watcher, the Ctrl+Z path, or Reload); wakeCh nudges the
	// supervisor to re-evaluate after a resume so a change made while paused is
	// applied. Both are buffered with a non-blocking send so producers never
	// block. paused is the authoritative pause flag, owned by the setters and
	// read by the supervisor and Paused.
	reloadCh chan struct{}
	wakeCh   chan struct{}
	paused   atomic.Bool
}

// initControl allocates the control-plane channels. Called by every constructor
// so Reload/Pause/Resume are safe to call before (and after) the supervisor is
// running — sends are non-blocking, so they are simply dropped when no
// supervisor is draining them.
func (engine *Engine) initControl() {
	engine.reloadCh = make(chan struct{}, 1)
	engine.wakeCh = make(chan struct{}, 1)
}

// nonBlockingSend pokes a single-slot signal channel without ever blocking the
// caller; a poke that finds the slot already full is coalesced into the pending
// one.
func nonBlockingSend(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

// Reload triggers a reload cycle (re-run blocking steps, restart the primary),
// exactly as a file change would. Honors pause: if the engine is paused the
// reload is deferred and applied on Resume. Safe to call from any goroutine.
func (engine *Engine) Reload() {
	nonBlockingSend(engine.reloadCh)
}

// Pause suspends reload handling. File changes (and Reload calls) made while
// paused are remembered and applied on the next Resume. Idempotent.
func (engine *Engine) Pause() {
	if engine.paused.CompareAndSwap(false, true) {
		slog.Warn("refresh paused — changes deferred until resumed")
	}
}

// Resume re-enables reload handling and applies any change that arrived while
// paused. Idempotent.
func (engine *Engine) Resume() {
	if engine.paused.CompareAndSwap(true, false) {
		slog.Warn("refresh resumed")
		nonBlockingSend(engine.wakeCh)
	}
}

// Paused reports whether the engine is currently paused.
func (engine *Engine) Paused() bool {
	return engine.paused.Load()
}

// togglePause flips the pause state; used by the Ctrl+Z handler.
func (engine *Engine) togglePause() {
	if engine.Paused() {
		engine.Resume()
	} else {
		engine.Pause()
	}
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
	return engine.run(context.Background(), true)
}

// Run is Start for an embedding caller that owns process and signal handling
// itself — for example a TUI that drives the terminal and translates its own
// keystrokes into pause/reload. It runs the initial pass and supervises reloads
// until ctx is cancelled, but installs no OS signal handlers (no SIGINT/SIGTERM
// trap, and no Ctrl+Z pause trap even when EnablePause is set). Cancel ctx, or
// call Stop, to shut down. Run blocks; call it from its own goroutine.
func (engine *Engine) Run(ctx context.Context) error {
	return engine.run(ctx, false)
}

// Processes returns a snapshot of every configured process and its current
// runtime state. Safe to call from any goroutine, including before Start/Run
// (every process reports as pending until started).
func (engine *Engine) Processes() []process.ProcessInfo {
	return engine.ProcessManager.Snapshot()
}

func (engine *Engine) run(parent context.Context, trapOSSignals bool) error {
	slog.Info("refresh starting")

	if len(EventMap) == 0 {
		return errors.New("file watching is not supported on this platform")
	}
	if engine.Config.Ignore.IgnoreGit {
		engine.Config.Ignore.gitPatterns = readGitIgnore(engine.Config.RootPath)
	}

	ctx, cancel := context.WithCancel(parent)
	engine.ctx = ctx
	engine.cancel = cancel

	// Trap signals before the initial pass: it starts processes and runs blocking
	// steps synchronously, so an interrupt mid-pass must cancel ctx to tear them down.
	if trapOSSignals {
		engine.trapSignals()
	}

	// Initial pass over all configured processes.
	if err := engine.ProcessManager.Start(ctx); err != nil {
		engine.ProcessManager.Shutdown()
		// A cancelled context means an interrupt arrived mid-startup: that's a
		// clean shutdown, not a startup failure, so don't surface it as an error.
		if ctx.Err() != nil {
			slog.Info("refresh stopped")
			return nil
		}
		cancel()
		return fmt.Errorf("starting processes: %w", err)
	}

	// The watcher delivers reload requests on the same control channel that
	// Reload uses, so file-driven and programmatic reloads share one path.
	if err := engine.startWatcher(ctx, engine.reloadCh); err != nil {
		engine.Stop()
		engine.ProcessManager.Shutdown()
		return err
	}

	// Optional pause/resume via the suspend key (Ctrl+Z). Only wired when this
	// engine owns OS signals; an embedding caller (Run) drives Pause/Resume
	// through its own input handling instead. Programmatic Pause/Resume/Reload
	// work regardless of this.
	if trapOSSignals && engine.Config.EnablePause {
		engine.trapControlSignals()
	}

	// Supervisor loop: the only goroutine that drives process lifecycle, so the
	// process manager needs no locking around its handles. pending lives here;
	// the pause flag is the engine's atomic, set by Pause/Resume.
	pending := false // a reload arrived while paused

	for {
		select {
		case <-ctx.Done():
			engine.ProcessManager.Shutdown()
			slog.Info("refresh stopped")
			return nil
		case <-engine.wakeCh:
			// A resume occurred; apply any change deferred while paused.
			if !engine.paused.Load() && pending {
				pending = false
				slog.Info("applying change made while paused, reloading")
				if err := engine.ProcessManager.Reload(ctx); err != nil {
					slog.Error("reload failed", "err", err)
				}
			}
		case <-engine.reloadCh:
			if engine.paused.Load() {
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
	engine.initControl()
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
	engine.initControl()
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
	engine.initControl()
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
