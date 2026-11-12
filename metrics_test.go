package dashboard_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/samber/do/v2"
)

func TestMetrics_DisabledByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/metrics")
	if w.Code != http.StatusNotFound {
		t.Errorf("disabled metrics route: want 404, got %d", w.Code)
	}
}

func TestMetrics_EnabledServesExposition(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMetrics(true))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/metrics")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type: want text/plain, got %s", ct)
	}

	// The watchdog-gauge series only exists once the pusher has completed
	// its first tick; Start returns before the goroutine's initial
	// broadcast necessarily ran, so poll instead of racing it.
	body := waitForMetricsLine(t, s, "# TYPE dashboard_pusher_last_tick_seconds gauge")

	wantLines := []string{
		"# TYPE dashboard_health_up gauge",
		"dashboard_health_up 1",
		"# TYPE dashboard_health_status gauge",
		"dashboard_health_status 2",
		`dashboard_health_check{check="database",status="pass"} 1`,
		`dashboard_health_check{check="redis",status="pass"} 1`,
		"# TYPE dashboard_health_latency_ms gauge",
		"# TYPE dashboard_health_shutting_down gauge",
		"dashboard_health_shutting_down 0",
		"# TYPE dashboard_sse_connections gauge",
		"# TYPE dashboard_sse_connections_opened_total counter",
		"dashboard_sse_connections_opened_total 0",
		"# TYPE dashboard_sse_connections_closed_total counter",
		"dashboard_sse_connections_closed_total 0",
		"# TYPE dashboard_pusher_active gauge",
		"dashboard_pusher_active 1",
		"# TYPE dashboard_pusher_last_tick_seconds gauge",
	}

	for _, line := range wantLines {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("exposition missing line %q\nbody:\n%s", line, body)
		}
	}
}

func TestMetrics_ReflectsFailures(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t, dashboard.WithMetrics(true))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/metrics")
	body := w.Body.String()

	wantLines := []string{
		"dashboard_health_up 0",
		"dashboard_health_status 1",
		// cache and queue are non-critical, so go-health marks them warn.
		`dashboard_health_check{check="cache",status="warn"} 0`,
		`dashboard_health_check{check="database",status="pass"} 1`,
		`dashboard_health_check{check="queue",status="warn"} 0`,
	}

	for _, line := range wantLines {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("exposition missing line %q\nbody:\n%s", line, body)
		}
	}
}

func TestMetrics_ChecksSortedForDeterministicOutput(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMetrics(true))
	defer s.cleanup()

	first := stripVaryingMetrics(doRequest(t, s.mux, "/health/metrics").Body.String())

	for range 5 {
		second := stripVaryingMetrics(doRequest(t, s.mux, "/health/metrics").Body.String())
		if second != first {
			t.Fatalf(
				"exposition output changed between scrapes:\nfirst:\n%s\nsecond:\n%s",
				first,
				second,
			)
		}
	}

	dbIdx := strings.Index(first, `check="database"`)
	redisIdx := strings.Index(first, `check="redis"`)
	if dbIdx == -1 || redisIdx == -1 || dbIdx > redisIdx {
		t.Errorf("check series not sorted by name:\n%s", first)
	}
}

func TestMetrics_EscapesLabelValues(t *testing.T) {
	t.Parallel()

	weirdName := `back\slash"quote` + "\n" + "newline"

	injector := do.New()
	do.ProvideNamed(injector, weirdName, func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
	invoke[*healthyService](t, injector, weirdName)

	probe := health.New(injector, health.WithRefreshInterval(100*time.Millisecond))

	s := startDashboardFixture(t, probe, dashboard.WithMetrics(true))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/metrics")
	body := w.Body.String()

	want := "dashboard_health_check{check=\"back\\\\slash\\\"quote\\nnewline\",status=\"pass\"} 1"
	if !strings.Contains(body, want+"\n") {
		t.Errorf("escaped label line missing.\nwant: %s\nbody:\n%s", want, body)
	}
}

func TestMetrics_EnabledWithEmptyRouteRegistersNothing(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t,
		dashboard.WithMetrics(true),
		dashboard.WithRoutes(dashboard.Routes{
			Dashboard: "/status",
			SSE:       "/status/sse",
			Favicon:   "/favicon.svg",
			Liveness:  "/healthz",
			Readiness: "/ready",
			Startup:   "/startupz",
			Metrics:   "",
		}),
	)
	defer s.cleanup()

	for _, route := range []string{"/health/metrics", "/metrics"} {
		w := doRequest(t, s.mux, route)
		if w.Code != http.StatusNotFound {
			t.Errorf("empty Metrics route should not register %s: want 404, got %d", route, w.Code)
		}
	}
}

func TestMetrics_BasePathPrefixesMetricsRoute(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t,
		dashboard.WithMetrics(true),
		dashboard.WithBasePath("/admin"),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/admin/health/metrics")
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if body := w.Body.String(); !strings.Contains(body, "dashboard_health_up") {
		t.Error("prefixed metrics route should serve exposition")
	}
}

// TestMetrics_PrometheusParserConformance scrapes the exposition and parses
// it with the official prometheus/common text-format parser — the same
// parser Prometheus itself uses. Invariants: the payload parses cleanly
// under legacy (strict) name validation, all seven metric families are
// present, and label values containing quotes, backslashes, and newlines
// survive the escape round-trip losslessly.
func TestMetrics_PrometheusParserConformance(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "database")
	provideUnhealthy(injector, `cache "weird"\name`, "err \\ \"quoted\"\nsecond line")
	provideUnhealthy(injector, "queue", "timeout")
	invoke[*healthyService](t, injector, "database")
	invoke[*unhealthyService](t, injector, `cache "weird"\name`)
	invoke[*unhealthyService](t, injector, "queue")

	probe := health.New(injector,
		health.WithVersion("2.1.0"),
		health.WithRefreshInterval(100*time.Millisecond),
	)
	dash := dashboard.New(probe, dashboard.WithMetrics(true))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	defer probe.Shutdown()

	w := doRequest(t, mux, "/health/metrics")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	body := w.Body.String()

	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(body))
	if err != nil {
		t.Fatalf("official Prometheus parser rejected the exposition: %v\npayload:\n%s", err, body)
	}

	requireMetricFamilies(
		t,
		families,
		body,
		"dashboard_health_up",
		"dashboard_health_status",
		"dashboard_health_check",
		"dashboard_health_latency_ms",
		"dashboard_health_shutting_down",
		"dashboard_sse_connections",
		"dashboard_pusher_active",
	)

	checkFamily, ok := families["dashboard_health_check"]
	if !ok {
		t.Fatalf("dashboard_health_check family missing in payload:\n%s", body)
	}

	if len(checkFamily.GetMetric()) != 3 {
		t.Errorf("dashboard_health_check: want 3 samples, got %d", len(checkFamily.GetMetric()))
	}

	byCheck := checkSampleIndex(t, checkFamily)

	// The check name carries a quote and a backslash; the error message
	// carries a newline. Both must survive escaping and parsing unchanged.
	const weirdName = `cache "weird"\name`

	// Escaped label values must round-trip through the official parser.
	requireSample(t, byCheck, "database\tpass", 1)
	requireSample(t, byCheck, weirdName+"\twarn", 0)
	requireSample(t, byCheck, "queue\twarn", 0)

	if len(checkFamily.GetHelp()) == 0 {
		t.Error("dashboard_health_check missing HELP text")
	}
}

// requireMetricFamilies fails the test when any expected family is absent.
func requireMetricFamilies(
	t *testing.T,
	families map[string]*dto.MetricFamily,
	body string,
	names ...string,
) {
	t.Helper()

	for _, name := range names {
		if _, ok := families[name]; !ok {
			t.Errorf("missing metric family %q in payload:\n%s", name, body)
		}
	}
}

// requireSample fails the test when an indexed sample is absent or has a
// different value.
func requireSample(t *testing.T, index map[string]float64, key string, want float64) {
	t.Helper()

	got, ok := index[key]
	if !ok || got != want {
		t.Errorf("sample %q: want %v, got %v", key, want, index)
	}
}

// checkSampleIndex maps "<check>\t<status>" to the sample value, failing
// the test when a sample lacks either label.
func checkSampleIndex(t *testing.T, family *dto.MetricFamily) map[string]float64 {
	t.Helper()

	index := make(map[string]float64)

	for _, sample := range family.GetMetric() {
		var name, status string

		for _, lp := range sample.GetLabel() {
			switch lp.GetName() {
			case "check":
				name = lp.GetValue()
			case "status":
				status = lp.GetValue()
			}
		}

		if name == "" || status == "" {
			t.Errorf("sample missing check/status labels: %v", sample.GetLabel())

			continue
		}

		index[name+"\t"+status] = sample.GetGauge().GetValue()
	}

	return index
}

// TestMetrics_PromtoolCheckWhenAvailable runs the official promtool linter
// against the exposition when a promtool binary is on PATH. The parser
// conformance test above always runs; this adds the scrape lint rules
// (help-text presence, naming conventions) where the tool is installed.
func TestMetrics_PromtoolCheckWhenAvailable(t *testing.T) {
	t.Parallel()

	promtool, err := exec.LookPath("promtool")
	if err != nil {
		t.Skip(
			"promtool not on PATH; install with: go install github.com/prometheus/prometheus/cmd/promtool@latest",
		)
	}

	s := setupDashboardWithFailures(t, dashboard.WithMetrics(true))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/metrics")

	cmd := exec.Command(promtool, "check", "metrics")
	cmd.Stdin = strings.NewReader(w.Body.String())

	if out, checkErr := cmd.CombinedOutput(); checkErr != nil {
		t.Errorf("promtool check metrics rejected the exposition: %v\n%s", checkErr, out)
	}
}

func TestMetrics_PusherActiveFalseBeforeStart(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "database")
	invoke[*healthyService](t, injector, "database")

	probe := health.New(injector, health.WithRefreshInterval(100*time.Millisecond))
	dash := dashboard.New(probe, dashboard.WithMetrics(true))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	w := doRequest(t, mux, "/health/metrics")

	if !strings.Contains(w.Body.String(), "dashboard_pusher_active 0\n") {
		t.Error("pusher_active should be 0 before Start")
	}
}

// stripVaryingMetrics removes metric families whose values legitimately
// change between scrapes (latency histogram observations) so the
// determinism assertion focuses on the structure it guards: sorted,
// stable check output.
func stripVaryingMetrics(body string) string {
	kept := make([]string, 0, 32)

	for line := range strings.SplitSeq(body, "\n") {
		if strings.Contains(line, "dashboard_health_check_duration_seconds") ||
			strings.Contains(line, "dashboard_pusher_last_tick_seconds") {
			continue
		}

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")
}

func TestMetrics_CheckDurationGauge(t *testing.T) {
	t.Parallel()

	resp := health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{
			"database": {Status: health.StatusPass, DurationNanos: int64(42 * time.Millisecond)},
			"queue": {
				Status:        health.StatusWarn,
				Error:         "slow",
				DurationNanos: int64(823 * time.Microsecond),
			},
			"plain": {Status: health.StatusPass},
		},
	}

	dash := dashboard.New(newStubProber(resp), dashboard.WithMetrics(true))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	body := doRequest(t, mux, "/health/metrics").Body.String()

	wantLines := []string{
		"# HELP dashboard_health_check_last_duration_seconds Duration of the most recent execution of each check, in seconds.",
		"# TYPE dashboard_health_check_last_duration_seconds gauge",
		`dashboard_health_check_last_duration_seconds{check="database"} 0.042`,
		`dashboard_health_check_last_duration_seconds{check="queue"} 0.000823`,
	}

	for _, line := range wantLines {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("exposition missing line %q\nbody:\n%s", line, body)
		}
	}

	if strings.Contains(body, `last_duration_seconds{check="plain"`) {
		t.Errorf("checks without reported timing must not emit a duration series:\n%s", body)
	}
}

func TestMetrics_CheckDurationGaugePublicMode(t *testing.T) {
	t.Parallel()

	resp := health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{
			"database": {Status: health.StatusPass, DurationNanos: int64(42 * time.Millisecond)},
			"queue":    {Status: health.StatusWarn, Error: "slow"},
		},
	}

	dash := dashboard.New(
		newStubProber(resp),
		dashboard.WithMetrics(true),
		dashboard.WithPublicMode(),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	body := doRequest(t, mux, "/health/metrics").Body.String()

	wantLines := []string{
		`dashboard_health_check{check="check-1",status="pass"} 1`,
		`dashboard_health_check_last_duration_seconds{check="check-1"} 0.042`,
	}

	for _, line := range wantLines {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("exposition missing line %q\nbody:\n%s", line, body)
		}
	}

	if strings.Contains(body, `last_duration_seconds{check="database"`) {
		t.Errorf("duration gauge leaks the check name in public mode:\n%s", body)
	}

	// queue sorts second (check-2) and reports no timing: no series at all.
	if strings.Contains(body, `last_duration_seconds{check="check-2"`) {
		t.Errorf("unreported timing must not emit a series in public mode:\n%s", body)
	}
}

// TestMetrics_SSEConnectionCounters pins the M13 lifecycle pair: opened
// counts accepted upgrades, closed counts their ends, and the connections
// gauge reconciles (opened − closed) once quiescent. The watchdog gauge
// rides the same scrape: present while a pusher has ticked, absent when it
// has not (an absent series must never read as 0s-fresh).
func TestMetrics_SSEConnectionCounters(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMetrics(true))
	defer s.cleanup()

	srv := httptest.NewServer(s.mux)
	defer srv.Close()

	metric := func(body, name string) string {
		for line := range strings.SplitSeq(body, "\n") {
			if value, ok := strings.CutPrefix(line, name+" "); ok {
				return value
			}
		}

		t.Fatalf("metrics exposition missing %s\nbody:\n%s", name, body)

		return ""
	}

	metrics := func() map[string]string {
		rec := httptest.NewRecorder()
		s.dash.MetricsHandler().
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/metrics", nil))

		body := rec.Body.String()

		return map[string]string{
			"opened":    metric(body, "dashboard_sse_connections_opened_total"),
			"closed":    metric(body, "dashboard_sse_connections_closed_total"),
			"active":    metric(body, "dashboard_sse_connections"),
			"last_tick": metric(body, "dashboard_pusher_last_tick_seconds"),
		}
	}

	intOf := func(t *testing.T, m map[string]string, key string) int {
		t.Helper()

		n, err := strconv.Atoi(m[key])
		if err != nil {
			t.Fatalf("%s = %q, want an integer", key, m[key])
		}

		return n
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE request: %v", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 1024)); err != nil {
		t.Fatalf("read initial patch: %v", err)
	}

	waitUntil(t, 5*time.Second, func() bool {
		return s.dash.SubscriberCount() == 1
	})

	m := metrics()

	if got := intOf(t, m, "opened"); got != 1 {
		t.Errorf("opened = %d, want 1 (one accepted upgrade)", got)
	}

	if got := intOf(t, m, "closed"); got != 0 {
		t.Errorf("closed = %d while the subscriber is live, want 0", got)
	}

	if got := intOf(t, m, "active"); got != 1 {
		t.Errorf("active gauge = %d, want 1", got)
	}

	if _, err := time.ParseDuration(m["last_tick"] + "s"); err != nil {
		t.Errorf("last_tick = %q, want a seconds value (pusher has ticked)", m["last_tick"])
	}

	// Drop the connection and wait for the server side to notice.
	cancel()

	waitUntil(t, 5*time.Second, func() bool {
		return s.dash.SubscriberCount() == 0
	})

	m = metrics()

	if got := intOf(t, m, "opened"); got != 1 {
		t.Errorf("opened after disconnect = %d, want 1", got)
	}

	if got := intOf(t, m, "closed"); got != 1 {
		t.Errorf("closed after disconnect = %d, want 1", got)
	}

	if got := intOf(t, m, "active"); got != 0 {
		t.Errorf("active gauge after disconnect = %d, want 0 (opened − closed)", got)
	}
}

// waitUntil polls cond every 25ms until it holds or the deadline expires.
func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held within the deadline")
		}

		time.Sleep(25 * time.Millisecond)
	}
}

// waitForMetricsLine polls the exposition until it contains the wanted
// line, re-scraping every 25ms; returns the last body either way.
func waitForMetricsLine(t *testing.T, s *probeSetup, want string) string {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	var body string

	for {
		body = doRequest(t, s.mux, "/health/metrics").Body.String()
		if strings.Contains(body, want) {
			return body
		}

		if time.Now().After(deadline) {
			return body
		}

		time.Sleep(25 * time.Millisecond)
	}
}

// TestMetrics_RateLimitFamily pins the M14 observability pair: every 429
// increments dashboard_ratelimit_429_total, and the Retry-After seconds
// served with it land in the histogram family. Both render only when
// WithRateLimit is configured.
func TestMetrics_RateLimitFamily(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMetrics(true), dashboard.WithRateLimit(2, time.Second))
	defer s.cleanup()

	// Burn the two-token budget; the third hit is rejected. The metrics
	// scrape itself is rate-limited like any dashboard-owned route, so
	// wait out the proven refill margin (the same 1.3s budget
	// TestRateLimit_RefillsOverTime uses) before scraping.
	doRequest(t, s.mux, "/health")
	doRequest(t, s.mux, "/health")

	rejected := doRequest(t, s.mux, "/health")
	if rejected.Code != http.StatusTooManyRequests {
		t.Fatalf("third request: want 429, got %d", rejected.Code)
	}

	if rejected.Header().Get("Retry-After") == "" {
		t.Fatal("429 response missing Retry-After")
	}

	time.Sleep(1300 * time.Millisecond)

	body := doRequest(t, s.mux, "/health/metrics").Body.String()

	value, ok := findLineValue(body, "dashboard_ratelimit_429_total ")
	if !ok {
		t.Fatalf("exposition missing dashboard_ratelimit_429_total\n%s", body)
	}

	if n, err := strconv.Atoi(strings.TrimSpace(value)); err != nil || n != 1 {
		t.Errorf("dashboard_ratelimit_429_total = %q, want 1", value)
	}

	for _, want := range []string{
		"# TYPE dashboard_ratelimit_429_total counter",
		"# TYPE dashboard_ratelimit_retry_after_seconds histogram",
		`dashboard_ratelimit_retry_after_seconds_count 1`,
	} {
		if !strings.Contains(body, want+"\n") {
			t.Errorf("exposition missing %q\n%s", want, body)
		}
	}

	// Without WithRateLimit the whole family stays absent.
	s2 := setupDashboard(t, dashboard.WithMetrics(true))
	defer s2.cleanup()

	if body := doRequest(
		t,
		s2.mux,
		"/health/metrics",
	).Body.String(); strings.Contains(
		body,
		"dashboard_ratelimit_",
	) {
		t.Errorf("rate-limit family rendered without WithRateLimit:\n%s", body)
	}
}

// findLineValue returns the value part of the first exposition line with
// the given prefix (the external-package twin of the internal
// findMetricLine).
func findLineValue(body, prefix string) (string, bool) {
	for line := range strings.SplitSeq(body, "\n") {
		if value, ok := strings.CutPrefix(line, prefix); ok {
			return value, true
		}
	}

	return "", false
}
