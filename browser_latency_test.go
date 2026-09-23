package dashboard_test

import (
	"os"
	"testing"
	"time"
)

// TestMeasureChromeLaunchLatency is the M91 measurement: serialized
// headless-Chrome launches vs the 45s announce timeout startHeadlessChrome
// enforces. Results land in docs/research/2026-09-10_benchmarks.md.
//
// Opt-in and isolation-required: running it alone keeps the parallel
// browser tests from launching Chrome concurrently and skewing the
// numbers.
//
//	nix develop -c go test -run TestMeasureChromeLaunchLatency -v .
func TestMeasureChromeLaunchLatency(t *testing.T) {
	if os.Getenv("GO_HEALTH_DASHBOARD_MEASURE_CHROME") == "" {
		t.Skip("set GO_HEALTH_DASHBOARD_MEASURE_CHROME=1 to measure Chrome launch latency")
	}

	chromePath := findChrome(t)

	const launches = 5

	var total time.Duration
	var maxSeen time.Duration

	for i := range launches {
		sessionStart := time.Now()

		wsURL, stopChrome := startHeadlessChrome(t, chromePath)
		if wsURL == "" {
			t.Fatal("empty websocket URL")
		}

		launch := time.Duration(lastChromeLaunch.Load())
		total += launch

		if launch > maxSeen {
			maxSeen = launch
		}

		t.Logf("launch %d/%d: start→announce %s (session overhead %s)", i+1, launches, launch, time.Since(sessionStart)-launch)

		stopChrome()
	}

	avg := total / launches
	t.Logf("SUMMARY: launches=%d avg=%s max=%s (announce timeout 45s, utilization %.1f%%)",
		launches, avg, maxSeen, float64(maxSeen)/float64(45*time.Second)*100)
}
