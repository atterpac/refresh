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
	"path/filepath"
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
	// First, get the current working directory as a fallback
	currentDir, err := os.Getwd()
	if err != nil {
		return errors.New("Unable to get current working directory")
	}

	// If dir is empty, use the current directory
	if dir == "" {
		pm.RootDir = currentDir
		return nil
	}

	// Change to the requested directory
	err = os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("Unable to change to root directory %s: %w", dir, err)
	}

	// Get the absolute path of the new working directory
	absDir, err := os.Getwd()
	if err != nil {
		// If we can't get the absolute path, restore the original directory
		os.Chdir(currentDir)
		return errors.New("Unable to get absolute path of root directory")
	}

	// Store the absolute path
	pm.RootDir = absDir
	slog.Debug("Set root directory", "dir", pm.RootDir)

	return nil
}

func (pm *ProcessManager) ChangeExecuteDirectory(dir string) error {
	if dir == "" {
		return nil
	}

	targetDir := dir
	// Check if the path is relative (doesn't start with / or drive letter)
	if !filepath.IsAbs(dir) {
		// Combine with the root directory
		targetDir = filepath.Join(pm.RootDir, dir)
	}

	slog.Debug("Changing directory", "to", targetDir)
	err := os.Chdir(targetDir)
	if err != nil {
		return fmt.Errorf("Unable to change execute directory: %s: %w", targetDir, err)
	}
	return nil
}

func (pm *ProcessManager) RestoreRootDirectory() error {
	if pm.RootDir == "" {
		return nil
	}
	slog.Debug("Restoring directory to root", "dir", pm.RootDir)
	err := os.Chdir(pm.RootDir)
	if err != nil {
		return fmt.Errorf("Unable to restore root directory: %s: %w", pm.RootDir, err)
	}
	return nil
}

func printSubProcess(ctx context.Context, pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Println(scanner.Text())
			}
		}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled, try to close the pipe
		pipe.Close()
	case <-done:
		// Scanner finished naturally
	}

	if err := scanner.Err(); err != nil && err != io.EOF && !errors.Is(err, os.ErrClosed) {
		slog.Debug("Scanner error", "err", err)
	}
}
