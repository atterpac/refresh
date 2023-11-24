//go:build darwin

package engine

import (
	"github.com/rjeczalik/notify"
)

var eventMap = map[notify.Event]eventInfo{
	notify.Write:  {Name: "Write", Reload: true},
	notify.Create: {Name: "Create", Reload: false},
	notify.Remove: {Name: "Remove", Reload: false},
	notify.Rename: {Name: "Rename", Reload: false},
}

var CallbackMap = map[notify.Event]Event{
	notify.Write:  Write,
	notify.Create: Create,
	notify.Remove: Remove,
	notify.Rename: Rename,
}


