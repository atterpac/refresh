//go:build windows

package engine

import (
	"github.com/rjeczalik/notify"
)

var eventMap = map[notify.Event]EventInfo{
	notify.FileNotifyChangeLastWrite:  {Name: "FileNotifyChangeLastWrite", Reload: true},
	notify.FileActionModified:         {Name: "FileActionModified", Reload: true},
	notify.FileActionRenamedNewName:   {Name: "FileActionRenamedNewName", Reload: false},
	notify.FileActionRenamedOldName:   {Name: "FileActionRenamedOldName", Reload: false},
	notify.FileActionAdded:            {Name: "FileActionAdded", Reload: true},
	notify.FileActionRemoved:          {Name: "FileActionRemoved", Reload: false},
	notify.FileNotifyChangeAttributes: {Name: "FileNotifyChangeAttributes", Reload: false},
	notify.FileNotifyChangeSize:       {Name: "FileNotifyChangeSize", Reload: false},
	notify.FileNotifyChangeDirName:    {Name: "FileNotifyChangeDirName", Reload: false},
	notify.FileNotifyChangeFileName:   {Name: "FileNotifyChangeFileName", Reload: false},
	notify.FileNotifyChangeSecurity:   {Name: "FileNotifyChangeSecurity", Reload: false},
	notify.FileNotifyChangeCreation:   {Name: "FileNotifyChangeCreation", Reload: false},
	notify.FileNotifyChangeLastAccess: {Name: "FileNotifyChangeLastAccess", Reload: true},
	notify.Write:                      {Name: "Write", Reload: true},
	notify.Create:                     {Name: "Create", Reload: false},
	notify.Remove:                     {Name: "Remove", Reload: false},
	notify.Rename:                     {Name: "Rename", Reload: false},
}

var CallbackMap = map[notify.Event]Event{
	notify.FileNotifyChangeLastWrite:  ChangeLastWrite,
	notify.FileActionModified:         ActionModified,
	notify.FileActionRenamedNewName:   ActionRenamedNewName,
	notify.FileActionRenamedOldName:   ActionRenamedOldName,
	notify.FileActionAdded:            ActionAdded,
	notify.FileActionRemoved:          ActionRemoved,
	notify.FileNotifyChangeAttributes: ChangeAttributes,
	notify.FileNotifyChangeSize:       ChangeSize,
	notify.FileNotifyChangeDirName:    ChangeDirName,
	notify.FileNotifyChangeFileName:   ChangeFileName,
	notify.FileNotifyChangeSecurity:   ChangeSecurity,
	notify.FileNotifyChangeCreation:   ChangeCreation,
	notify.FileNotifyChangeLastAccess: ChangeLastAccess,
	notify.Write:                      Write,
	notify.Create:                     Create,
	notify.Remove:                     Remove,
	notify.Rename:                     Rename,
}
