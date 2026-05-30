package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	refresh "github.com/atterpac/refresh/engine"
)

const version = "0.4.9"

// cliFlags holds the parsed command-line configuration.
type cliFlags struct {
	rootPath    string
	execCommand string
	logLevel    string
	configPath  string
	debounce    int
	version     bool
	gitIgnore   bool
	trapSuspend bool
	ignoreDir   string
	ignoreFile  string
	ignoreExt   string
}

// parseFlags parses args (without the program name) into a cliFlags.
func parseFlags(args []string) (cliFlags, error) {
	var f cliFlags
	fs := flag.NewFlagSet("refresh", flag.ContinueOnError)
	fs.StringVar(&f.rootPath, "p", "./", "Root path to watch")
	fs.StringVar(&f.execCommand, "e", "", "Comma-separated commands to execute on changes")
	fs.StringVar(&f.logLevel, "l", "info", "Log level: debug|info|warn|error|mute")
	fs.StringVar(&f.configPath, "f", "", "Config file to read (.toml or .yaml)")
	fs.StringVar(&f.ignoreDir, "id", "", "Ignore directories (comma-separated)")
	fs.StringVar(&f.ignoreFile, "if", "", "Ignore files (comma-separated)")
	fs.StringVar(&f.ignoreExt, "ie", "", "Watched extensions (comma-separated)")
	fs.IntVar(&f.debounce, "d", 1000, "Debounce time in milliseconds")
	fs.BoolVar(&f.version, "v", false, "Print version")
	fs.BoolVar(&f.gitIgnore, "git", false, "Read .gitignore in the root")
	fs.BoolVar(&f.trapSuspend, "pause", false, "Use Ctrl+Z to toggle pause/resume instead of suspending")
	if err := fs.Parse(args); err != nil {
		return f, err
	}
	return f, nil
}

// splitList splits a comma-separated flag value, trimming whitespace and
// dropping empty entries (so an unset flag yields nil, not [""]).
func splitList(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	var out []string
	for p := range strings.SplitSeq(csv, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// toConfig maps the flags to an engine.Config (used when no config file is given).
func (f cliFlags) toConfig() refresh.Config {
	return refresh.Config{
		RootPath:    f.rootPath,
		ExecList:    splitList(f.execCommand),
		LogLevel:    f.logLevel,
		Debounce:    f.debounce,
		EnablePause: f.trapSuspend,
		Ignore: refresh.Ignore{
			File:         splitList(f.ignoreFile),
			Dir:          splitList(f.ignoreDir),
			WatchedExten: splitList(f.ignoreExt),
			IgnoreGit:    f.gitIgnore,
		},
	}
}

// newEngine builds an engine from a config file when -f is given, otherwise from
// the individual flags.
func newEngine(f cliFlags) (*refresh.Engine, error) {
	if f.configPath != "" {
		switch {
		case strings.HasSuffix(f.configPath, ".toml"):
			return refresh.NewEngineFromTOML(f.configPath)
		case strings.HasSuffix(f.configPath, ".yaml"), strings.HasSuffix(f.configPath, ".yml"):
			return refresh.NewEngineFromYAML(f.configPath)
		default:
			return nil, fmt.Errorf("unsupported config file %q (want .toml or .yaml)", f.configPath)
		}
	}
	return refresh.NewEngineFromConfig(f.toConfig())
}

func main() {
	f, err := parseFlags(os.Args[1:])
	if err != nil {
		os.Exit(2)
	}
	if f.version {
		fmt.Println(PrintBanner(version))
		return
	}

	watch, err := newEngine(f)
	if err != nil {
		slog.Error("failed to configure refresh", "err", err)
		os.Exit(1)
	}
	// Start blocks until a signal triggers shutdown.
	if err := watch.Start(); err != nil {
		slog.Error("refresh exited with error", "err", err)
		os.Exit(1)
	}
}

func PrintBanner(ver string) string {
	return fmt.Sprintf(`
   ___  ___________  __________ __
  / _ \/ __/ __/ _ \/ __/ __/ // /
 / , _/ _// _// , _/ _/_\ \/ _  /
/_/|_/___/_/ /_/|_/___/___/_//_/ CLI v%s
`, ver)
}
