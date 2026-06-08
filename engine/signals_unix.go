//go:build linux || darwin

package engine

import (
	"os"
	"os/signal"
	"syscall"
)

// trapControlSignals repurposes the terminal suspend key (Ctrl+Z / SIGTSTP) as a
// pause/resume toggle. Trapping SIGTSTP overrides the kernel's default suspend
// disposition, so the process is never actually stopped; each delivery instead
// flips the engine's pause state. togglePause is itself non-blocking, so a press
// while the supervisor is busy is never lost behind a blocked signal goroutine.
func (engine *Engine) trapControlSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTSTP)
	go func() {
		for range ch {
			engine.togglePause()
		}
	}()
}
