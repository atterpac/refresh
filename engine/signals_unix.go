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
// pokes toggle, which the supervisor loop interprets as flip-paused. The send is
// non-blocking so a press while a toggle is already queued is coalesced.
func (engine *Engine) trapControlSignals(toggle chan<- struct{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTSTP)
	go func() {
		for range ch {
			select {
			case toggle <- struct{}{}:
			default:
			}
		}
	}()
}
