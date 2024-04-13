package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

var firstRun = true

type Process struct {
	Exec       string
	Blocking   bool
	Background bool
	Primary    bool
	logPipe    io.ReadCloser
	cmd        *exec.Cmd
	pid        int
	pgid       int
}

type ProcessManager struct {
	Processes []*Process
	mu        sync.RWMutex
	Ctxs      map[string]context.Context
	Cancels   map[string]context.CancelFunc
	mainCtx   context.Context
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

func printSubProcess(ctx context.Context, pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	defer pipe.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}
	}
}
