package engine

import (
	"bufio"
	"errors"
	"log/slog"
	"os"
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
	slog.Debug("verifying config")
	if engine.Config.RootPath == "" {
		return errors.New("root path is required")
	}
	if err := engine.verifyExecute(); err != nil {
		return err
	}
	slog.Debug("config verified")
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
	slog.Debug("reading .gitignore")
	scanner := bufio.NewScanner(file)
	linesMap := make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments and blank lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Normalize to a glob so patternMatch can use it.
		if !strings.HasPrefix(line, "*") {
			line = "*" + line
		}
		linesMap[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		slog.Debug("reading .gitignore", "err", err)
	}
	slog.Debug("read .gitignore", "patterns", len(linesMap))
	return linesMap
}

func (e *Engine) generateProcess() {
	for _, ex := range e.Config.ExecStruct {
		e.ProcessManager.AddProcess(ex.Cmd, string(ex.Type), ex.ChangeDir)
	}
}
