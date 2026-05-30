package engine

import (
	"bufio"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/atterpac/refresh/process"
	"gopkg.in/yaml.v2"
)

type Config struct {
	RootPath         string            `toml:"root_path"  yaml:"root_path"`
	BackgroundStruct process.Execute   `toml:"background" yaml:"background"`
	Ignore           Ignore            `toml:"ignore"     yaml:"ignore"`
	ExecStruct       []process.Execute `toml:"executes"   yaml:"executes"`
	ExecList         []string          `toml:"exec_list"  yaml:"exec_list"`
	LogLevel         string            `toml:"log_level"  yaml:"log_level"`
	Debounce         int               `toml:"debounce"   yaml:"debounce"`
	Callback         func(*EventCallback) EventHandle
	Slog             *slog.Logger
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
	engine.normalizeExecutes()
	if err := engine.verifyExecute(); err != nil {
		return err
	}
	slog.Debug("config verified")
	return nil
}

// normalizeExecutes converts the simpler ExecList (string) form into the
// canonical ExecStruct form when no struct executes were supplied, so the rest
// of the engine only ever deals with one representation.
func (engine *Engine) normalizeExecutes() {
	if len(engine.Config.ExecStruct) == 0 && len(engine.Config.ExecList) > 0 {
		engine.Config.ExecStruct = execListToSpecs(engine.Config.ExecList)
	}
}

// execListToSpecs maps an ExecList into Execute structs. Commands are blocking
// by default; REFRESH_EXEC marks the following command as the primary process,
// and KILL_EXEC is accepted but ignored (the supervisor handles stale kills).
// If no REFRESH_EXEC marker is present, the last command becomes the primary so
// a bare list still runs something long-lived.
func execListToSpecs(list []string) []process.Execute {
	specs := make([]process.Execute, 0, len(list))
	primaryNext := false
	for _, raw := range list {
		cmd := strings.TrimSpace(raw)
		switch cmd {
		case "":
			continue
		case process.KILL_EXEC:
			continue
		case process.REFRESH_EXEC:
			primaryNext = true
			continue
		}
		execType := process.Blocking
		if primaryNext {
			execType = process.Primary
			primaryNext = false
		}
		specs = append(specs, process.Execute{Cmd: cmd, Type: execType})
	}

	if len(specs) > 0 && !hasPrimary(specs) {
		specs[len(specs)-1].Type = process.Primary
	}
	return specs
}

func hasPrimary(specs []process.Execute) bool {
	for _, s := range specs {
		if s.Type == process.Primary {
			return true
		}
	}
	return false
}

// verifyExecute ensures at least one execute is configured and that no more than
// one primary process is declared.
func (engine *Engine) verifyExecute() error {
	if len(engine.Config.ExecStruct) == 0 {
		return errors.New("at least one execute must be provided via ExecStruct or ExecList")
	}
	primary := 0
	for _, exe := range engine.Config.ExecStruct {
		if exe.Type == process.Primary {
			primary++
		}
	}
	if primary > 1 {
		return errors.New("only one primary execute can be set")
	}
	return nil
}

// readGitIgnore reads the root .gitignore and returns its entries as globs that
// patternMatch can apply. Returns nil if there is no .gitignore.
func readGitIgnore(path string) []string {
	file, err := os.Open(filepath.Join(path, ".gitignore"))
	if err != nil {
		return nil
	}
	defer file.Close()
	slog.Debug("reading .gitignore")
	scanner := bufio.NewScanner(file)
	var patterns []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and blank lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Normalize to a glob so patternMatch can use it.
		if !strings.HasPrefix(line, "*") {
			line = "*" + line
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		slog.Debug("reading .gitignore", "err", err)
	}
	slog.Debug("read .gitignore", "patterns", len(patterns))
	return patterns
}

func (e *Engine) generateProcess() {
	// A configured background command is started once at startup, survives
	// reloads, and is killed on shutdown — regardless of any Type set on it.
	if bg := e.Config.BackgroundStruct; bg.Cmd != "" {
		_ = e.ProcessManager.AddProcess(bg.Cmd, string(process.Background), bg.ChangeDir)
	}
	for _, ex := range e.Config.ExecStruct {
		_ = e.ProcessManager.AddProcess(ex.Cmd, string(ex.Type), ex.ChangeDir)
	}
}
