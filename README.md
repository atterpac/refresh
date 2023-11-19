# :construction: EARLY DEVELOPMENT :construction:
This is a small tool I have built that is largely untested off of my machine. You are welcome to try it and if you notice any issues report them on the github and I will look into them.

## GOTATO Hot Reload
Gotato (golang hot potato) is a tool for hot reloading your codebase based on file system changes using [notify](https://github.com/rjeczalik/notify)

## Install
Installing via go CLI is the easiest method more methods are on the list
```bash
go install github.com/atterpac/gotato/cmd/gotato@latest
```
Alternative if you wish to use as a package and not a cli
```bash
go get github.com/atterpac/gotato
```
## Usage

#### Flags
`-p` Root path that will be watched and commands will be executed in typically this is './'

`-be` Command to be called before the exec command for example `go mod tidy`

`-e` Command to be called when a modification is detected for example `go run main.go`

`-ae` Command to b be called when a modifcation is detected after the main process closes 

`-l` Log Level to display options can include `"debug", "info","warn","error"`

`-f` path to a TOML config file see [Config File](https://github.com/Atterpac/gotato#config-file) for details on the format of config

`-id` Ignore directories provided as a comma-separated list

`-if` Ignore files provided as a comma-separated list

`-ie` Ignore extensions provided as a comma-separated list

`-d` Debounce timer in milliseconds, used to ignore repetitive system

#### Example
```bash
gotato -p ./ -e "go run main.go" -be "go mod tidy" -ae "rm ./main" -l "debug" -id ".git, node_modules" -if ".env" -ie ".db, .sqlite" -d 500
```

### Embedding into your dev project
There can be some uses where you might want to start a watcher internally or for a tool for development Gotato provides a function `NewEngineFromOptions` which takes an `gotato.Config` and allows for the `engine.Start()` function

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
    "github.com/atterpac/gotato/engine"
    )

func main () {
	ignore := gotato.Ignore{
		File:      map[string]bool{{"ignore.go",true},{".env", true}},
		Dir:       map[string]bool{{".git",true},{"node_modules", true}},
		Extension: map[string]bool{{".txt",true},{".db", true}},
	}
	config := gotato.Config{
		RootPath:    "./subExecProcess",
		ExecCommand: "go run main.go",
		LogLevel:    "info",
		Ignore:      ignore,
		Debounce:    1000,
		Slog: nil // Optionally provide a slog interface for gotato to use if nil a default will be provided
	}
	engine := gotato.NewEngineFromConfig(config)
	engine.Start()

	// Stop monitoring files and kill child processes
	engine.Stop()
}
```

If you would prefer to load from a [config](https://github.com/Atterpac/gotato#config-file) file rather than building the structs you can use 
```go

gotato.NewEngineFromTOML("path/to/toml")
```

### Config File
Gotato is able to read a config from a .toml file and passed in through the `-f /path/to/config` and example file is provided but should follow the following format

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
# debug | info | warn | error | fatal
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

