package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

type Process struct {
	Exec    string
	Type    ExecuteType
	Dir     string
	logPipe io.ReadCloser
	cmd     *exec.Cmd
	pid     int
	pgid    int
}

type ProcessManager struct {
	Processes []*Process
	RootDir   string
	mu        sync.RWMutex
	Ctxs      map[string]context.Context
	Cancels   map[string]context.CancelFunc
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

func (pm *ProcessManager) AddProcess(exec string, typing string, dir string) error {
	execType, err := stringToExecuteType(typing)
	if err != nil {
		return err
	}
	pm.Processes = append(pm.Processes, &Process{
		Exec: exec,
		Type: execType,
		Dir:  dir,
	})
	return nil
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

func (pm *ProcessManager) SetRootDirectory(dir string) error {
	err := pm.ChangeExecuteDirectory(dir)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return errors.New("Unable to get working directory")
	}
	pm.RootDir = wd
	return nil
}

func (pm *ProcessManager) ChangeExecuteDirectory(dir string) error {
	err := os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("Unable to change execute directory: %s", dir)
	}
	return nil
}

func (pm *ProcessManager) RestoreRootDirectory() error {
	return pm.ChangeExecuteDirectory(pm.RootDir)
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
