package process

import (
	"context"
	"os/exec"
	"sync"
)

type Process struct {
	Exec       string
	Blocking   bool
	Background bool
	Primary    bool
	cmd        *exec.Cmd
	pid        int
	pgid       int
}

type ProcessManager struct {
	Processes []*Process
	mu        sync.RWMutex
	Ctxs      map[string]context.Context
	Cancels   map[string]context.CancelFunc
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		Processes: make([]*Process, 0),
		Ctxs:      make(map[string]context.Context),
		Cancels:   make(map[string]context.CancelFunc),
	}
}

func (pm *ProcessManager) AddProcess(exec string, blocking bool, primary bool, background bool) {
	pm.Processes = append(pm.Processes, &Process{
		Exec:       exec,
		Blocking:   blocking,
		Primary:    primary,
		Background: background,
	})
}

func (pm *ProcessManager) GetExecutes() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var execs []string
	for _, p := range pm.Processes {
		execs = append(execs, p.Exec)
	}
	return execs
}
