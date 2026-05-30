//go:build !windows && !linux && !darwin

package engine

import "github.com/rjeczalik/notify"

// This platform has no event mappings. File watching will not function, but the
// package must still compile. Start() surfaces the lack of support as an error
// rather than calling os.Exit from a library init.
var (
	EventMap    = map[notify.Event]eventInfo{}
	CallbackMap = map[notify.Event]Event{}
)
