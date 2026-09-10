package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// TestWebhookDeliveryMetrics verifies WithMetrics exposes webhook delivery
// counters and a duration histogram, classified ok/error by receiver
// response.
func TestWebhookDeliveryMetrics(t *testing.T) {
	t.Parallel()

	var fail atomic.Bool

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	s := setupDashboard(t,
		dashboard.WithMetrics(true),
		dashboard.WithWebhook(receiver.URL),
	)
	defer s.cleanup()

	// The initial-state announcement fires on Start (setupDashboard), so at
	// least one ok delivery is already recorded. Force one error delivery
	// through the notifier's transition path by flipping the receiver, then
	// toggling the service twice to trigger a new fire.
	fail.Store(true)

	deadline := time.Now().Add(10 * time.Second)
	scrape := func() string {
		w := doRequest(t, s.mux, "/health/metrics")

		return w.Body.String()
	}

	var okCount, errCount string
	for time.Now().Before(deadline) {
		body := scrape()
		okCount = metricValue(body, `dashboard_webhook_deliveries_total{result="ok"}`)
		errCount = metricValue(body, `dashboard_webhook_deliveries_total{result="error"}`)

		if okCount != "" && errCount != "" && errCount != "0" {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if okCount == "" {
		t.Error("dashboard_webhook_deliveries_total{result=\"ok\"} missing from scrape")
	}

	if errCount == "" || errCount == "0" {
		t.Errorf("dashboard_webhook_deliveries_total{result=\"error\"} want >= 1, got %q", errCount)
	}

	body := scrape()
	for _, want := range []string{
		"# TYPE dashboard_webhook_deliveries_total counter",
		"# TYPE dashboard_webhook_delivery_duration_seconds histogram",
		"dashboard_webhook_delivery_duration_seconds_bucket",
		"dashboard_webhook_delivery_duration_seconds_count",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape missing %q", want)
		}
	}
}

// metricValue extracts the sample value for an exact metric series line, or
// "" when absent.
func metricValue(scrape, series string) string {
	for line := range strings.SplitSeq(scrape, "\n") {
		if strings.HasPrefix(line, series+" ") {
			fields := strings.Fields(line)

			if len(fields) == 2 {
				return fields[1]
			}
		}
	}

	return ""
}
