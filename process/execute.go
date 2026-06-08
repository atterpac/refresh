package process

import (
	"fmt"
	"os/exec"
)

type Execute struct {
	// Name is a stable, human-meaningful identifier for the process. Consumers
	// (e.g. a TUI) use it as the key for a per-process log pane. Optional; when
	// empty it defaults to the command string.
	Name      string `toml:"name"       yaml:"name"`
	Cmd       string `toml:"cmd"        yaml:"cmd"`        // Execute command
	ChangeDir string `toml:"dir"        yaml:"dir"`        // If directory needs to be changed to call this command relative to the root path
	DelayNext int    `toml:"delay_next" yaml:"delay_next"` // Pause in ms held after this step completes, before the next process starts
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

// generateExec builds a command run through the platform shell, so command
// strings may use quoting, pipes, &&, and redirection rather than being a bare
// argv split on spaces.
func generateExec(cmd string) *exec.Cmd {
	shell, args := shellInvocation(cmd)
	return exec.Command(shell, args...)
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
