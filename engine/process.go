package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
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

func (e *Engine) StartProcesses() {
	slog.Warn("Restarting processes...", "len", len(e.ProcessManager.processes))
	pm := e.ProcessManager
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.processes) == 0 {
		slog.Warn("No Processes to Start")
		return
	}
	slog.Warn("Starting Processes", "count", len(pm.processes))
	for _, p := range pm.processes {
		if p.Exec == "KILL_STALE" {
			continue
		}
		if !firstRun && p.Background {
			slog.Debug("Leaving background process running", "exec", p.Exec)
			continue
		}
		slog.Debug("Starting Process", "exec", p.Exec, "blocking", p.Blocking, "primary", p.Primary, "background", p.Background, "firstRun", firstRun)
		cmd := generateExec(p.Exec)
		p.cmd = cmd
		if p.Primary {
			if !firstRun {
				pm.KillProcesses(true)
				slog.Debug("Processes killed")
				time.Sleep(100 * time.Millisecond)
			}
			if !e.Config.externalSlog {
				cmd.Stderr = os.Stderr
				e.ProcessLogPipe, _ = cmd.StdoutPipe()
				go printSubProcess(e.ProcessLogPipe)
			}
		}
		var err error
		slog.Warn("Starting Process", "exec", p.Exec)
		if p.Blocking {
			err = cmd.Run()
		} else {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			err = cmd.Start()
			p.pgid, _ = syscall.Getpgid(cmd.Process.Pid)
			go func() {
				cmd.Wait()
			}()
		}
		if err != nil {
			slog.Error("Running Command", "exec", p.Exec, "err", err)
		}
	}
}
