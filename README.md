## Refresh Hot Reload
Refresh is CLI tool for hot reloading your codebase based on file system changes using [notify](https://github.com/rjeczalik/notify) with the ablity to use as a golang library in your own projects.

While refresh was built for reloading codebases it can be used to execute terminal commands based on file system changes

## Key Features
- Based on [Notify](https://github.com/rjeczalik/notify) to allievate common problems with popular FS libraries on mac that open a listener per file by using apples built-in FSEvents.
- Allows for customization via code / config file / cli flags
- Extended customization when used as a library using reloadCallback to bypass refresh rulesets and add addtional logic/logging on your applications end
- Default slogger built in with the ablity to mute logs as well as pass in your own slog handler to be used in app
- MIT licensed

## Install
Installing via go CLI is the easiest method .tar.gz files per platform are available via github releases
```bash
go install github.com/atterpac/refresh/cmd/refresh@latest
```
Alternative if you wish to use as a package and not a cli
```bash
go get github.com/atterpac/refresh
```
## Usage

#### Flags
`-p` Root path that will be watched and commands will be executed in typically this is './'

`-be` Command to be called before the exec command for example `go mod tidy`

`-w` Flag to decide wether the exec process should wait on the pre exec to complete

`-e` Command to be called when a modification is detected for example `go run main.go`

`-ae` Command to b be called when a modifcation is detected after the main process closes 

`-l` Log Level to display options can include `"debug", "info","warn","error", "mute"`

`-f` path to a TOML config file see [Config File](https://github.com/atterpac/refresh#config-file) for details on the format of config

`-id` Ignore directories provided as a comma-separated list

`-if` Ignore files provided as a comma-separated list

`-ie` Ignore extensions provided as a comma-separated list

`-d` Debounce timer in milliseconds, used to ignore repetitive system

#### Example
```bash
refresh -p ./ -e "go run main.go" -be "go mod tidy" -ae "rm ./main" -l "debug" -id ".git, node_modules" -if ".env" -ie ".db, .sqlite" -d 500
```

## Embedding into your dev project
There can be some uses where you might want to start a watcher internally or for a tool for development refresh provides a function `NewEngineFromOptions` which takes an `refresh.Config` and allows for the `engine.Start()` function

Using refresh as a library also opens the ability to add a [Callback](https://github.com/atterpac/refresh#reload-callback) function that is called on every FS notification

### Structs
```go
type Config struct {
	RootPath     string `toml:"root_path"`
	PreExec      string `toml:"pre_exec"`
    PreWait      bool   `toml:"pre_wait"`
	ExecCommand  string `toml:"exec_command"`
	PostExec     string `toml:"post_exec"`
	Ignore       Ignore `toml:"ignore"`
	LogLevel     string `toml:"log_level"`
	Debounce     int    `toml:"debounce"`
	Slog         *slog.Logger
	ExternalSlog bool
}

type Ignore struct {
	Dir       map[string]bool `toml:"dir"`
	File      map[string]bool `toml:"file"`
	Extension map[string]bool `toml:"extension"`
    GitIgnore bool            `toml:"git_ignore"`
    Git       map[string]bool // Genrated on start
}
```

### Example
For a functioning example see ./example and run main.go below describes what declaring an engine could look like
```go
import ( // other imports
     refresh "github.com/atterpac/refresh/engine"
    )

func main () {
	ignore := refresh.Ignore{
        // Can use * wildcards per usual 
        // ! denoted an invert in this example ignoring any extensions that are not *.go
		File:      map[string]bool{"ignore*.go":true, ".gitignore"},
		Dir:       map[string]bool{".git":true,"*/node_modules":true},
		Extension: map[string]bool{"!*.go":true},
        IgnoreGit: true, // .gitignore sitting in the root directory? set this to true to automatially ignore those files
	}
	config := refresh.Config{
		RootPath:    "./subExecProcess",
		ExecCommand: "go run main.go",
		LogLevel:    "info", // debug | info | warn | error | mute (discards all logs)
		Ignore:      ignore,
		Debounce:    1000,
		Slog: nil, // Optionally provide a slog interface
                  // if nil a default will be provided
                  // If provided stdout will not be piped through refresh

		// Optionally provide a callback function to be called upon file notification events
        	Callback: func(*EventCallback) EventHandle 
	}
	engine := refresh.NewEngineFromConfig(config)
	engine.Start()

	// Stop monitoring files and kill child processes
	engine.Stop()
}
```
### Reload Callback

#### Event Types
The following are all the file system event types that can be passed into the callback functions.
Important to note that some actions only are emitted are certain OSs and you may have to handle those if you wish to bypass refresh rulesets 
```go
const (
    // Base Actions
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

// Used as a response to the Callback 
const (
	EventContinue EventHandle = iota
	EventBypass
	EventIgnore
)
```

#### Callback Function

Below describes the data that you recieve in the callback function as well as an example of how this could be used.

Callbacks should return an refresh.EventHandle

`refresh.EventContinue` continues with the reload process as normal and follows the refresh ruleset defined in the config

`refresh.EventBypass` disregards all config rulesets and restarts the exec process

`refresh.EventIgnore` ignores the event and continues monitoring

```go
// Called whenever a change is detected in the filesystem
// By default we ignore file rename/remove and a bunch of other events that would likely cause breaking changes on a reload  see eventmap_[oos].go for default rules
type EventCallback struct {
	Type Event  // Type of Notification (Write/Create/Remove...)
	Time time.Time // time.Now() when event was triggered
	Path string    // Relative path based on root if root is ./myProject paths start with "myProject/..."
}
// Available returns from the Callback function
const (
	EventContinue EventHandle = iota
	EventBypass
	EventIgnore
)

func ExampleCallback(e refresh.EventCallback) refresh.EventHandle {
	switch e.Type {
	case refresh.Create:
		// Continue with reload process based on configured ruleset
		return refresh.EventContinue
	case refresh.Write:
		// Ignore a file that would normally trigger a reload based on config
		if e.Path == "path/to/watched/file" {
			return refresh.EventIgnore
		}
		// Continue with reload ruleset but add some extra logs/logic
		fmt.Println("File Modified: %s", e.Path)	
		return EventContinue
	case refresh.Remove:
		// refresh will ignore this event by default
		// Return EventBypass to force reload process
		return refresh.EventBypass
	}
	return refresh.EventContinue
}
```
### Config File

If you would prefer to load from a [config](https://github.com/Atterpac/refresh#config-file) file rather than building the structs you can use 
```go
refresh.NewEngineFromTOML("path/to/toml")
```
#### Example Config
```toml
[config]
# Relative to this files location
root_path = "./"
# Runs prior to the exec command starting
pre_exec = "go mod tidy"
pre_wait = true
# Command to run on reload
exec_command = "go run main.go"
# Runs when a file reload is triggered after killing the previous process
post_exec = ""
# debug | info | warn | error | mute
# Defaults to Info if not provided
log_level = "info" 
# Debounce setting for ignoring reptitive file system notifications
debounce = 1000 # Milliseconds
# Sets what files the watcher should ignore
[config.ignore]
# Directories to ignore
dir = [".git", "node_modules", "newdir"]
# Files to ignore
file = [".DS_Store", ".gitignore", ".gitkeep", "newfile.go"]
# File extensions to ignore
extension = [".db", ".sqlite"]
# Add .gitignore paths to ignore
git_ignore = true
```

### Alternatives
Refresh not for you? Here are some popular hot reload alternatives

- [Air](https://github.com/cosmtrek/air)
- [Realize](https://github.com/oxequa/realize)
- [Fresh](https://github.com/gravityblast/fresh)
