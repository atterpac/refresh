//go:build darwin

package watcher

import (
	"github.com/rjeczalik/notify"
)

var eventMap = map[notify.Event]EventInfo{
	notify.Write:  {Name: "Write", Reload: true},
	notify.Create: {Name: "Create", Reload: false},
	notify.Remove: {Name: "Remove", Reload: false},
	notify.Rename: {Name: "Rename", Reload: false},
}
