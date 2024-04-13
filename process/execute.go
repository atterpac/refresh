package process

import (
	"os"
	"os/exec"
	"strings"
)

type Execute struct {
	Cmd        string      `toml:"cmd" yaml:"cmd"`               // Execute command
	ChangeDir  string      `toml:"dir" yaml:"dir"`               // If directory needs to be changed to call this command relative to the root path
	IsBlocking bool        `toml:"blocking" yaml:"blocking"`     // Should the following executes wait for this one to complete
	IsPrimary  bool        `toml:"primary" yaml:"primary"`       // Only one primary command can be run at a time
	DelayNext  int         `toml:"delay_next" yaml:"delay_next"` // Delay in milliseconds before running command
	process    *os.Process // Stores the Exec.Start() process
}

var KILL_STALE = Execute{
	Cmd:        "KILL_STALE",
	IsBlocking: true,
	IsPrimary:  false,
}

var REFRESH_EXEC = "REFRESH"
var KILL_EXEC = "KILL_STALE"

// Takes a string and splits it on spaces to create a slice of strings
func generateExec(cmd string) *exec.Cmd {
	slice := strings.Split(cmd, " ")
	cmdEx := exec.Command(slice[0], slice[1:]...)
	return cmdEx
}
