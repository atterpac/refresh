package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
)

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
	FirstRun  bool
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		Processes: make([]*Process, 0),
		Ctxs:      make(map[string]context.Context),
		Cancels:   make(map[string]context.CancelFunc),
		FirstRun:  true,
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
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context closed, stopping printSubProcess")
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					slog.Debug("Scanner error", "err", err)
				}
				return
			}
			fmt.Println(scanner.Text())
		}
	}
}
