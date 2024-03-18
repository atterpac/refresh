package engine

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// SetDefault sets the default logger.
func newLogger(level string) *slog.Logger {
	var writer io.Writer = os.Stderr
	if level == "mute" {
		writer = io.Discard
	}
	return slog.New(tint.NewHandler(writer, &tint.Options{
		Level:      getLogLevel(level),
		TimeFormat: time.Kitchen,
	}))
}

func printSubProcess(ctx context.Context, pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	defer pipe.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				println(scanner.Text())
			} else {
				if err := scanner.Err(); err != nil {
					slog.Debug("Couldnt connect to process log pipe", "err", err)
				}
				return
			}
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
