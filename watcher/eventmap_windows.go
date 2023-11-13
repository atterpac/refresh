//go:build windows

package watcher

import (
	"github.com/rjeczalik/notify"
)

var eventMap = map[notify.Event]EventInfo{
	notify.FileNotifyChangeLastWrite:  {Name: "FileNotifyChangeLastWrite", Reload: true},
	notify.FileActionModified:         {Name: "FileActionModified", Reload: true},
	notify.FileActionRenamedNewName:   {Name: "FileActionRenamedNewName", Reload: true},
	notify.FileActionRenamedOldName:   {Name: "FileActionRenamedOldName", Reload: true},
	notify.FileActionAdded:            {Name: "FileActionAdded", Reload: true},
	notify.FileActionRemoved:          {Name: "FileActionRemoved", Reload: true},
	notify.FileNotifyChangeAttributes: {Name: "FileNotifyChangeAttributes", Reload: true},
	notify.FileNotifyChangeSize:       {Name: "FileNotifyChangeSize", Reload: true},
	notify.FileNotifyChangeDirName:    {Name: "FileNotifyChangeDirName", Reload: true},
	notify.FileNotifyChangeFileName:   {Name: "FileNotifyChangeFileName", Reload: true},
	notify.FileNotifyChangeSecurity:   {Name: "FileNotifyChangeSecurity", Reload: true},
	notify.FileNotifyChangeCreation:   {Name: "FileNotifyChangeCreation", Reload: true},
	notify.FileNotifyChangeLastAccess: {Name: "FileNotifyChangeLastAccess", Reload: true},
	notify.Write:                      {Name: "Write", Reload: true},
	notify.Create:                     {Name: "Create", Reload: false},
	notify.Remove:                     {Name: "Remove", Reload: false},
	notify.Rename:                     {Name: "Rename", Reload: false},
}
