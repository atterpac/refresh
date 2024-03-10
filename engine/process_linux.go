//go:build !windows

package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type Process struct {
	Exec       string
	Blocking   bool
	Background bool
	Primary    bool
	cmd        *exec.Cmd
	done       chan struct{}
}

type ProcessManager struct {
	processes []*Process
	mu 		 sync.Mutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make([]*Process, 0),
	}
}

func (pm *ProcessManager) StartProcesses() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.processes) == 0 {
		slog.Warn("No Processes to Start")
		return
	}
	for _, p := range pm.processes {
		cmd := exec.Command("sh", "-c", p.Exec)
		p.cmd = cmd
		p.done = make(chan struct{})

		if p.Primary {
			cmd.Stderr = os.Stderr
			pipe, err := cmd.StdoutPipe()
			if err != nil {
				slog.Error("Getting log pipe", "err", err.Error())
			}
			go printSubProcess(pipe)
		}
		// if !firstRun {
		// 	pm.KillProcesses(true)
		// }
		var err error
		slog.Warn("Starting Process", "exec", p.Exec)
		if p.Blocking {
			err = cmd.Run()
		} else {
			err = cmd.Start()
			go func(p *Process) {
				cmd.Wait()
				pm.mu.Lock()
				close(p.done)
				pm.mu.Unlock()
			}(p)
		}

		if err != nil {
			slog.Error("Running Command", "exec", p.Exec, "err", err)
		}
	}
}

func (pm *ProcessManager) KillProcesses(ignoreBackground bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	slog.Warn("Killing Processes")
	for _, p := range pm.processes {
		slog.Info("Attempting to kill process", "exec", p.Exec)
		if p.Background && ignoreBackground {
			continue
		}
		if p.Primary {
			slog.Warn("Killing Primary Process", "exec", p.Exec, "pid", p.cmd.Process.Pid)
		}
		if p.cmd != nil && p.cmd.Process != nil {
			select {
			case <-p.done:
				slog.Warn("Process already exited", "exec", p.Exec)
				continue
			default:
				err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
				if err != nil {
				} else {
					slog.Warn("Killed Processes")
				}
			}
		}
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

// type Process struct {
// 	Process *os.Process
// 	pgid    int
// 	Output  bytes.Buffer
// 	Error   bytes.Buffer
// }
//
// // Start process with exec command and a root path to call it in
// func (engine *Engine) startPrimaryProcess(runString string) (string, error) {
//     var err error
//     var process Process
//
//     slog.Debug("Starting Primary")
//
//     cmd := generateExec(runString)
//
//     // If an external slog is provided do not pipe stdout to the engine
//     if !engine.Config.externalSlog {
//         cmd.Stderr = os.Stderr
//         engine.ProcessLogPipe, err = cmd.StdoutPipe()
//         if err != nil {
//             slog.Error("Getting log pipe", "err", err.Error())
//             return "", err
//         }
//         defer engine.ProcessLogPipe.Close()
//     }
//
//     attachNewProcessGroup(cmd)
//
//     err = cmd.Start()
//     if err != nil {
//         slog.Error("Starting Primary", "err", err.Error())
//         return "", err
//     }
//
//     if !engine.Config.externalSlog {
//         slog.Debug("Starting log pipe")
//         go printSubProcess(engine.ProcessLogPipe)
//     }
//
//     err = cmd.Wait()
//     if err != nil {
//         slog.Error("Running Primary", "err", err.Error())
//         return "", err
//     }
//
//     process.Process = cmd.Process
//     process.pgid, err = syscall.Getpgid(cmd.Process.Pid)
//     if err != nil {
//         slog.Error("Getting process group id", "err", err.Error())
//         return "", err
//     }
//
//     engine.ProcessSync.Lock()
//     engine.ProcessMap[runString] = process
//     engine.ProcessSync.Unlock()
//
//     return runString, nil
// }
//
// func (engine *Engine) startBackgroundProcess(runString string) (Process, error) {
// 	var err error
// 	var process Process
// 	cmd := generateExec(runString)
// 	// Let Process run in background
// 	cmd.Stdout = &process.Output
// 	cmd.Stderr = &process.Error
// 	attachNewProcessGroup(cmd)
// 	cmdErr := cmd.Start()
// 	if cmdErr != nil {
// 		slog.Error("Background Execute failed", "err", cmdErr)
// 		return Process{}, cmdErr
// 	}
// 	slog.Debug("Complete Exec Command", "cmd", runString)
// 	// Get PGID
// 	process.pgid, err = syscall.Getpgid(cmd.Process.Pid)
// 	process.Process = cmd.Process
// 	if err != nil {
// 		slog.Error("Getting process group id", "err", err.Error())
// 		return Process{}, err
// 	}
// 	slog.Debug("Process group id", "pgid", process.pgid)
// 	return process, nil
// }
//
// func (engine *Engine) killProcess(key string) bool {
// 	engine.ProcessSync.Lock()
// 	defer engine.ProcessSync.Unlock()
// 	process, ok := engine.ProcessMap[key]
// 	if !ok {
// 		slog.Warn("No Process Found", "key", key)
// 		return false
// 	}
// 	slog.Warn("Killing process", "process", process)
// 	err := syscall.Kill(-process.pgid, syscall.SIGKILL)
// 	if err != nil {
// 		slog.Error("Killing process", "err", err)
// 		return false
// 	}
// 	delete(engine.ProcessMap, key)
// 	return true
// }
//
// func attachNewProcessGroup(cmd *exec.Cmd) *exec.Cmd {
// 	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
// 	return cmd
// }
