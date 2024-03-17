package engine

import (
	"os/exec"
	"sync"
)

type Process struct {
	Exec       string
	Blocking   bool
	Background bool
	Primary    bool
	cmd        *exec.Cmd
	pgid       int
}

type ProcessManager struct {
	processes []*Process
	mu        sync.RWMutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make([]*Process, 0, 10),
	}
}

func (pm *ProcessManager) AddProcess(exec string, blocking bool, primary bool, background bool) {
	pm.processes = append(pm.processes, &Process{
		Exec:       exec,
		Blocking:   blocking,
		Primary:    primary,
		Background: background,
	})
}
