
package engine

import ( 
	"time"
)

type eventInfo struct {
	Name   string
	Reload bool
}

// Called whenever a change is detected in the filesystem
// By default we ignore file rename/remove and a bunch of other events that would likely cause breaking changes on a reload  see eventmap_[oos].go for default rules
type EventCallback struct {
	Type Event    // Event enum
	Time time.Time // time.Now() when event was triggered
	Path string    // Full path to the modified file
}

// EventHandle is used to determine how to handle a reload callback 
type EventHandle int
const (
	EventContinue EventHandle = iota
	EventBypass
	EventIgnore
)

// Event is used to determine what type of event was triggered
type Event int
const (
	Create Event = iota
	Write
	Remove
	Rename
	// Windows Specific Actions
	ActionModified
	ActionRenamedNewName
	ActionRenamedOldName
	ActionAdded
	ActionRemoved
	ChangeLastWrite
	ChangeAttributes
	ChangeSize
	ChangeDirName
	ChangeFileName
	ChangeSecurity
	ChangeCreation
	ChangeLastAccess
	// Linux Specific Actions
	InCloseWrite
	InModify
	InMovedTo
	InMovedFrom
	InCreate
	InDelete
)
