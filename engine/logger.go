package engine

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// SetDefault sets the default logger.
func newLogger(level string) *slog.Logger {
	tint := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      getLogLevel(level),
		TimeFormat: time.Kitchen,
	}))
	return tint
}

func printSubProcess(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for {
		for scanner.Scan() {
			println(scanner.Text())
		}
	}
}

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
