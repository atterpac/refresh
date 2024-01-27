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

#### Execute Lifecycle
In order to provide flexibility in your execute calls and project reloads refresh provides two declarations that are required in your execute list 

`"REFRESH_EXEC"` -> The next execute after `"REFRESH"` will be consider the "main" subprocess to refresh 

`"KILL_EXEC"` -> This declaration is replaced with the calls to kill the "main" subprocess, if one is not running this step is ignored

**THESE ARE REQUIRED INSIDE YOUR EXEC LIST TO PROPERLY FUNCTION**

These declarations let refresh know when you would like to kill the stale process thats been out of date due to a filechange and when to start your new version for example

` "go build -o app", "KILL_STALE", "REFRESH", "./app" ` -> This list would detect a change in the watched files build the new version kill the old one and start the new version (the most likely case)

` "KILL_STALE", "go build -o app", "REFRESH", "./app" ` -> This list would detect a change in the watched files Kill the now stale version build a new one and run it

Whatever command after REFRESH is considered your "main" subprocess and the one that is tracked inside of refresh

## Embedding into your dev project
There can be some uses where you might want to start a watcher internally or for a tool for development refresh provides a function `NewEngineFromOptions` which takes an `engine.Config` and allows for the `engine.Start()` function

Using refresh as a library also opens the ability to add a [Callback](https://github.com/atterpac/refresh#reload-callback) function that is called on every FS notification

### Structs
```go
type Config struct {
	RootPath        string   `toml:"root_path"`
	BackgroundExec  string   `toml:"background_exec"` // Execute that stays running and is unaffected by any reloads npm run dev for example
    BackgroundCheck bool     `toml:"background_check"`
	Ignore          Ignore   `toml:"ignore"`
	ExecList        []string `toml:"exec_list"` // See [Execute Lifecycle](https://github.com/atterpac/refresh#execute-lifecycle)
	LogLevel        string   `toml:"log_level"`
	Debounce        int      `toml:"debounce"`
	Callback        func(*EventCallback) EventHandle
	Slog            *slog.Logger
}

type Ignore struct {
	Dir        []string `toml:"dir"` // Specfic directory to ignore ie; node_modules
	File       []string `toml:"file"` // Specific file to ignore 
	WatchExten []string `toml:"extension"` // Extensions to watch NOT ignore, ie; `*.go, *.js` would ignore any file that is not go or javascript
    GitIgnore  bool     `toml:"git_ignore"` // When true will check for a .gitignore in the root directory and add all entries to the ignore
}

type Execute struct {
	Cmd          string      `toml:"cmd" yaml:"cmd"`                     // Execute command
	ChangeDir    string      `toml:"dir" yaml:"dir"`                     // If directory needs to be changed to call this command relative to the root path
	IsBlocking   bool        `toml:"blocking" yaml:"blocking"`           // Should the following executes wait for this one to complete
	IsPrimary    bool        `toml:"primary" yaml:"primary"`             // Only one primary command can be run at a time
	DelayNext    int         `toml:"delay_next" yaml:"delay_next"`       // Delay in milliseconds before running command
}
```

### Example
For a functioning example see ./example and run main.go below describes what declaring an engine could look like
```go
import ( // other imports
     "github.com/atterpac/refresh/engine"
    )

func main () {
    // Setup your watched exensions and any ignored files or directories
	ignore := engine.Ignore{
        // Can use * wildcards per usual filepath pattern matching (including /**/) 
        // ! denoted an invert in this example ignoring any extensions that are not *.go
        WatchedExten: []string{"*.go"}, // Ignore all files that are not go
		File:         []string{"ignore*.go"},  // Pattern match to ignore any golang files that start with ignore
		Dir:          []string{".git","*/node_modules", "!api/*"}, // Ignore .git and any node_modules in the directory or anything not within the api directory
        IgnoreGit: true, // .gitignore sitting in the root directory? set this to true to automatially ignore those files
	}
    // Build execute structs
	tidy := engine.Execute{
		Cmd:        "go mod tidy",
		IsBlocking: true, // Next command should wait for this to finish
	}
	build := engine.Execute{
		Cmd:        "go build -o ./bin/myapp",
		IsBlocking: true, // Wait to kill (next step) until the new binary is built
	}
    // Provided KILL_STALE will tell refresh when you would like to remove the stale process to prepare to launch the new one
	kill := engine.KILL_STALE 
    // Primary process usually runs your binary
	run := engine.Execute{
        ChangeDir:   "./bin", // Change directory to call command in
		Cmd:        "./myapp",
		IsBlocking: false, // Should not block because it doesnt finish until Killed by refresh
		IsPrimary:  true, // This is the main process refersh is rerunning so denoting it as primary
	}
    // Create config to pass into refresh.NewEngineFromConfig()
	config := engine.Config{
		RootPath: "./test",
		// Below is ran when a reload is triggered before killing the stale version
		Ignore:     ignore,
		Debounce:   1000, // Time in ms to ignore repitive reload triggers usually caused by an OS creating multiple write/rename events for a singular change
		LogLevel:   "debug", // debug | info | warn | error | mute -> surpresses all logs to the stdOut
        Callback:   RefreshCallback, // func(*engine.Callback) refresh.EventHandle {}
		ExecStruct: []refresh.Execute{tidy, build, kill, run},
        // Alternatively for easier config but less control over executes
        // ExecList: []string{"go mod tidy", "go build -o ./myapp", refresh.KILL_EXEC, refresh.REFRESH_EXEC, "./myapp"}
        // All calls will be blocking with the exception of the call after REFRESH
        // Both KILL_EXEC and REFRESH_EXEC are **REQUIRED** for refresh to function properly
        // engine.KILL_EXEC denotes when the stale process should be killed
        // engine.REFRESH_EXEC denotes the next execute is "primary"
		Slog:       nil, // Optionally provide a slog interface
                         // if nil a default will be provided
                         // If provided stdout will not be piped through refresh
	}

	engine, err := refresh.NewEngineFromConfig(config)
    if err != nil {
        //Handle err
    }
	err = engine.Start()
    if err != nil {
        // Start will return an error when a user hits ctrl-c after it gracefully kills the processes
    }

	// Stop monitoring files and kill child processes
	engine.Stop()
}

func RefreshCallback(e *engine.EventCallback) engine.EventHandle {
    switch e.Type {
        case engine.Create:
            return engine.EventIgnore
        case engine.Write:
                if e.Path == "test/monitored/ignore.go" {
                    return engine.EventBypass
                }
                return engine.EventContinue
        case engine.Remove:
                    return engine.EventContinue
                        // Other cases as needed ...
    }
    return engine.EventContinue
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

`engine.EventContinue` continues with the reload process as normal and follows the refresh ruleset defined in the config

`engine.EventBypass` disregards all config rulesets and restarts the exec process

`engine.EventIgnore` ignores the event and continues monitoring

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
	case engine.Create:
		// Continue with reload process based on configured ruleset
		return refresh.EventContinue
	case engine.Write:
		// Ignore a file that would normally trigger a reload based on config
		if e.Path == "path/to/watched/file" {
			return engine.EventIgnore
		}
		// Continue with reload ruleset but add some extra logs/logic
		fmt.Println("File Modified: %s", e.Path)	
		return engine.EventContinue
	case engine.Remove:
		// refresh will ignore this event by default
		// Return EventBypass to force reload process
		return engine.EventBypass
	}
	return engine.EventContinue
}
```
### Config File

If you would prefer to load from a [config](https://github.com/Atterpac/refresh#config-file) file rather than building the structs you can use 
```go
engine.NewEngineFromTOML("path/to/toml")
engine.SetLogger(//Input slog.Logger)
```
#### Example Config
```toml
[config]
# Relative to this files location
root_path = "./"
# debug | info(default) | warn | error | mute
log_level = "info" 
# Debounce setting for ignoring reptitive file system notifications
debounce = 1000 # Milliseconds
# Sets what files the watcher should ignore
background_check = true

[config.ignore]
# Ignore follows normal pattern matching including /**/
# Directories to ignore
dir = [".git", "node_modules", "newdir"]
# Files to ignore
file = [".DS_Store", ".gitignore", ".gitkeep", "newfile.go", "*ignoreme*"]
# File extensions to watch
watched_extensions = ["*.go"]
# Add .gitignore paths to ignore
git_ignore = true

# Runs process in the background and doesnt restart when a refresh is triggered
# Vite dev and other processes take varying durations and the following commands might rely on them being "complete"
# This is where setting background_check = true and using a callback in golang library to confirm its state
[config.background]
cmd="vite dev"

# Execute structs
# dir is used to change the working directory to execute into
# cmd is the command to be executed
# primary denotes this is the process refresh should be tracking to kill on reload
# blocking denotes wether the next execute should wait for it to complete ie; build the application and when its done run it
# KILL_STALE is required to be ran at any point before the primary is executed this kills the previous version of the application
[[config.executes]]
cmd="go mod tidy"
primary=false
blocking=true

[[config.executes]]
cmd="go build -o ./bin/app"
blocking=true

[[config.executes]]
cmd="KILL_STALE"

[[config.executes]]
dir="./bin"
cmd="./app"
primary=true
```

### Background Check Callback
There are instances where you want to wait for the "build" steps for something like vite or a server connection that could take a varying amount
of time to reach a ready state. Refresh adds `engine.AttachBackgroundCallback()` which will hault the execute commands until the callback returns 
true (or false for error and shutting down). This could be used along side a ping to the vite port for example to ensure it is reached before 
running commands that rely on it. This requires 2 things

-  A callback function that is `func() bool` and returns true when ready and false when errored or exited 
-  Attaching the callback via `engine.AttachBackgroundCallback()` prior to running `engine.Start()`

#### Flags
This method is possible but not the most verbose and controlled way to use refresh

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
### Alternatives
Refresh not for you? Here are some popular hot reload alternatives

- [Air](https://github.com/cosmtrek/air)
- [Realize](https://github.com/oxequa/realize)
- [Fresh](https://github.com/gravityblast/fresh)
