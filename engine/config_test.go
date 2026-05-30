package engine

import (
	"testing"

	"github.com/atterpac/refresh/process"
)

func TestExecListToSpecs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []process.Execute
	}{
		{
			name: "refresh marker promotes following command to primary",
			in:   []string{"go mod tidy", "go build -o ./app", "KILL_STALE", "REFRESH", "./app"},
			want: []process.Execute{
				{Cmd: "go mod tidy", Type: process.Blocking},
				{Cmd: "go build -o ./app", Type: process.Blocking},
				{Cmd: "./app", Type: process.Primary},
			},
		},
		{
			name: "no refresh marker makes the last command primary",
			in:   []string{"go build -o ./app", "./app"},
			want: []process.Execute{
				{Cmd: "go build -o ./app", Type: process.Blocking},
				{Cmd: "./app", Type: process.Primary},
			},
		},
		{
			name: "whitespace and empties are trimmed and dropped",
			in:   []string{" go build ", "", "REFRESH", " ./app "},
			want: []process.Execute{
				{Cmd: "go build", Type: process.Blocking},
				{Cmd: "./app", Type: process.Primary},
			},
		},
		{
			name: "empty list yields no specs",
			in:   []string{""},
			want: []process.Execute{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := execListToSpecs(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d specs, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("spec[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestVerifyExecuteRejectsMultiplePrimaries(t *testing.T) {
	e := &Engine{Config: Config{
		RootPath: ".",
		ExecStruct: []process.Execute{
			{Cmd: "a", Type: process.Primary},
			{Cmd: "b", Type: process.Primary},
		},
	}}
	if err := e.verifyExecute(); err == nil {
		t.Fatal("expected error for two primary executes")
	}
}

func TestVerifyExecuteRequiresAtLeastOne(t *testing.T) {
	e := &Engine{Config: Config{RootPath: "."}}
	if err := e.verifyExecute(); err == nil {
		t.Fatal("expected error when no executes are configured")
	}
}

func TestBackgroundStructBecomesProcess(t *testing.T) {
	eng, err := NewEngineFromConfig(Config{
		RootPath:         ".",
		BackgroundStruct: process.Execute{Cmd: "echo bg"},
		ExecStruct:       []process.Execute{{Cmd: "./app", Type: process.Primary}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The background command is registered ahead of the configured executes.
	if got := eng.ProcessManager.GetExecutes(); len(got) != 2 || got[0] != "echo bg" || got[1] != "./app" {
		t.Errorf("executes = %v, want [echo bg, ./app]", got)
	}
}

func TestBackgroundTypeIsIgnored(t *testing.T) {
	// A type set on the background block is dropped: the background command always
	// registers as a background process regardless of what Type was configured.
	eng, err := NewEngineFromConfig(Config{
		RootPath:         ".",
		BackgroundStruct: process.Execute{Cmd: "echo bg", Type: process.Primary},
		ExecStruct:       []process.Execute{{Cmd: "./app", Type: process.Primary}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.ProcessManager.Processes[0].Type; got != process.Background {
		t.Errorf("background process type = %q, want %q (type on the background block must be ignored)", got, process.Background)
	}
}

func TestNormalizeExecutesPrefersStruct(t *testing.T) {
	e := &Engine{Config: Config{
		RootPath:   ".",
		ExecStruct: []process.Execute{{Cmd: "./app", Type: process.Primary}},
		ExecList:   []string{"should", "be", "ignored"},
	}}
	e.normalizeExecutes()
	if len(e.Config.ExecStruct) != 1 || e.Config.ExecStruct[0].Cmd != "./app" {
		t.Fatalf("ExecStruct should be preferred over ExecList: %+v", e.Config.ExecStruct)
	}
}
