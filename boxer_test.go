package boxer_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/benbjohnson/boxer"
)

// Ensure the ticker can tick for each new step and interval.
func TestTicker_Tick(t *testing.T) {
	// Create a new ticker that steps every 1m and intervals every 15m.
	ticker := boxer.NewTicker()

	// Mock the current time.
	now := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	ticker.NowFunc = func() time.Time { return now }

	// Setup command with a handler.
	var stepN, intervalN int
	cmd := boxer.Command{
		Step:     1 * time.Minute,
		Interval: 15 * time.Minute,
		Handler: func(i, n int) error {
			stepN++
			if i == 0 {
				intervalN++
			}
			return nil
		},
	}
	ticker.Commands = append(ticker.Commands, cmd)

	// Execute the initial tick.
	ticker.Tick()

	// Move forward 10 seconds at a time for 1h.
	start := now
	for i := time.Duration(0); i <= 1*time.Hour; i += 10 * time.Second {
		now = start.Add(i)
		ticker.Tick()
	}

	// Ensure the step and interval count are correct.
	if stepN != 61 {
		t.Fatalf("unexpected step count: %d", stepN)
	} else if intervalN != 5 {
		t.Fatalf("unexpected interval count: %d", intervalN)
	}
}

// Ensure the default command executor can execute and return the output.
func TestDefaultCommandExecutor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	b, err := boxer.DefaultCommandExecutor("echo", []string{"foo", "bar"}, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	} else if string(b) != "foo bar\n" {
		t.Fatalf("unexpected output: %s", b)
	}
}
