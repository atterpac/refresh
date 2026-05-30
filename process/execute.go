package process

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Execute struct {
	Cmd       string `toml:"cmd"        yaml:"cmd"`        // Execute command
	ChangeDir string `toml:"dir"        yaml:"dir"`        // If directory needs to be changed to call this command relative to the root path
	DelayNext int    `toml:"delay_next" yaml:"delay_next"` // Delay in milliseconds before running command
	// Type can have one of a few types to define how it reacts to a file change
	// background -- runs once at startup and is killed when refresh is canceled
	// once -- runs once at refresh startup but is blocking
	// blocking -- runs every refresh cycle as a blocking process
	// primary -- Is the primary process that kills the previous processes before running
	Type ExecuteType `toml:"type"       yaml:"type"`
}

type ExecuteType string

var (
	Background ExecuteType = "background"
	Once       ExecuteType = "once"
	Blocking   ExecuteType = "blocking"
	Primary    ExecuteType = "primary"
)

var KILL_STALE = Execute{
	Cmd:  "KILL_STALE",
	Type: "blocking",
}

var REFRESH_EXEC = "REFRESH"
var KILL_EXEC = "KILL_STALE"

// generateExec splits a command string on whitespace into a runnable command.
// strings.Fields collapses repeated spaces so "go  build" is handled correctly.
func generateExec(cmd string) *exec.Cmd {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return exec.Command("")
	}
	return exec.Command(fields[0], fields[1:]...)
}

// generateExecContext is generateExec bound to a context, so the command is
// killed if the context is cancelled (used for blocking/once steps).
func generateExecContext(ctx context.Context, cmd string) *exec.Cmd {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return exec.CommandContext(ctx, "")
	}
	return exec.CommandContext(ctx, fields[0], fields[1:]...)
}

func stringToExecuteType(typing string) (ExecuteType, error) {
	switch typing {
	case "background":
		return Background, nil
	case "once":
		return Once, nil
	case "blocking":
		return Blocking, nil
	case "primary":
		return Primary, nil
	default:
		return "", fmt.Errorf("execute type of %q is invalid", typing)
	}
}
