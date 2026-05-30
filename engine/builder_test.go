package engine

import (
	"bytes"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/atterpac/refresh/process"
)

func TestDefaultEngineConfig(t *testing.T) {
	c := DefaultEngineConfig()
	if c.RootPath != "." {
		t.Errorf("RootPath = %q, want .", c.RootPath)
	}
	if c.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn", c.LogLevel)
	}
	if c.Debounce != 1000 {
		t.Errorf("Debounce = %d, want 1000", c.Debounce)
	}
	if !c.Ignore.IgnoreGit {
		t.Error("IgnoreGit = false, want true")
	}
	if !slices.Contains(c.Ignore.Dir, ".git") {
		t.Errorf("Ignore.Dir = %v, want it to contain .git", c.Ignore.Dir)
	}
}

func TestConfigBuilders(t *testing.T) {
	c := DefaultEngineConfig()
	c.WithRootPath("./app").
		WithLogLevel("debug").
		WithDebounce(50).
		WithIgnoreDirs([]string{"x"}).
		WithIgnoreFiles([]string{"y.go"}).
		WithIgnoreGit(false).
		WithWatchedExtensions([]string{"*.go"}).
		WithExecuteCommand(process.Execute{Cmd: "./app", Type: process.Primary})

	if c.RootPath != "./app" {
		t.Errorf("RootPath = %q", c.RootPath)
	}
	if c.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", c.LogLevel)
	}
	if c.Debounce != 50 {
		t.Errorf("Debounce = %d", c.Debounce)
	}
	if c.Ignore.IgnoreGit {
		t.Error("IgnoreGit should be false after WithIgnoreGit(false)")
	}
	if !slices.Equal(c.Ignore.Dir, []string{"x"}) {
		t.Errorf("Ignore.Dir = %v", c.Ignore.Dir)
	}
	if !slices.Equal(c.Ignore.File, []string{"y.go"}) {
		t.Errorf("Ignore.File = %v", c.Ignore.File)
	}
	if !slices.Equal(c.Ignore.WatchedExten, []string{"*.go"}) {
		t.Errorf("Ignore.WatchedExten = %v", c.Ignore.WatchedExten)
	}
	if len(c.ExecStruct) != 1 || c.ExecStruct[0].Cmd != "./app" {
		t.Errorf("ExecStruct = %+v", c.ExecStruct)
	}
}

func TestWithIgnoreReplacesWholeStruct(t *testing.T) {
	c := DefaultEngineConfig()
	c.WithIgnore(Ignore{Dir: []string{"z"}})
	if !slices.Equal(c.Ignore.Dir, []string{"z"}) || c.Ignore.IgnoreGit {
		t.Errorf("WithIgnore did not replace the struct: %+v", c.Ignore)
	}
}

// TestEngineLoggingControls covers SetLogger plus the engine-level runtime
// controls (SetLogLevel/DisableLogs/EnableLogs) end to end, by routing a
// caller-supplied logger into a buffer.
func TestEngineLoggingControls(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	eng := &Engine{Config: Config{LogLevel: "info"}}
	eng.SetLogger(custom)

	eng.Config.Slog.Info("hello")
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("SetLogger did not route output: %q", buf.String())
	}

	buf.Reset()
	eng.DisableLogs()
	eng.Config.Slog.Error("suppressed")
	if buf.Len() != 0 {
		t.Errorf("DisableLogs did not mute output: %q", buf.String())
	}

	buf.Reset()
	eng.EnableLogs()
	eng.SetLogLevel("debug")
	eng.Config.Slog.Debug("verbose")
	if !strings.Contains(buf.String(), "verbose") {
		t.Errorf("EnableLogs + SetLogLevel(debug) failed: %q", buf.String())
	}
}

// TestEngineLogControlsNilSafe verifies the controls are safe before a logger is
// installed and that SetLogLevel records the level for later initialization.
func TestEngineLogControlsNilSafe(t *testing.T) {
	eng := &Engine{}
	eng.DisableLogs() // must not panic
	eng.EnableLogs()
	eng.SetLogLevel("warn")
	if eng.Config.LogLevel != "warn" {
		t.Errorf("SetLogLevel before init = %q, want warn recorded", eng.Config.LogLevel)
	}
}
