//go:build linux || darwin

package engine

import "testing"

// TestEngineStartFailsWhenBootProcessFails verifies that a failure during the
// initial process pass propagates out of Start as an error (rather than the
// engine silently entering its watch loop).
func TestEngineStartFailsWhenBootProcessFails(t *testing.T) {
	cfg := Config{
		RootPath: t.TempDir(),
		LogLevel: "mute",
		Debounce: 100,
		Ignore:   Ignore{WatchedExten: []string{"*.go"}},
		// "false" exits non-zero, so the blocking boot step fails.
		ExecStruct: []Execute{{Cmd: "false", Type: Blocking}},
	}
	eng, err := NewEngineFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineFromConfig: %v", err)
	}
	if err := eng.Start(); err == nil {
		t.Fatal("expected Start to return an error when the boot process fails")
	}
}
