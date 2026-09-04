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
		health.WithRefreshInterval(100*time.Millisecond),
	)

	dash := dashboard.New(probe, opts...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	return &probeSetup{
		probe: probe,
		dash:  dash,
		mux:   mux,
		cleanup: func() {
			dash.Shutdown()
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
		health.WithRefreshInterval(100*time.Millisecond),
	)

	dash := dashboard.New(probe, opts...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	return &probeSetup{
		probe: probe,
		dash:  dash,
		mux:   mux,
		cleanup: func() {
			dash.Shutdown()
			probe.Shutdown()
		},
	}
}

func doRequest(
	t *testing.T,
	handler http.Handler,
	target string,
) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()

	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, r)

	return w
}

// --- HTML page rendering tests ---.

func TestHandler_RendersHTMLDashboard(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

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

func TestHTML_ContainsAlertBanner(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()
	if !strings.Contains(body, "All Systems Operational") {
		t.Error("HTML should contain status text for healthy system")
	}
}

func TestHTML_ContainsStatCards(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

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

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()
	for _, name := range []string{"database", "redis"} {
		if !strings.Contains(body, name) {
			t.Errorf("HTML should contain service name %q", name)
		}
	}
}

func TestHTML_WithFailures_ShowsFailureContent(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()
	if !strings.Contains(body, "Degraded") {
		t.Error("HTML should show degraded status for non-critical failures")
	}

	if !strings.Contains(body, "cache") {
		t.Error("HTML should list the failing 'cache' service")
	}
}

// --- Datastar integration tests ---.

func TestHTML_ContainsDatastarSDK(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()
	// Datastar SDK script loads from CDN with a type="module" attribute
	if !strings.Contains(body, "datastar") {
		t.Error("HTML should load the Datastar SDK script")
	}
}

func TestHTML_ContainsLiveRegionWithSSEURL(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()
	if !strings.Contains(body, "health-region") {
		t.Error("HTML should contain the LiveRegion div with id=health-region")
	}

	if !strings.Contains(body, "/health/sse") {
		t.Error("HTML should contain the SSE endpoint URL in the LiveRegion")
	}
}

func TestHTML_DoesNotContainHTMX(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()
	// No HTMX script tags or hx-* attributes
	if strings.Contains(body, "htmx.org") || strings.Contains(body, "htmx.min.js") {
		t.Error("HTML should not load HTMX — Datastar handles real-time")
	}
}

// --- JSON probe endpoint tests ---.

func TestReadiness_ReturnsJSON(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/readyz")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("readiness content-type: want application/json, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"status"`) {
		t.Error("JSON response should contain health status")
	}
}

func TestLiveness_ReturnsJSON(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/healthz")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("liveness content-type: want application/json, got %s", ct)
	}
}

// --- Content negotiation tests ---.

func doRequestWithAccept(
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
	r.Header.Set("Accept", accept)

	handler.ServeHTTP(w, r)

	return w
}

func TestContentNegotiation_AcceptHeaderSelectsContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		accept string
		wantCT string
	}{
		{name: "json accept returns json", accept: "application/json", wantCT: "application/json"},
		{name: "html accept returns html", accept: "text/html", wantCT: "text/html"},
		{
			name:   "q-value prefers json",
			accept: "application/json;q=0.9, text/html;q=0.8",
			wantCT: "application/json",
		},
		{
			name:   "q-value prefers html",
			accept: "text/html;q=1.0, application/json;q=0.1",
			wantCT: "text/html",
		},
		{name: "wildcard returns html default", accept: "*/*", wantCT: "text/html"},
		{
			name:   "equal q-values return html default",
			accept: "application/json, text/html",
			wantCT: "text/html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := setupDashboard(t)
			defer s.cleanup()

			w := doRequestWithAccept(t, s.mux, "/health", tt.accept)

			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, tt.wantCT) {
				t.Errorf("content-type: want %s, got %s", tt.wantCT, ct)
			}
		})
	}
}

func TestContentNegotiation_JSONAcceptReturnsJSONBody(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequestWithAccept(t, s.mux, "/health", "application/json")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type: want application/json, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"status"`) {
		t.Error("JSON response should contain status field")
	}
}

func TestContentNegotiation_NoAcceptReturnsHTML(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type: want text/html, got %s", ct)
	}
}

func TestContentNegotiation_JSONHealthyReturns200(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequestWithAccept(t, s.mux, "/health", "application/json")

	if w.Code != http.StatusOK {
		t.Errorf("healthy JSON status: want 200, got %d", w.Code)
	}
}

func TestContentNegotiation_JSONCriticalFailReturns503(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideUnhealthy(injector, "database", "connection refused")
	invoke[*unhealthyService](t, injector, "database")

	probe := health.New(injector,
		health.WithCriticalServices("database"),
		health.WithRefreshInterval(100*time.Millisecond),
	)

	dash := dashboard.New(probe)

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

	w := doRequestWithAccept(t, mux, "/health", "application/json")

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("critical failure JSON status: want 503, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), `"fail"`) {
		t.Error("JSON response should contain fail status")
	}
}

// --- SSE endpoint tests ---.

func TestSSE_EndpointRegisteredAndStreams(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	// SSE handler blocks; use a short-timeout context so it returns.
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	s.mux.ServeHTTP(w, r)

	if w.Code == http.StatusNotFound {
		t.Error("SSE endpoint should be registered")
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("SSE content-type: want text/event-stream, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Error("SSE response should contain Datastar patch event type")
	}
}

func TestSSE_SendsInitialState(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	s.mux.ServeHTTP(w, r)

	body := w.Body.String()
	// Initial state should contain the health-region selector for patching
	if !strings.Contains(body, "#health-region") {
		t.Error("SSE initial state should target #health-region element")
	}
}

// --- Options tests ---.

func TestWithTitle_CustomTitle(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithTitle("My Awesome Service"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	if !strings.Contains(w.Body.String(), "My Awesome Service") {
		t.Error("HTML should contain custom title")
	}
}

func TestWithPushMode_AcceptsBothModes(t *testing.T) {
	t.Parallel()

	t.Run("on-change", func(t *testing.T) {
		t.Parallel()
		s := setupDashboard(t, dashboard.WithPushMode(dashboard.PushOnChange))
		defer s.cleanup()
		// Just verify it doesn't panic
	})

	t.Run("always", func(t *testing.T) {
		t.Parallel()
		s := setupDashboard(t, dashboard.WithPushMode(dashboard.PushAlways))
		defer s.cleanup()
		// Just verify it doesn't panic
	})
}

// --- Routes tests ---.

func TestDefaultRoutes_ConventionalPaths(t *testing.T) {
	t.Parallel()

	routes := dashboard.DefaultRoutes()

	if routes.Dashboard != "/health" {
		t.Errorf("dashboard route: want /health, got %s", routes.Dashboard)
	}

	if routes.SSE != "/health/sse" {
		t.Errorf("SSE route: want /health/sse, got %s", routes.SSE)
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

func TestWithRoutes_CustomPaths(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	customRoutes := dashboard.Routes{
		Dashboard: "/status",
		SSE:       "/status/sse",
		Liveness:  "/live",
		Readiness: "/ready",
		Startup:   "/started",
	}

	probe := health.New(injector, health.WithCriticalServices("db"))
	dash := dashboard.New(probe, dashboard.WithRoutes(customRoutes))

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

	// Custom dashboard route should work
	w := doRequest(t, mux, "/status")
	if w.Code != http.StatusOK {
		t.Errorf("custom dashboard route: want 200, got %d", w.Code)
	}

	// Default route should NOT be registered
	w2 := doRequest(t, mux, "/health")
	if w2.Code != http.StatusNotFound {
		t.Errorf("default route should not exist: want 404, got %d", w2.Code)
	}
}

func TestRegisterRoutes_AllRoutesRespond(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	for _, path := range []string{"/health", "/healthz", "/readyz", "/startupz"} {
		w := doRequest(t, s.mux, path)
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

	w := doRequest(t, s.mux, "/health")

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

	w := doRequest(t, s.mux, "/readyz")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readiness after shutdown: want 503, got %d", w.Code)
	}
}

// --- Version constant test ---.

func TestVersion_IsNotEmpty(t *testing.T) {
	t.Parallel()

	if dashboard.Version == "" {
		t.Error("Version should be a non-empty semver string")
	}
}

// --- Benchmarks ---.

func BenchmarkHandler_HTMLRendering(b *testing.B) {
	injector := do.New()
	do.ProvideNamed(injector, "db", func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
	do.ProvideNamed(injector, "cache", func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
	do.MustInvokeNamed[*healthyService](injector, "db")
	do.MustInvokeNamed[*healthyService](injector, "cache")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe)

	handler := dash.Handler()

	ctx := context.Background()
	r, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/health", nil)

	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		handler(w, r)
	}
}

func TestFavicon_ReturnsSVG(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/favicon.svg")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("content-type: want image/svg+xml, got %s", ct)
	}

	if !strings.HasPrefix(w.Body.String(), "<svg") {
		t.Error("favicon body should start with <svg")
	}
}

// --- SubscriberCount not-started test ---.

func TestSubscriberCount_ZeroWhenNotStarted(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe)

	if count := dash.SubscriberCount(); count != 0 {
		t.Errorf("SubscriberCount before Start: want 0, got %d", count)
	}
}

// --- WithBasePath tests ---.

func TestWithBasePath_PrefixesAllRoutes(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithBasePath("/admin"))
	defer s.cleanup()

	for _, path := range []string{
		"/admin/health",
		"/admin/healthz",
		"/admin/readyz",
		"/admin/startupz",
		"/admin/favicon.svg",
	} {
		w := doRequest(t, s.mux, path)
		if w.Code == http.StatusNotFound {
			t.Errorf("prefixed route %s should be registered", path)
		}
	}

	for _, path := range []string{"/health", "/healthz", "/readyz"} {
		w := doRequest(t, s.mux, path)
		if w.Code != http.StatusNotFound {
			t.Errorf("unprefixed route %s should NOT be registered: want 404, got %d", path, w.Code)
		}
	}
}

func TestWithBasePath_SSEURLInHTML(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithBasePath("/admin"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/admin/health")

	body := w.Body.String()
	if !strings.Contains(body, "/admin/health/sse") {
		t.Error("HTML should reference the prefixed SSE URL /admin/health/sse")
	}

	if strings.Contains(body, "\"/health/sse\"") {
		t.Error("HTML should not reference the unprefixed SSE URL /health/sse")
	}
}

// --- WithBasePath / WithRoutes ordering interaction tests ---.

func TestWithBasePath_AfterWithRoutes(t *testing.T) {
	t.Parallel()

	custom := dashboard.Routes{
		Dashboard: "/status",
		SSE:       "/status/sse",
		Favicon:   "/icon.svg",
		Liveness:  "/live",
		Readiness: "/ready",
		Startup:   "/start",
	}

	s := setupDashboard(t,
		dashboard.WithRoutes(custom),
		dashboard.WithBasePath("/admin"),
	)
	defer s.cleanup()

	for _, path := range []string{
		"/admin/status",
		"/admin/live",
		"/admin/ready",
		"/admin/start",
		"/admin/icon.svg",
	} {
		w := doRequest(t, s.mux, path)
		if w.Code == http.StatusNotFound {
			t.Errorf("prefixed custom route %s should be registered", path)
		}
	}
}

func TestWithRoutes_AfterWithBasePath(t *testing.T) {
	t.Parallel()

	custom := dashboard.Routes{
		Dashboard: "/status",
		SSE:       "/status/sse",
		Favicon:   "/icon.svg",
		Liveness:  "/live",
		Readiness: "/ready",
		Startup:   "/start",
	}

	// BasePath is applied once after ALL options run, so option order no
	// longer matters (historically WithRoutes after WithBasePath silently
	// dropped the prefix). Both orders must yield the same prefixed set.
	s := setupDashboard(t,
		dashboard.WithBasePath("/admin"),
		dashboard.WithRoutes(custom),
	)
	defer s.cleanup()

	reversed := setupDashboard(t,
		dashboard.WithRoutes(custom),
		dashboard.WithBasePath("/admin"),
	)
	defer reversed.cleanup()

	want := []string{
		"/admin/status",
		"/admin/status/sse",
		"/admin/live",
		"/admin/ready",
		"/admin/start",
	}
	for _, dash := range []*dashboard.Dashboard{s.dash, reversed.dash} {
		if got := dash.Routes(); got.Dashboard != want[0] || got.SSE != want[1] ||
			got.Liveness != want[2] {
			t.Errorf(
				"resolved routes: want dashboard=%s sse=%s liveness=%s, got %+v",
				want[0],
				want[1],
				want[2],
				got,
			)
		}
	}

	for _, path := range want[:1] {
		if w := doRequest(t, s.mux, path); w.Code == http.StatusNotFound {
			t.Errorf("prefixed custom route %s should be registered", path)
		}
		if w := doRequest(t, reversed.mux, path); w.Code == http.StatusNotFound {
			t.Errorf("prefixed custom route %s should be registered (reverse order)", path)
		}
	}

	// The unprefixed custom routes must NOT be registered — the prefix won.
	for _, mux := range []*http.ServeMux{s.mux, reversed.mux} {
		if w := doRequest(t, mux, "/status"); w.Code != http.StatusNotFound {
			t.Errorf("unprefixed /status should NOT be registered: want 404, got %d", w.Code)
		}
	}
}

// setupDashboardWithProber is setupDashboard for an arbitrary Prober — used
// by the aggregate browser test to render a merged multi-probe page.
func setupDashboardWithProber(
	t *testing.T,
	probe dashboard.Prober,
	opts ...dashboard.Option,
) *probeSetup {
	t.Helper()

	dash := dashboard.New(probe, opts...)

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	return &probeSetup{
		probe:   nil,
		dash:    dash,
		mux:     mux,
		cleanup: func() { dash.Shutdown() },
	}
}
