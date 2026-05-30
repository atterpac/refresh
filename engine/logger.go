package engine

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/lmittmann/tint"
)

// dynamicLogger wraps an slog.Logger with two runtime controls that survive
// across handler boundaries (WithAttrs/WithGroup):
//
//   - level   — the minimum level to emit, changeable at runtime
//   - enabled — a master switch to mute/unmute all output
//
// Both are shared by pointer with every derived handler, so a single call to
// SetLevel/Disable/Enable affects the whole logger tree immediately and is safe
// to call from any goroutine.
type dynamicLogger struct {
	level   *slog.LevelVar
	enabled *atomic.Bool
	logger  *slog.Logger
}

// switchHandler gates an inner handler behind a dynamic level and an enabled
// flag. Gating in Enabled (rather than rebuilding handlers) keeps level/mute
// changes atomic and lock-free, and applies uniformly even to a caller-supplied
// handler that has its own internal level.
type switchHandler struct {
	inner   slog.Handler
	level   *slog.LevelVar
	enabled *atomic.Bool
}

func (h *switchHandler) Enabled(_ context.Context, l slog.Level) bool {
	if !h.enabled.Load() {
		return false
	}
	return l >= h.level.Level()
}

func (h *switchHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.inner.Handle(ctx, r)
}

func (h *switchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &switchHandler{inner: h.inner.WithAttrs(attrs), level: h.level, enabled: h.enabled}
}

func (h *switchHandler) WithGroup(name string) slog.Handler {
	return &switchHandler{inner: h.inner.WithGroup(name), level: h.level, enabled: h.enabled}
}

// newDynamicLogger builds a logger from the configured level string. When a
// caller supplies their own *slog.Logger, its handler is wrapped so the enable
// /disable switch and runtime level still apply; otherwise a tinted stderr
// handler is used.
func newDynamicLogger(level string, custom *slog.Logger) *dynamicLogger {
	levelVar := new(slog.LevelVar)
	enabled := new(atomic.Bool)

	muted := level == "mute"
	enabled.Store(!muted)
	if muted {
		// Keep a sensible threshold so re-enabling logs something useful.
		levelVar.Set(slog.LevelInfo)
	} else {
		levelVar.Set(getLogLevel(level))
	}

	var inner slog.Handler
	if custom != nil {
		inner = custom.Handler()
	} else {
		inner = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      levelVar,
			TimeFormat: time.Kitchen,
		})
	}

	return &dynamicLogger{
		level:   levelVar,
		enabled: enabled,
		logger:  slog.New(&switchHandler{inner: inner, level: levelVar, enabled: enabled}),
	}
}

// SetLevel changes the minimum emitted level at runtime. The special value
// "mute" disables all output; any other recognized level re-enables it.
func (d *dynamicLogger) SetLevel(level string) {
	if level == "mute" {
		d.enabled.Store(false)
		return
	}
	d.level.Set(getLogLevel(level))
	d.enabled.Store(true)
}

func (d *dynamicLogger) Disable() { d.enabled.Store(false) }
func (d *dynamicLogger) Enable()  { d.enabled.Store(true) }

func getLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
