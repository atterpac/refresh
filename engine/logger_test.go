package engine

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// newCapture returns a dynamicLogger whose output is captured into buf, so the
// runtime level/enable controls can be asserted on real emitted records.
func newCapture(level string, buf *bytes.Buffer) *dynamicLogger {
	custom := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return newDynamicLogger(level, custom)
}

func TestLoggerRespectsConfiguredLevel(t *testing.T) {
	var buf bytes.Buffer
	d := newCapture("warn", &buf)

	d.logger.Info("hidden")
	d.logger.Warn("shown")

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Errorf("info record emitted at warn level: %q", out)
	}
	if !strings.Contains(out, "shown") {
		t.Errorf("warn record missing at warn level: %q", out)
	}
}

func TestSetLevelChangesVerbosityAtRuntime(t *testing.T) {
	var buf bytes.Buffer
	d := newCapture("error", &buf)

	d.logger.Debug("before")
	if strings.Contains(buf.String(), "before") {
		t.Fatalf("debug emitted while level=error: %q", buf.String())
	}

	d.SetLevel("debug")
	d.logger.Debug("after")
	if !strings.Contains(buf.String(), "after") {
		t.Errorf("debug not emitted after SetLevel(debug): %q", buf.String())
	}
}

func TestDisableEnableLogs(t *testing.T) {
	var buf bytes.Buffer
	d := newCapture("debug", &buf)

	d.Disable()
	d.logger.Error("while-disabled")
	if buf.Len() != 0 {
		t.Errorf("output emitted while disabled: %q", buf.String())
	}

	d.Enable()
	d.logger.Error("while-enabled")
	if !strings.Contains(buf.String(), "while-enabled") {
		t.Errorf("no output after Enable: %q", buf.String())
	}
}

func TestMuteLevelDisablesOutput(t *testing.T) {
	var buf bytes.Buffer
	d := newCapture("mute", &buf)

	d.logger.Error("muted")
	if buf.Len() != 0 {
		t.Errorf("mute did not suppress output: %q", buf.String())
	}

	// "mute" must remain recoverable: switching to a real level re-enables.
	d.SetLevel("info")
	d.logger.Info("recovered")
	if !strings.Contains(buf.String(), "recovered") {
		t.Errorf("output not restored after leaving mute: %q", buf.String())
	}
}
