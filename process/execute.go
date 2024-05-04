package process

import (
	"os/exec"
	"strings"
)

type Execute struct {
	Cmd       string `toml:"cmd" yaml:"cmd"`               // Execute command
	ChangeDir string `toml:"dir" yaml:"dir"`               // If directory needs to be changed to call this command relative to the root path
	DelayNext int    `toml:"delay_next" yaml:"delay_next"` // Delay in milliseconds before running command
	// Type can have one of a few types to define how it reacts to a file change
	// background -- runs once at startup and is killed when refresh is canceled
	// once -- runs once at refresh startup but is blocking
	// blocking -- runs every refresh cycle as a blocking process
	// primary -- Is the primary process that kills the previous processes before running
	Type string `toml:"type" yaml:"type"`
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

// Takes a string and splits it on spaces to create a slice of strings
func generateExec(cmd string) *exec.Cmd {
	slice := strings.Split(cmd, " ")
	cmdEx := exec.Command(slice[0], slice[1:]...)
	return cmdEx
}
