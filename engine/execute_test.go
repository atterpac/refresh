package	engine

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	engine := &Engine{} // Make sure to initialize it appropriately

	// Test case 1: KILL_STALE command
	killStaleCommand := Execute{
		Cmd:        "KILL_STALE",
		IsBlocking: true,
		IsPrimary:  false,
	}
	err := killStaleCommand.run(engine)
	if err != nil {
		t.Errorf("Error executing KILL_STALE command: %v", err)
	}

	// Test case 2: Custom command with blocking
	counterCommand := Execute{
		Cmd:        "./testdata/counter.sh 3",
		IsBlocking: true,
		IsPrimary:  false,
	}
	startTime := time.Now()
	err = counterCommand.run(engine)
	elapsedTime := time.Since(startTime)
	if err != nil {
		t.Errorf("Error executing counter command: %v", err)
	}
	expectedDuration := 3 * time.Second // Change this according to your counter arg
	if elapsedTime < expectedDuration {
		t.Errorf("Blocking command did not take expected time. Got: %v, Expected at least: %v", elapsedTime, expectedDuration)
	}

	// Test case 3: Custom command without blocking
	counterCommandNonBlocking := Execute{
		Cmd:        "./testdata/counter.sh 5",
		IsBlocking: false,
		IsPrimary:  false,
	}
	startTimeNonBlocking := time.Now()
	err = counterCommandNonBlocking.run(engine)
	elapsedTimeNonBlocking := time.Since(startTimeNonBlocking)
	if err != nil {
		t.Errorf("Error executing non-blocking counter command: %v", err)
	}
	if elapsedTimeNonBlocking > expectedDuration {
		t.Errorf("Non-blocking command took longer than expected. Got: %v", elapsedTimeNonBlocking)
	}

	// Test case 4: Primary
	_, err = engine.startPrimary("echo 'Primary'")
	if err != nil {
		t.Errorf("Error Executing Primary : %v", err)
	}
}
