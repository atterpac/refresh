package engine

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/atterpac/refresh/process"
	"gopkg.in/yaml.v2"
)

type Config struct {
	RootPath           string            `toml:"root_path"  yaml:"root_path"`
	BackgroundStruct   process.Execute   `toml:"background" yaml:"background"`
	BackgroundCallback func() bool       `toml:"-"          yaml:"-"`
	Ignore             Ignore            `toml:"ignore"     yaml:"ignore"`
	ExecStruct         []process.Execute `toml:"executes"   yaml:"executes"`
	ExecList           []string          `toml:"exec_list"  yaml:"exec_list"`
	LogLevel           string            `toml:"log_level"  yaml:"log_level"`
	Debounce           int               `toml:"debounce"   yaml:"debounce"`
	Callback           func(*EventCallback) EventHandle
	Slog               *slog.Logger
	ignoreMap          ignoreMap
}

func DefaultEngineConfig() Config {
	return Config{
		RootPath: ".",
		LogLevel: "warn",
		Debounce: 1000,
		Ignore: Ignore{
			Dir:       []string{".git", ".idea", ".node_modules", "vendor"},
			File:      []string{".DS_Store", ".gitignore", ".gitkeep"},
			IgnoreGit: true,
		},
		ExecStruct: make([]process.Execute, 0),
	}
}

func (c *Config) WithRootPath(path string) *Config {
	c.RootPath = path
	return c
}

func (c *Config) WithLogLevel(level string) *Config {
	c.LogLevel = level
	return c
}

func (c *Config) WithDebounce(value int) *Config {
	c.Debounce = value
	return c
}

func (c *Config) WithIgnore(ignore Ignore) *Config {
	c.Ignore = ignore
	return c
}

func (c *Config) WithIgnoreDirs(dir []string) *Config {
	c.Ignore.Dir = dir
	return c
}
func (c *Config) WithIgnoreFiles(files []string) *Config {
	c.Ignore.File = files
	return c
}

func (c *Config) WithIgnoreGit(truthy bool) *Config {
	c.Ignore.IgnoreGit = truthy
	return c
}

func (c *Config) WithWatchedExtensions(extensions []string) *Config {
	c.Ignore.WatchedExten = extensions
	return c
}

func (c *Config) WithExecuteCommand(cmd process.Execute) *Config {
	c.ExecStruct = append(c.ExecStruct, cmd)
	return c
}

// Reads a config.toml file and returns the engine
func (engine *Engine) readConfigFile(path string) (*Engine, error) {
	if _, err := toml.DecodeFile(path, &engine); err != nil {
		slog.Error("reading config file", "path", path, "err", err)
		return nil, err
	}
	return engine, nil
}

func (engine *Engine) readConfigYaml(path string) (*Engine, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		slog.Error("reading config file", "path", path, "err", err)
		return nil, err
	}
	err = yaml.Unmarshal(file, &engine)
	if err != nil {
		slog.Error("parsing yaml config", "path", path, "err", err)
		return nil, err
	}
	return engine, nil
}

func (engine *Engine) StringtoConfigYAML(yamlString string) error {
	err := yaml.Unmarshal([]byte(yamlString), &engine)
	if err != nil {
		slog.Error("parsing yaml config string", "err", err)
		return err
	}
	return nil
}

func (engine *Engine) StringtoConfigTOML(tomlString string) error {
	if _, err := toml.Decode(tomlString, &engine); err != nil {
		slog.Error("parsing toml config string", "err", err)
		return err
	}
	return nil
}

// Verify required data is present in config
func (engine *Engine) verifyConfig() error {
	slog.Debug("Verifying Config")
	if engine.Config.RootPath == "" {
		slog.Error("Required Root Path is not set")
		return errors.New("Required Root Path is not set")
	}
	err := engine.verifyExecute()
	if err != nil {
		return err
	}
	slog.Debug("Config Verified")
	// Change directory executes are called in to match root directory
	cleaned := cleanDirectory(engine.Config.RootPath)
	slog.Info("Changing Working Directory", "dir", cleaned)
	changeWorkingDirectory(cleaned)
	return nil
}

// Verify execute structs
func (engine *Engine) verifyExecute() error {
	var primary bool
	if len(engine.Config.ExecList) == 2 && len(engine.Config.ExecStruct) < 2 {
		return errors.New("Execute list or struct's must be provided in the refresh config")
	}
	if engine.Config.ExecList == nil {
		for _, exe := range engine.Config.ExecStruct {
			if exe.Type == "primary" {
				if primary {
					return errors.New("Only one primary execute can be set")
				}
				primary = true
			}
		}
	}
	return nil
}

func readGitIgnore(path string) map[string]struct{} {
	file, err := os.Open(path + "/.gitignore")
	if err != nil {
		return nil
	}
	defer file.Close()
	slog.Debug("Reading .gitignore")
	scanner := bufio.NewScanner(file)
	var linesMap = make(map[string]struct{})
	for scanner.Scan() {
		// Check if line is a comment
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		// Check if line is empty
		if len(scanner.Text()) == 0 {
			continue
		}

		line := scanner.Text()
		// Check if line does not start with '*'
		if !strings.HasPrefix(line, "*") {
			// Add asterisk to the beginning of line
			line = "*" + line
		}
		// Add to the map
		linesMap[line] = struct{}{}
	}
	slog.Debug(fmt.Sprintf("Read %v lines from .gitignore", linesMap))
	return linesMap
}

func cleanDirectory(path string) string {
	cleaned := strings.TrimPrefix(path, ".")
	cleaned = strings.TrimPrefix(cleaned, "/")
	if runtime.GOOS == "windows" {
		cleaned = strings.TrimPrefix(cleaned, `\`) // Windows >:(
	}
	wd, err := os.Getwd()
	if err != nil {
		slog.Error("Getting Working Directory")
	}
	return wd + "/" + cleaned
}

func changeWorkingDirectory(path string) {
	err := os.Chdir(path)
	if err != nil {
		slog.Error("Setting new directory", "dir", path)
	}
}

func (e *Engine) generateProcess() {
	for _, ex := range e.Config.ExecStruct {
		e.ProcessManager.AddProcess(ex.Cmd, string(ex.Type), ex.ChangeDir)
	}
}
