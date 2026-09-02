package dashboard_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// waitForBody polls the dashboard until the body contains match or the
// deadline expires. Used to wait for the pusher to accumulate trend samples
// without fixed sleeps.
func waitForBody(t *testing.T, handler http.Handler, target, match string) string {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	var body string

	for time.Now().Before(deadline) {
		body = doRequest(t, handler, target).Body.String()
		if strings.Contains(body, match) {
			return body
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("body never contained %q within deadline", match)

	return body
}

func TestHTML_TrendHiddenByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	time.Sleep(150 * time.Millisecond) // let the pusher tick a few times anyway

	w := doRequest(t, s.mux, "/health")
	if strings.Contains(w.Body.String(), "Health Trend") {
		t.Error("trend card must not render without WithTrend")
	}
}

func TestHTML_TrendAppearsOnceSamplesExist(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithTrend(60))
	defer s.cleanup()

	// Probe refresh and push interval are 100ms in the test setup, so two
	// samples exist quickly. Poll instead of sleeping to avoid flake.
	body := waitForBody(t, s.mux, "/health", "Health Trend")

	if !strings.Contains(body, "<svg") {
		t.Error("trend card should contain the sparkline SVG")
	}

	if !strings.Contains(body, `aria-label="Recent health status trend`) {
		t.Error("sparkline should carry an aria-label")
	}
}

func TestHTML_TrendSampleCapRespected(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithTrend(3))
	defer s.cleanup()

	body := waitForBody(t, s.mux, "/health", "Health Trend")

	// A 3-sample ring buffer can never exceed 3 points; each point renders
	// as one polyline coordinate pair.
	points := strings.Count(body, "<polyline")
	if points != 1 {
		t.Fatalf("want exactly 1 polyline, got %d", points)
	}
}

func TestHTML_HideStatCards(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithHideStatCards())
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	for _, label := range []string{"Uptime", "Check Latency", ">Version<"} {
		if strings.Contains(body, label) {
			t.Errorf("stat card label %q should be hidden by WithHideStatCards", label)
		}
	}

	if !strings.Contains(body, "All Systems Operational") {
		t.Error("status banner must still render when stat cards are hidden")
	}
}

func TestHTML_StatCardsShownByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	for _, label := range []string{"Uptime", "Check Latency"} {
		if !strings.Contains(body, label) {
			t.Errorf("stat card label %q missing by default", label)
		}
	}
}
