//go:build linux

package engine

import (
	"github.com/rjeczalik/notify"
)

var eventMap = map[notify.Event]EventInfo{
	notify.InCloseWrite: {Name: "InCloseWrite", Reload: true},
	notify.InModify:     {Name: "InModify", Reload: true},
	notify.InMovedTo:    {Name: "InMovedTo", Reload: true},
	notify.InMovedFrom:  {Name: "InMovedFrom", Reload: true},
	notify.InCreate:     {Name: "InCreate", Reload: true},
	notify.InDelete:     {Name: "InDelete", Reload: true},
	notify.Write:        {Name: "Write", Reload: true},
	notify.Create:       {Name: "Create", Reload: false},
	notify.Remove:       {Name: "Remove", Reload: false},
	notify.Rename:       {Name: "Rename", Reload: false},
}

var CallbackMap = map[notify.Event]Event{
	notify.InCloseWrite: InCloseWrite,
	notify.InModify:     InModify,
	notify.InMovedTo:    InMovedTo,
	notify.InMovedFrom:  InMovedFrom,
	notify.InCreate:     InCreate,
	notify.InDelete:     InDelete,
	notify.Write:        Write,
	notify.Create:       Create,
	notify.Remove:       Remove,
	notify.Rename:       Rename,
}

