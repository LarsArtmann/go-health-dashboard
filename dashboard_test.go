package dashboard_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// --- Test service types ---.

type healthyService struct{}

var _ do.HealthcheckerWithContext = (*healthyService)(nil)

func (healthyService) HealthCheck(_ context.Context) error { return nil }

type unhealthyService struct{ reason string }

var _ do.HealthcheckerWithContext = (*unhealthyService)(nil)

func (u *unhealthyService) HealthCheck(_ context.Context) error {
	return fmt.Errorf("service unhealthy: %s", u.reason)
}

// --- Test helpers ---.

func provideHealthy(i do.Injector, name string) {
	do.ProvideNamed(i, name, func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
}

func provideUnhealthy(i do.Injector, name, reason string) {
	do.ProvideNamed(i, name, func(_ do.Injector) (*unhealthyService, error) {
		return &unhealthyService{reason: reason}, nil
	})
}

func invoke[T any](t *testing.T, i do.Injector, name string) T {
	t.Helper()
	return do.MustInvokeNamed[T](i, name)
}

type probeSetup struct {
	probe   *health.Probe
	dash    *dashboard.Dashboard
	mux     *http.ServeMux
	cleanup func()
}

func setupDashboard(t *testing.T, opts ...dashboard.Option) *probeSetup {
	t.Helper()

	injector := do.New()
	provideHealthy(injector, "database")
	provideHealthy(injector, "redis")
	invoke[*healthyService](t, injector, "database")
	invoke[*healthyService](t, injector, "redis")

	probe := health.New(injector,
		health.WithVersion("1.0.0"),
		health.WithCriticalServices("database"),
	)

	dash := dashboard.New(probe, opts...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	return &probeSetup{
		probe: probe,
		dash:  dash,
		mux:   mux,
		cleanup: func() {
			probe.Shutdown()
		},
	}
}

func setupDashboardWithFailures(t *testing.T, opts ...dashboard.Option) *probeSetup {
	t.Helper()

	injector := do.New()
	provideHealthy(injector, "database")
	provideUnhealthy(injector, "cache", "connection refused")
	provideUnhealthy(injector, "queue", "timeout")
	invoke[*healthyService](t, injector, "database")
	invoke[*unhealthyService](t, injector, "cache")
	invoke[*unhealthyService](t, injector, "queue")

	probe := health.New(injector,
		health.WithVersion("2.1.0"),
		health.WithCriticalServices("database"),
	)

	dash := dashboard.New(probe, opts...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	return &probeSetup{
		probe: probe,
		dash:  dash,
		mux:   mux,
		cleanup: func() {
			probe.Shutdown()
		},
	}
}

func doRequest(
	t *testing.T,
	handler http.Handler,
	target, accept string,
) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()

	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}

	if accept != "" {
		r.Header.Set("Accept", accept)
	}

	handler.ServeHTTP(w, r)

	return w
}

// --- Content negotiation tests ---.

func TestHandler_HTMLRequest_RendersDashboard(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type: want text/html, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Health Dashboard") {
		t.Error("HTML response should contain the dashboard title")
	}
}

func TestHandler_JSONRequest_DelegatesToProbe(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "application/json")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type: want application/json, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"status"`) {
		t.Error("JSON response should contain health status")
	}
}

func TestHandler_MissingAcceptHeader_DelegatesToJSON(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("no Accept header: want JSON, got %s", ct)
	}
}

func TestHandler_WildcardAccept_RendersHTML(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "*/*")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("*/* Accept: want HTML, got %s", ct)
	}
}

func TestHandler_BrowserAcceptPattern_RendersHTML(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	// Real browser Accept header
	browserAccept := "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

	w := doRequest(t, s.mux, "/health", browserAccept)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("browser Accept: want HTML, got %s", ct)
	}
}

// --- HTML output validation tests ---.

func TestHTML_ContainsAlertBanner(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "All Systems Operational") {
		t.Error("HTML should contain status text for healthy system")
	}
}

func TestHTML_ContainsStatCards(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "1.0.0") {
		t.Error("HTML should contain version from StatCard")
	}

	if !strings.Contains(body, "Uptime") {
		t.Error("HTML should contain uptime StatCard label")
	}
}

func TestHTML_ContainsServiceNames(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	for _, name := range []string{"database", "redis"} {
		if !strings.Contains(body, name) {
			t.Errorf("HTML should contain service name %q", name)
		}
	}
}

func TestHTML_ContainsPolledRegion(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "hx-get") {
		t.Error("HTML should contain HTMX polling attributes")
	}

	if !strings.Contains(body, "/health/partial") {
		t.Error("HTML should contain partial URL for polling")
	}
}

func TestHTML_WithFailures_ShowsFailureContent(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t)
	defer s.cleanup()

	// The critical service (database) is healthy, but non-critical ones fail
	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "Degraded") {
		t.Error("HTML should show degraded status for non-critical failures")
	}

	if !strings.Contains(body, "cache") {
		t.Error("HTML should list the failing 'cache' service")
	}
}

// --- Partial handler tests ---.

func TestPartialHandler_ReturnsHTMLFragment(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/partial", "text/html")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type: want text/html, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "hx-get") {
		t.Error("Partial should contain PolledRegion wrapper with hx-get")
	}
}

// --- Options tests ---.

func TestWithTitle_CustomTitle(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithTitle("My Awesome Service"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	if !strings.Contains(w.Body.String(), "My Awesome Service") {
		t.Error("HTML should contain custom title")
	}
}

func TestWithRefreshInterval_CustomPolling(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithRefreshInterval(5*time.Second))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "every 5s") {
		t.Errorf(
			"HTML should contain 5s polling interval, got body snippet: %s",
			body[strings.Index(body, "hx-trigger"):min(len(body), strings.Index(body, "hx-trigger")+40)],
		)
	}
}

func TestWithRefreshMode_Off_NoPolling(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithRefreshMode(dashboard.RefreshModeOff))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if strings.Contains(body, "hx-get") {
		t.Error("RefreshModeOff should not render PolledRegion")
	}
}

func TestWithRoutes_CustomPaths(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	dash := dashboard.New(probe, dashboard.WithRoutes(dashboard.Routes{
		Dashboard: "/status",
		Partial:   "/status/partial",
		SSE:       "/status/sse",
		Liveness:  "/live",
		Readiness: "/ready",
		Startup:   "/started",
	}))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.Routes{
		Dashboard: "/status",
		Partial:   "/status/partial",
		SSE:       "/status/sse",
		Liveness:  "/live",
		Readiness: "/ready",
		Startup:   "/started",
	})
	defer probe.Shutdown()

	// Custom dashboard route should work
	w := doRequest(t, mux, "/status", "text/html")
	if w.Code != http.StatusOK {
		t.Errorf("custom dashboard route: want 200, got %d", w.Code)
	}

	// Default route should NOT be registered
	w2 := doRequest(t, mux, "/health", "text/html")
	if w2.Code != http.StatusNotFound {
		t.Errorf("default route should not exist: want 404, got %d", w2.Code)
	}
}

// --- RegisterRoutes tests ---.

func TestRegisterRoutes_AllRoutesRespond(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	for _, path := range []string{"/health", "/health/partial", "/healthz", "/readyz", "/startupz"} {
		w := doRequest(t, s.mux, path, "application/json")
		if w.Code == http.StatusNotFound {
			t.Errorf("route %s should be registered", path)
		}
	}
}

// --- Shutdown state tests ---.

func TestShutdownState_ShowsShuttingDown(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	s.probe.Shutdown()

	w := doRequest(t, s.mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "Shutting Down") {
		t.Error("HTML should show shutdown message")
	}
}

func TestShutdownState_ReadinessReturns503(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	s.probe.Shutdown()

	w := doRequest(t, s.mux, "/readyz", "application/json")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readiness after shutdown: want 503, got %d", w.Code)
	}
}

// --- SSE mode tests ---.

func TestSSEMode_RegisterSSEEndpoint(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(50*time.Millisecond),
	)
	defer probe.Shutdown()

	dash := dashboard.New(probe,
		dashboard.WithRefreshMode(dashboard.RefreshModeSSE),
		dashboard.WithRefreshInterval(50*time.Millisecond),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer dash.Shutdown()

	// SSE handler blocks; use a short-timeout context so it returns.
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	mux.ServeHTTP(w, r)

	// Should not be 404 — the endpoint exists and started streaming
	if w.Code == http.StatusNotFound {
		t.Error("SSE endpoint should be registered in SSE mode")
	}

	// Should have received SSE content type and at least the initial event
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("SSE content-type: want text/event-stream, got %s", ct)
	}
}

func TestSSEMode_HTMLContainsSSEScript(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe,
		dashboard.WithRefreshMode(dashboard.RefreshModeSSE),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer dash.Shutdown()

	w := doRequest(t, mux, "/health", "text/html")

	body := w.Body.String()
	if !strings.Contains(body, "EventSource") {
		t.Error("SSE mode HTML should contain EventSource script")
	}

	if !strings.Contains(body, "/health/sse") {
		t.Error("SSE mode HTML should contain SSE endpoint URL")
	}
}

func TestSSEMode_DoesNotContainHTMXPolling(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe,
		dashboard.WithRefreshMode(dashboard.RefreshModeSSE),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer dash.Shutdown()

	w := doRequest(t, mux, "/health", "text/html")

	body := w.Body.String()
	// Should NOT have HTMX polling attributes in the health region
	if strings.Contains(body, "data-sse-url") && strings.Contains(body, "hx-trigger") {
		t.Error("SSE mode should not use HTMX polling for the health region")
	}
}

// --- Default routes tests ---.

func TestDefaultRoutes_ConventionalPaths(t *testing.T) {
	t.Parallel()

	routes := dashboard.DefaultRoutes()

	if routes.Dashboard != "/health" {
		t.Errorf("dashboard route: want /health, got %s", routes.Dashboard)
	}

	if routes.Partial != "/health/partial" {
		t.Errorf("partial route: want /health/partial, got %s", routes.Partial)
	}

	if routes.Liveness != "/healthz" {
		t.Errorf("liveness route: want /healthz, got %s", routes.Liveness)
	}

	if routes.Readiness != "/readyz" {
		t.Errorf("readiness route: want /readyz, got %s", routes.Readiness)
	}

	if routes.Startup != "/startupz" {
		t.Errorf("startup route: want /startupz, got %s", routes.Startup)
	}
}

// --- Benchmark ---.

func BenchmarkHandler_HTMLRendering(b *testing.B) {
	injector := do.New()
	provideHealthy(injector, "db")
	provideHealthy(injector, "cache")
	invoke[*healthyService](&testing.T{}, injector, "db")
	invoke[*healthyService](&testing.T{}, injector, "cache")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe)

	handler := dash.Handler()

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	r.Header.Set("Accept", "text/html")

	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		handler(w, r)
	}
}

func BenchmarkPartialHandler_Rendering(b *testing.B) {
	injector := do.New()
	provideHealthy(injector, "db")
	provideHealthy(injector, "cache")
	invoke[*healthyService](&testing.T{}, injector, "db")
	invoke[*healthyService](&testing.T{}, injector, "cache")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe)

	handler := dash.PartialHandler()

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health/partial", nil)

	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		handler(w, r)
	}
}
