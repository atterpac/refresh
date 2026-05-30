package main

import (
	"slices"
	"testing"
)

func TestParseFlagsToConfig(t *testing.T) {
	f, err := parseFlags([]string{
		"-p", "./svc",
		"-e", "go build, REFRESH, ./svc",
		"-l", "warn",
		"-d", "250",
		"-id", ".git, vendor",
		"-ie", "*.go",
		"-git",
	})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	cfg := f.toConfig()
	if cfg.RootPath != "./svc" {
		t.Errorf("RootPath = %q, want ./svc", cfg.RootPath)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn", cfg.LogLevel)
	}
	if cfg.Debounce != 250 {
		t.Errorf("Debounce = %d, want 250", cfg.Debounce)
	}
	if !cfg.Ignore.IgnoreGit {
		t.Error("IgnoreGit = false, want true")
	}
	if want := []string{"go build", "REFRESH", "./svc"}; !slices.Equal(cfg.ExecList, want) {
		t.Errorf("ExecList = %v, want %v", cfg.ExecList, want)
	}
	if want := []string{".git", "vendor"}; !slices.Equal(cfg.Ignore.Dir, want) {
		t.Errorf("Ignore.Dir = %v, want %v", cfg.Ignore.Dir, want)
	}
	if want := []string{"*.go"}; !slices.Equal(cfg.Ignore.WatchedExten, want) {
		t.Errorf("Ignore.WatchedExten = %v, want %v", cfg.Ignore.WatchedExten, want)
	}
}

func TestSplitListDropsEmpties(t *testing.T) {
	// An unset flag must yield nil, not [""] (the previous bug, which polluted
	// the ignore lists with an empty string).
	if got := splitList(""); got != nil {
		t.Errorf("splitList(\"\") = %v, want nil", got)
	}
	if got := splitList("a, ,b,"); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("splitList = %v, want [a b]", got)
	}
}

func TestNewEngineRejectsUnsupportedConfigExtension(t *testing.T) {
	if _, err := newEngine(cliFlags{configPath: "config.json"}); err == nil {
		t.Fatal("expected error for unsupported config extension")
	}
}

func TestNewEngineFromFlags(t *testing.T) {
	eng, err := newEngine(cliFlags{
		rootPath:    ".",
		execCommand: "sleep 1",
		logLevel:    "mute",
		debounce:    100,
	})
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	// The single command should have been promoted to the primary process.
	if execs := eng.ProcessManager.GetExecutes(); !slices.Equal(execs, []string{"sleep 1"}) {
		t.Errorf("executes = %v, want [sleep 1]", execs)
	}
}
