package engine

import (
	"os"
	"path/filepath"
	"testing"
)

const tomlConfig = `
[config]
root_path = "."
log_level = "warn"
debounce = 250

[config.ignore]
watched_extension = ["*.go"]
dir = ["vendor"]

[[config.executes]]
cmd = "go build -o ./app"
type = "blocking"

[[config.executes]]
cmd = "./app"
type = "primary"
`

const yamlConfig = `
config:
  root_path: "."
  log_level: warn
  debounce: 250
  ignore:
    watched_extension: ["*.go"]
    dir: ["vendor"]
  executes:
    - cmd: "go build -o ./app"
      type: blocking
    - cmd: "./app"
      type: primary
`

func assertLoaded(t *testing.T, eng *Engine) {
	t.Helper()
	if eng.Config.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn", eng.Config.LogLevel)
	}
	if eng.Config.Debounce != 250 {
		t.Errorf("Debounce = %d, want 250", eng.Config.Debounce)
	}
	if got := eng.ProcessManager.GetExecutes(); len(got) != 2 ||
		got[0] != "go build -o ./app" || got[1] != "./app" {
		t.Errorf("executes = %v, want [go build..., ./app]", got)
	}
}

func TestNewEngineFromTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh.toml")
	if err := os.WriteFile(path, []byte(tomlConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngineFromTOML(path)
	if err != nil {
		t.Fatalf("NewEngineFromTOML: %v", err)
	}
	assertLoaded(t, eng)
}

func TestNewEngineFromYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh.yaml")
	if err := os.WriteFile(path, []byte(yamlConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngineFromYAML(path)
	if err != nil {
		t.Fatalf("NewEngineFromYAML: %v", err)
	}
	assertLoaded(t, eng)
}

// TestStringToConfigTOML guards the fix for the bug where StringtoConfigTOML
// decoded TOML with the YAML unmarshaler (which would leave the config empty).
func TestStringToConfigTOML(t *testing.T) {
	e := &Engine{}
	if err := e.StringtoConfigTOML("[config]\nroot_path = \"from-toml\"\ndebounce = 42\n"); err != nil {
		t.Fatalf("StringtoConfigTOML: %v", err)
	}
	if e.Config.RootPath != "from-toml" {
		t.Errorf("RootPath = %q, want from-toml (TOML was not parsed as TOML)", e.Config.RootPath)
	}
	if e.Config.Debounce != 42 {
		t.Errorf("Debounce = %d, want 42", e.Config.Debounce)
	}
}

func TestStringToConfigYAML(t *testing.T) {
	e := &Engine{}
	if err := e.StringtoConfigYAML("config:\n  root_path: from-yaml\n  debounce: 7\n"); err != nil {
		t.Fatalf("StringtoConfigYAML: %v", err)
	}
	if e.Config.RootPath != "from-yaml" {
		t.Errorf("RootPath = %q, want from-yaml", e.Config.RootPath)
	}
	if e.Config.Debounce != 7 {
		t.Errorf("Debounce = %d, want 7", e.Config.Debounce)
	}
}
