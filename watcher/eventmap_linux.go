//go:build linux

package watcher

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
