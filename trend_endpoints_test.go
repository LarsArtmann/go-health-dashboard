package dashboard_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// setupTrendDashboard starts a dashboard with a toggleable service and the
// trend endpoints enabled, returning the toggle so tests can force status
// changes.
func setupTrendDashboard(t *testing.T, opts ...dashboard.Option) (*probeSetup, *toggleService) {
	t.Helper()

	svc := &toggleService{}
	svc.healthy.Store(true)

	injector := do.New()
	provideToggleService(injector, "database", svc)
	provideHealthy(injector, "redis")
	invoke[*healthyService](t, injector, "redis")

	probe := health.New(injector,
		health.WithVersion("2.1.0"),
		health.WithCriticalServices("database"),
		health.WithRefreshInterval(50*time.Millisecond),
	)

	dash := dashboard.New(probe, append([]dashboard.Option{
		dashboard.WithPushInterval(30 * time.Millisecond),
		dashboard.WithTrend(200),
	}, opts...)...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	s := &probeSetup{
		probe: probe,
		dash:  dash,
		mux:   mux,
		cleanup: func() {
			dash.Shutdown()
			probe.Shutdown()
		},
	}

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
			if err := json.Unmarshal(
				w.Body.Bytes(),
				&payload,
			); err == nil &&
				len(payload.Samples) >= minSamples {
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
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err == nil {
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
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
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
	if last.From != string(health.StatusPass) || last.To != string(health.StatusFail) {
		t.Errorf("transition: want pass->fail, got %s->%s", last.From, last.To)
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

	var payload struct {
		Samples []struct {
			At     string `json:"at"`
			Status string `json:"status"`
		} `json:"samples"`
		Checks map[string]struct {
			Since      string `json:"since"`
			DurationNs int64  `json:"duration_ns"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode export: %v", err)
	}

	if len(payload.Samples) < 2 {
		t.Errorf("samples: want >= 2, got %d", len(payload.Samples))
	}

	if len(payload.Checks) == 0 {
		t.Errorf("checks object missing or empty: %s", w.Body.String())
	}
}

// TestExportHandler_JSONCarriesCheckMetadata proves the export document —
// the one dashboard-owned JSON shape — carries the current per-check
// go-health v0.2.0 metadata: the state-entry since stamp and the executor
// duration, with unknown facts omitted rather than rendered as zero.
func TestExportHandler_JSONCarriesCheckMetadata(t *testing.T) {
	t.Parallel()

	since := time.Now().UTC().Add(-5 * time.Minute)
	resp := health.Response{
		Status: health.StatusPass,
		Checks: map[string]health.Check{
			"api/database": {
				Status:        health.StatusPass,
				Since:         since,
				DurationNanos: int64(42 * time.Millisecond),
			},
			"api/plain": {Status: health.StatusPass},
		},
	}

	dash := dashboard.New(newStubProber(resp), dashboard.WithTrend(8))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	defer dash.Shutdown()

	w := doRequest(t, mux, "/health/export")
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	var payload struct {
		Checks map[string]struct {
			Since      string `json:"since"`
			DurationNs int64  `json:"duration_ns"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode export: %v\n%s", err, w.Body.String())
	}

	dbMeta, ok := payload.Checks["api/database"]
	if !ok {
		t.Fatalf("api/database missing from checks: %s", w.Body.String())
	}

	if got, want := dbMeta.DurationNs, int64(42*time.Millisecond); got != want {
		t.Errorf("api/database duration_ns = %d, want %d", got, want)
	}

	parsed, err := time.Parse(time.RFC3339, dbMeta.Since)
	if err != nil {
		t.Fatalf("api/database since not RFC3339: %q", dbMeta.Since)
	}

	if d := parsed.Sub(since); d < -time.Second || d > time.Second {
		t.Errorf("api/database since = %s, want within 1s of %s", parsed, since)
	}

	plainMeta, ok := payload.Checks["api/plain"]
	if !ok {
		t.Fatalf("api/plain missing from checks: %s", w.Body.String())
	}

	if plainMeta.DurationNs != 0 || plainMeta.Since != "" {
		t.Errorf(
			"check without reported metadata must omit both fields, got %+v",
			plainMeta,
		)
	}
}

// startByteStableScrapeTarget builds a trend-enabled dashboard from the
// shared mixed-checks response, starts it, and waits until the first sample
// is recorded so successive scrapes see a quiescent sample array. The push
// interval is set far beyond the test so no tick can land between the two
// scrapes and change the samples array.
func startByteStableScrapeTarget(t *testing.T, path, label string) *http.ServeMux {
	t.Helper()

	resp := health.Response{
		Status: health.StatusPass,
		Checks: map[string]health.Check{
			"zulu/queue":     {Status: health.StatusPass},
			"alpha/database": {Status: health.StatusPass, DurationNanos: int64(time.Millisecond)},
			"mid/cache":      {Status: health.StatusPass},
		},
	}

	dash := dashboard.New(
		newStubProber(resp),
		dashboard.WithTrend(8),
		dashboard.WithPushInterval(time.Hour),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	t.Cleanup(dash.Shutdown)

	// The pusher records its first sample immediately on start; wait for
	// it so the sample array is quiescent before the two scrapes.
	deadline := time.Now().Add(5 * time.Second)

	for {
		probe := doRequest(t, mux, path)
		if strings.Contains(probe.Body.String(), `"samples":[{`) {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("%s never recorded a sample: %s", label, probe.Body.String())
		}

		time.Sleep(5 * time.Millisecond)
	}

	return mux
}

// assertScrapeIsByteStable fetches path twice and fails unless both
// payloads are byte-identical. Returns the first scrape for further
// assertions.
func assertScrapeIsByteStable(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()

	first := doRequest(t, mux, path)
	if first.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", first.Code)
	}

	second := doRequest(t, mux, path)

	if first.Body.String() != second.Body.String() {
		t.Fatalf(
			"%s JSON not byte-stable across scrapes:\nfirst:  %s\nsecond: %s",
			path,
			first.Body.String(),
			second.Body.String(),
		)
	}

	return first
}

// TestExportHandler_JSONIsByteStable pins the export wire contract: the
// checks object is a Go map, so successive scrapes must marshal with
// deterministic (sorted) key ordering or consumers diffing exports see
// phantom key reshuffles.
func TestExportHandler_JSONIsByteStable(t *testing.T) {
	t.Parallel()

	mux := startByteStableScrapeTarget(t, "/health/export", "export")

	first := assertScrapeIsByteStable(t, mux, "/health/export")

	want := `"checks":{"alpha/database"`
	if !strings.Contains(first.Body.String(), want) {
		t.Errorf(
			"checks keys not in deterministic (sorted) order, want %s in: %s",
			want,
			first.Body.String(),
		)
	}
}

// TestTrendHandler_JSONIsByteStable future-proofs the trend wire contract:
// the payload is slices-only today (already stable), but one future
// map-bearing field would silently regress byte-stability for diff-based
// scrapers. The endpoint marshals with Deterministic(true); this test makes
// that regression loud.
func TestTrendHandler_JSONIsByteStable(t *testing.T) {
	t.Parallel()

	mux := startByteStableScrapeTarget(t, "/health/trend", "trend")

	assertScrapeIsByteStable(t, mux, "/health/trend")
}

func TestExportHandler_CSV(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 2)

	for _, testCase := range []struct {
		name   string
		accept string
		query  string
	}{
		{name: "query param", query: "?format=csv"},
		{name: "accept header", accept: "text/csv"},
	} {
		w := doRequestWithAccept(t, s.mux, "/health/export"+testCase.query, testCase.accept)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status: want 200, got %d", testCase.name, w.Code)
		}

		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
			t.Errorf("%s: content-type: want text/csv, got %s", testCase.name, ct)
		}

		lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
		if len(lines) < 3 {
			t.Errorf(
				"%s: csv rows: want >= 3 (header + 2 samples), got %d",
				testCase.name,
				len(lines),
			)
		}

		if lines[0] != "timestamp,value,status" {
			t.Errorf("%s: csv header: got %q", testCase.name, lines[0])
		}
	}
}

func TestExportHandler_NDJSON(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 2)

	w := doRequest(t, s.mux, "/health/export?format=ndjson")
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/x-ndjson") {
		t.Fatalf("content-type: want application/x-ndjson, got %s", ct)
	}

	body := strings.TrimSpace(w.Body.String())
	if body == "" {
		t.Fatal("ndjson body is empty")
	}

	if !strings.HasSuffix(w.Body.String(), "\n") {
		t.Error("ndjson payload should end with a newline")
	}

	lines := strings.Split(body, "\n")

	if len(lines) < 2 {
		t.Fatalf("ndjson lines: want >= 2, got %d", len(lines))
	}

	for i, line := range lines {
		var sample struct {
			At     string  `json:"at"`
			Value  float64 `json:"value"`
			Status string  `json:"status"`
		}
		if err := json.Unmarshal([]byte(line), &sample); err != nil {
			t.Fatalf("ndjson line %d is not a self-contained JSON object: %v\n%s", i, err, line)
		}

		if sample.At == "" || sample.Status == "" {
			t.Errorf("ndjson line %d missing at/status: %s", i, line)
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

// TestTrendEndpoints_503WhenPusherNotStarted covers the nil-pusher branch:
// a dashboard that was constructed but never started must answer both trend
// endpoints with 503 and a message that distinguishes "not started" from
// "trend not enabled".
func TestTrendEndpoints_503WhenPusherNotStarted(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "database")
	invoke[*healthyService](t, injector, "database")

	probe := health.New(injector, health.WithRefreshInterval(time.Hour))

	dash := dashboard.New(probe, dashboard.WithTrend(10))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	for _, path := range []string{"/health/trend", "/health/export"} {
		w := doRequest(t, mux, path)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s without Start: want 503, got %d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "pusher is not active") {
			t.Errorf("%s 503 body should name the inactive pusher: %s", path, w.Body.String())
		}
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
