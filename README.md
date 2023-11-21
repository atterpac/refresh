# :construction: EARLY DEVELOPMENT :construction:
This is a small tool I have built that is largely untested off of my machine. You are welcome to try it and if you notice any issues report them on the github and I will look into them.

## HOTATO Hot Reload
Hotato (hot potato) is CLI tool for hot reloading your codebase based on file system changes using [notify](https://github.com/rjeczalik/notify) with the ablity to use as a golang library in your own projects.

## Key Features
- Based on [Notify](https://github.com/rjeczalik/notify) to allievate common problems with popular FS libraries on mac that open a listener per file by using apples built-in FSEvents.
- Allows for customization via code / config file / cli flags
- Extended customization when used as a library using reloadCallback to bypass gotato rulesets and add addtional logic/logging on your applications end
- Default slogger built in with the ablity to mute logs as well as pass in your own slog handler to be used in app
- MIT licensed

## Install
Installing via go CLI is the easiest method more methods are on the list
```bash
go install github.com/atterpac/hotato/cmd/hotato@latest
```
Alternative if you wish to use as a package and not a cli
```bash
go get github.com/atterpac/hotato
```
## Usage

#### Flags
`-p` Root path that will be watched and commands will be executed in typically this is './'

`-be` Command to be called before the exec command for example `go mod tidy`

`-e` Command to be called when a modification is detected for example `go run main.go`

`-ae` Command to b be called when a modifcation is detected after the main process closes 

`-l` Log Level to display options can include `"debug", "info","warn","error"`

`-f` path to a TOML config file see [Config File](https://github.com/atterpac/hotato#config-file) for details on the format of config

`-id` Ignore directories provided as a comma-separated list

`-if` Ignore files provided as a comma-separated list

`-ie` Ignore extensions provided as a comma-separated list

`-d` Debounce timer in milliseconds, used to ignore repetitive system

#### Example
```bash
hotato -p ./ -e "go run main.go" -be "go mod tidy" -ae "rm ./main" -l "debug" -id ".git, node_modules" -if ".env" -ie ".db, .sqlite" -d 500
```

### Embedding into your dev project
There can be some uses where you might want to start a watcher internally or for a tool for development Gotato provides a function `NewEngineFromOptions` which takes an `gotato.Config` and allows for the `engine.Start()` function

Using gotato as a library also opens the ability to add a Callback [Callback](https://github.com/atterpac/gotato#reload-callback-function) function that is called on every FS notification

```go
type Config struct {
	RootPath    string `toml:"root_path"`
	PreExec     string `toml:"pre_exec"`
	ExecCommand string `toml:"exec_command"`
	PostExec    string `toml:"post_exec"`
	// Ignore uses a custom unmarshaler see ignore.go
	Ignore       Ignore `toml:"ignore"`
	LogLevel     string `toml:"log_level"`
	Debounce     int    `toml:"debounce"`
	Slog         *slog.Logger
	ExternalSlog bool
}
```

```go 
type Ignore struct {
	Dir       map[string]bool `toml:"dir"`
	File      map[string]bool `toml:"file"`
	Extension map[string]bool `toml:"extension"`
}
```

```go
import ( // other imports
    "github.com/atterpac/hotato/engine"
    )

func main () {
	ignore := hotato.Ignore{
		File:      map[string]bool{{"ignore.go",true},{".env", true}},
		Dir:       map[string]bool{{".git",true},{"node_modules", true}},
		Extension: map[string]bool{{".txt",true},{".db", true}},
	}
	config := hotato.Config{
		RootPath:    "./subExecProcess",
		ExecCommand: "go run main.go",
		LogLevel:    "info", // debug | info | warn | error | mute (discards all logs)
		Ignore:      ignore,
		Debounce:    1000,
		Slog: nil, // Optionally provide a slog interface
                  // if nil a default will be provided
                  // If provided stdout will not be piped through gotato
        Callback: func(*EventCallback) bool // Optionally provide a callback function to be called upon file notification events
                                            // If callback returns true reload will process
                                            // EventCallback is a struct of Name, Path, Time of the event
	}
	engine := gotato.NewEngineFromConfig(config)
	engine.Start()

	// Stop monitoring files and kill child processes
	engine.Stop()
}
```
Reload Callback Function
```go
// Called whenever a change is detected in the filesystem
// By default we ignore file rename/remove and a bunch of other events that would likely cause breaking changes on a reload  see eventmap_[oos].go for default rules
// Callback returns two booleans reload and bypass
// reload: if true will reload the process as long as the eventMap allows it
// bypass: if true will bypass the eventMap and reload the process regardless of any hotato ruleset
type EventCallback struct {
	Name Event  // Type of Notification (Write/Create/Remove...)
	Time time.Time // time.Now() when event was triggered
	Path string    // Full path to the modified file
}

// Example
func Callback(e *gotato.EventCallBack) (bool, bool) {
    // Ignore create file notif
    if e.Name == hotato.Create {
        return false, false
    }
    // Continue as normal for write but add some logs
    if e.Name == hotato.Write{
        fmt.Println("Wow a write was done")
        return true, false
    }
    // Default would normally ignore a remove function, both reload and bypass being true would force a reload 
    if e.Name == hotato.Remove{
        return true, true
    }
```

If you would prefer to load from a [config](https://github.com/Atterpac/hotato#config-file) file rather than building the structs you can use 
```go

hotato.NewEngineFromTOML("path/to/toml")
```

### Config File
Gotato is able to read a config from a .toml file and passed in through the `-f /path/to/config` and example file is provided but should follow the following format

Config can be used with `hotato -f /path/to/config.toml`

```toml
[config]
# Relative to this files location
root_path = "./"
# Runs prior to the exec command starting
pre_exec = "go mod tidy"
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
```

### Alternatives
Hotato not for you? Here are some popular hot reload alternatives

- [Air](https://github.com/cosmtrek/air)
- [Realize](https://github.com/oxequa/realize)
- [Fresh](https://github.com/gravityblast/fresh)
