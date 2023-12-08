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

`-w` Flag to decide wether the exec process should wait on the pre exec to complete

`-e` Commands to be called when a modification is detected in the form of a comma seperated list required refresh declrations 
    
**See [Execute Lifecycle](https://github.com/atterpac/refresh#execute-lifecycle) for more details**

`-l` Log Level to display options can include `"debug", "info","warn","error", "mute"`

`-f` path to a TOML config file see [Config File](https://github.com/atterpac/refresh#config-file) for details on the format of config

`-id` Ignore directories provided as a comma-separated list

`-if` Ignore files provided as a comma-separated list

`-ie` Ignore extensions provided as a comma-separated list

`-d` Debounce timer in milliseconds, used to ignore repetitive system

#### Example
```bash
refresh -p ./ -e "go mod tidy, go build -o ./myapp, KILL_STALE, REFRESH, ./myapp" -l "debug" -id ".git, node_modules" -if ".env" -ie ".db, .sqlite" -d 500
```

#### Execute Lifecycle
In order to provide flexibility in your execute calls and project reloads refresh provides two declarations that are required in your execute list 

`"REFRESH"` -> The next execute after `"REFRESH"` will be consider the "main" subprocess to refresh 

`"KILL_STALE"` -> This declaration is replaced with the calls to kill the "main" subprocess, if one is not running this step is ignored

**THESE ARE REQUIRED INSIDE YOUR EXEC LIST TO PROPERLY FUNCTION**

These declarations let refresh know when you would like to kill the stale process thats been out of date due to a filechange and when to start your new version for example

` "go build -o app", "KILL_STALE", "REFRESH", "./app" ` -> This list would detect a change in the watched files build the new version kill the old one and start the new version (the most likely case)

` "KILL_STALE", "go build -o app", "REFRESH", "./app" ` -> This list would detect a change in the watched files Kill the now stale version build a new one and run it

Whatever command after REFRESH is considered your "main" subprocess and the one that is tracked inside of refresh

## Embedding into your dev project
There can be some uses where you might want to start a watcher internally or for a tool for development refresh provides a function `NewEngineFromOptions` which takes an `refresh.Config` and allows for the `engine.Start()` function

Using refresh as a library also opens the ability to add a [Callback](https://github.com/atterpac/refresh#reload-callback) function that is called on every FS notification

### Structs
```go
type Config struct {
	RootPath       string   `toml:"root_path"`
	BackgroundExec string   `toml:"background_exec"` // Execute that stays running and is unaffected by any reloads npm run dev for example
	Ignore         Ignore   `toml:"ignore"`
	ExecList       []string `toml:"exec_list"` // See [Execute Lifecycle](https://github.com/atterpac/refresh#execute-lifecycle)
	LogLevel       string   `toml:"log_level"`
	Debounce       int      `toml:"debounce"`
	Callback       func(*EventCallback) EventHandle
	Slog           *slog.Logger
}

type Ignore struct {
	Dir        []string `toml:"dir"` // Specfic directory to ignore ie; node_modules
	File       []string `toml:"file"` // Specific file to ignore 
	WatchExten []string `toml:"extension"` // Extensions to watch NOT ignore, ie; `*.go, *.js` would ignore any file that is not go or javascript
    GitIgnore  bool     `toml:"git_ignore"` // When true will check for a .gitignore in the root directory and add all entries to the ignore
}

type Execute struct {
	Cmd        string
	IsBlocking bool // Should the next command wait for this command to finish 
	IsPrimary  bool // Only one primary command can be run at a time
}
```

### Example
For a functioning example see ./example and run main.go below describes what declaring an engine could look like
```go
import ( // other imports
     refresh "github.com/atterpac/refresh/engine"
    )

func main () {
    // Setup your watched exensions and any ignored files or directories
	ignore := refresh.Ignore{
        // Can use * wildcards per usual filepath pattern matching (including /**/) 
        // ! denoted an invert in this example ignoring any extensions that are not *.go
        WatchedExten: []string{"*.go"}, // Ignore all files that are not go
		File:         []string{"ignore*.go"},  // Pattern match to ignore any golang files that start with ignore
		Dir:          []string{".git","*/node_modules", "!api/*"}, // Ignore .git and any node_modules in the directory or anything not within the api directory
        IgnoreGit: true, // .gitignore sitting in the root directory? set this to true to automatially ignore those files
	}
    // Build execute structs
	tidy := refresh.Execute{
		Cmd:        "go mod tidy",
		IsBlocking: true, // Next command should wait for this to finish
		IsPrimary:  false,
	}
	build := refresh.Execute{
		Cmd:        "go build -o ./bin/myapp",
		IsBlocking: true,
		IsPrimary:  false,
	}
    // Provided KILL_STALE will tell refresh when you would like to remove the out of date version to prepare to launch the new one
	kill := refresh.KILL_STALE
	run := refresh.Execute{
		Cmd:        "./bin/myapp",
		IsBlocking: false, // Should not block because it doesnt finish until Killed by refresh
		IsPrimary:  true, // This is the main process refersh is rerunning so denoting it as primary
	}

    // Config to pass into refresh.NewEngineFromConfig()
	config := refresh.Config{
		RootPath: "./test",
		// Below is ran when a reload is triggered before killing the stale version
		Ignore:     ignore,
		Debounce:   1000,
		LogLevel:   "debug",
        Callback:   RefreshCallback, // func(*refresh.Callback) refresh.EventHandle {}
		ExecStruct: []refresh.Execute{tidy, build, kill, run},
        // Alternatively for easier config but less control over executes
        // KILL_STALE and REFRESH are required for the ExecList to function
        // ExecList: []string{"go mod tidy", "go build -o ./myapp", "KILL_STALE", "REFRESH", "./myapp"}
		Slog:       nil, // Optionally provide a slog interface
                         // if nil a default will be provided
                         // If provided stdout will not be piped through refresh
	}

	engine := refresh.NewEngineFromConfig(config)
	engine.Start()

	// Stop monitoring files and kill child processes
	engine.Stop()
}

func RefreshCallback(e *refresh.EventCallback) refresh.EventHandle {
    switch e.Type {
        case refresh.Create:
            return refresh.EventIgnore
        case refresh.Write:
                if e.Path == "test/monitored/ignore.go" {
                    return refresh.EventBypass
                }
                return refresh.EventContinue
        case refresh.Remove:
                    return refresh.EventContinue
                        // Other cases as needed ...
    }
    return refresh.EventContinue
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
// By default we ignore file rename/remove and a bunch of other events that would likely cause breaking changes on a reload see eventmap_[oos].go for default rules
type EventCallback struct {
	Type Event  // Type of Notification (Write/Create/Remove...)
	Time time.Time // time.Now() when event was triggered
	Path string    // Relative path based on root if root is ./myProject paths start with "myProject/..."
}
// Available returns from the Callback function
const (
	EventContinue EventHandle = iota // Continue with refresh ruleset 
	EventBypass // Bypass all rule and reload the process
	EventIgnore // Force Ignore event and continue watching 
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
# ordered list of execs to run including the required KILL_STALE, and REFERSH
exec_list = ["go mod tidy", "go build -o ./app", "KILL_STALE", "REFRESH", "./app"
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
# File extensions to watch
watched_extensions = [".db", ".sqlite"]
# Add .gitignore paths to ignore
git_ignore = true
```

### Alternatives
Refresh not for you? Here are some popular hot reload alternatives

- [Air](https://github.com/cosmtrek/air)
- [Realize](https://github.com/oxequa/realize)
- [Fresh](https://github.com/gravityblast/fresh)
