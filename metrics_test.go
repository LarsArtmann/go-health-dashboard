package dashboard_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
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

	body := w.Body.String()

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
		"# TYPE dashboard_pusher_active gauge",
		"dashboard_pusher_active 1",
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

	first := doRequest(t, s.mux, "/health/metrics").Body.String()

	for range 5 {
		second := doRequest(t, s.mux, "/health/metrics").Body.String()
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

	dash := dashboard.New(probe, dashboard.WithMetrics(true))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	defer func() {
		dash.Shutdown()
		probe.Shutdown()
	}()

	w := doRequest(t, mux, "/health/metrics")
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
