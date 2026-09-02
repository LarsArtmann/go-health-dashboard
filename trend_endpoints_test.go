package dashboard_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
)

// setupTrendDashboard starts a dashboard with a toggleable service and the
// trend endpoints enabled, returning the toggle so tests can force status
// changes.
func setupTrendDashboard(t *testing.T, opts ...dashboard.Option) (*probeSetup, *toggleService) {
	t.Helper()

	svc := &toggleService{}
	svc.healthy.Store(true)

	s := setupDashboard(t, append([]dashboard.Option{
		dashboard.WithPushInterval(30 * time.Millisecond),
		dashboard.WithTrend(200),
	}, opts...)...)

	return s, svc
}

func waitForTrendSamples(t *testing.T, s *probeSetup, minSamples int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for {
		w := doRequest(t, s.mux, "/health/trend")
		if w.Code == http.StatusOK {
			var payload struct {
				Samples []struct {
					Status string `json:"status"`
				} `json:"samples"`
			}
			if err := json.Unmarshal([]byte(w.Body.String()), &payload); err == nil && len(payload.Samples) >= minSamples {
				return
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("trend never produced %d samples", minSamples)
		}

		time.Sleep(25 * time.Millisecond)
	}
}

func TestTrendHandler_ServesSamplesAndTransitions(t *testing.T) {
	t.Parallel()

	s, svc := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 3)

	svc.healthy.Store(false)

	var transitions int

	deadline := time.Now().Add(5 * time.Second)

	for transitions == 0 {
		w := doRequest(t, s.mux, "/health/trend")

		var payload struct {
			Transitions []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"transitions"`
		}
		if err := json.Unmarshal([]byte(w.Body.String()), &payload); err == nil {
			transitions = len(payload.Transitions)
		}

		if time.Now().After(deadline) {
			t.Fatal("trend never recorded a status transition")
		}

		time.Sleep(25 * time.Millisecond)
	}

	w := doRequest(t, s.mux, "/health/trend")
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type: want application/json, got %s", ct)
	}

	var payload struct {
		Samples []struct {
			At     string  `json:"at"`
			Value  float64 `json:"value"`
			Status string  `json:"status"`
		} `json:"samples"`
		Transitions []struct {
			At   string `json:"at"`
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"transitions"`
	}
	if err := json.Unmarshal([]byte(w.Body.String()), &payload); err != nil {
		t.Fatalf("decode trend payload: %v", err)
	}

	if len(payload.Samples) < 3 {
		t.Errorf("samples: want >= 3, got %d", len(payload.Samples))
	}

	for _, s := range payload.Samples {
		if _, err := time.Parse(time.RFC3339, s.At); err != nil {
			t.Errorf("sample timestamp %q not RFC3339: %v", s.At, err)
		}
	}

	last := payload.Transitions[len(payload.Transitions)-1]
	if last.From != string(health.StatusPass) || last.To != string(health.StatusWarn) {
		t.Errorf("transition: want pass->warn, got %s->%s", last.From, last.To)
	}
}

func TestExportHandler_JSON(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 2)

	w := doRequest(t, s.mux, "/health/export")
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	var samples []struct {
		At     string `json:"at"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(w.Body.String()), &samples); err != nil {
		t.Fatalf("decode export: %v", err)
	}

	if len(samples) < 2 {
		t.Errorf("samples: want >= 2, got %d", len(samples))
	}
}

func TestExportHandler_CSV(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 2)

	for _, tc := range []struct {
		name   string
		accept string
		query  string
	}{
		{name: "query param", query: "?format=csv"},
		{name: "accept header", accept: "text/csv"},
	} {
		w := doRequest(t, s.mux, "/health/export"+tc.query)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status: want 200, got %d", tc.name, w.Code)
		}

		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
			t.Errorf("%s: content-type: want text/csv, got %s", tc.name, ct)
		}

		lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
		if len(lines) < 3 {
			t.Errorf("%s: csv rows: want >= 3 (header + 2 samples), got %d", tc.name, len(lines))
		}

		if lines[0] != "timestamp,value,status" {
			t.Errorf("%s: csv header: got %q", tc.name, lines[0])
		}
	}
}

func TestTrendEndpoints_DisabledWithoutTrend(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health/trend"); w.Code != http.StatusNotFound {
		t.Errorf("trend without WithTrend: want 404, got %d", w.Code)
	}

	if w := doRequest(t, s.mux, "/health/export"); w.Code != http.StatusNotFound {
		t.Errorf("export without WithTrend: want 404, got %d", w.Code)
	}
}

func TestMetrics_LatencyHistogram(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t, dashboard.WithMetrics(true))
	defer s.cleanup()

	waitForTrendSamples(t, s, 3)

	w := doRequest(t, s.mux, "/health/metrics")
	body := w.Body.String()

	for _, want := range []string{
		"# TYPE dashboard_health_check_duration_seconds histogram",
		`dashboard_health_check_duration_seconds_bucket{le="0.1"}`,
		`dashboard_health_check_duration_seconds_bucket{le="+Inf"}`,
		"dashboard_health_check_duration_seconds_sum ",
		"dashboard_health_check_duration_seconds_count 3",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("histogram output missing %q", want)
		}
	}
}

func TestHTML_RefreshTimestampAndTimeline(t *testing.T) {
	t.Parallel()

	s, svc := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 3)

	svc.healthy.Store(false)

	time.Sleep(150 * time.Millisecond)

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	if !strings.Contains(body, "Updated ") {
		t.Error("dashboard HTML missing refresh timestamp")
	}

	if !strings.Contains(body, "Status Changes") {
		t.Error("dashboard HTML missing status-change timeline")
	}
}
