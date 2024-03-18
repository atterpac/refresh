package engine

import (
	"context"
	"os"
	"os/exec"
	"sync"
)

type Process struct {
	Exec       string
	Blocking   bool
	Background bool
	Primary    bool
	cmd        *exec.Cmd
	termCh     chan struct{}
	doneCh     chan struct{}
	pty        *os.File
	pid        int
	pgid       int
	ctx        context.Context
	cancel     context.CancelFunc
}

type ProcessManager struct {
	processes []*Process
	mu        sync.RWMutex
	ctxs      map[string]context.Context
	cancels   map[string]context.CancelFunc
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make([]*Process, 0, 10),
		ctxs:      make(map[string]context.Context),
		cancels:   make(map[string]context.CancelFunc),
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
