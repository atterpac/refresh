//go:build windows

package engine

// trapControlSignals is a no-op on Windows: there is no SIGTSTP/Ctrl+Z suspend
// signal to repurpose as a pause/resume toggle, so enabling Config.TrapSuspend
// has no effect on this platform.
func (engine *Engine) trapControlSignals(chan<- struct{}) {}
