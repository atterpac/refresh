package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewEngineFromTOMLErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := NewEngineFromTOML(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
			t.Fatal("expected error for missing TOML file")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.toml")
		if err := os.WriteFile(path, []byte("[unterminated"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := NewEngineFromTOML(path); err == nil {
			t.Fatal("expected error for malformed TOML")
		}
	})
}

func TestNewEngineFromYAMLErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := NewEngineFromYAML(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
			t.Fatal("expected error for missing YAML file")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.yaml")
		if err := os.WriteFile(path, []byte("a: b: c"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := NewEngineFromYAML(path); err == nil {
			t.Fatal("expected error for malformed YAML")
		}
	})
}

func TestNewEngineFromConfigRequiresRootPath(t *testing.T) {
	if _, err := NewEngineFromConfig(Config{ExecStruct: []Execute{{Cmd: "x", Type: Primary}}}); err == nil {
		t.Fatal("expected error when RootPath is empty")
	}
}

func TestStringToConfigInvalid(t *testing.T) {
	e := &Engine{}
	if err := e.StringtoConfigTOML("= 1"); err == nil {
		t.Error("expected error for invalid TOML string")
	}
	if err := e.StringtoConfigYAML("a: b: c"); err == nil {
		t.Error("expected error for invalid YAML string")
	}
}
