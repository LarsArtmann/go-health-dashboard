package dashboard_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// bearerAuthMiddleware returns middleware that requires
// "Authorization: Bearer <token>" and rejects everything else with 401.
func bearerAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				http.Error(w, "dashboard: unauthorized", http.StatusUnauthorized)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func requestWithAuth(
	t *testing.T,
	handler http.Handler,
	target string,
	authorized bool,
) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()

	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}

	if authorized {
		r.Header.Set("Authorization", "Bearer secret-token")
	}

	handler.ServeHTTP(w, r)

	return w
}

func TestMiddleware_ProtectsDashboardRoute(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMiddleware(bearerAuthMiddleware("secret-token")))
	defer s.cleanup()

	w := requestWithAuth(t, s.mux, "/health", false)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated /health: want 401, got %d", w.Code)
	}

	w = requestWithAuth(t, s.mux, "/health", true)
	if w.Code != http.StatusOK {
		t.Errorf("authenticated /health: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("authenticated /health content-type: want text/html, got %s", ct)
	}
}

func TestMiddleware_ProtectsSSERoute(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMiddleware(bearerAuthMiddleware("secret-token")))
	defer s.cleanup()

	w := requestWithAuth(t, s.mux, "/health/sse", false)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated /health/sse: want 401, got %d", w.Code)
	}
}

func TestMiddleware_ProtectsFavicon(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMiddleware(bearerAuthMiddleware("secret-token")))
	defer s.cleanup()

	w := requestWithAuth(t, s.mux, "/favicon.svg", false)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated /favicon.svg: want 401, got %d", w.Code)
	}

	w = requestWithAuth(t, s.mux, "/favicon.svg", true)
	if w.Code != http.StatusOK {
		t.Errorf("authenticated /favicon.svg: want 200, got %d", w.Code)
	}
}

func TestMiddleware_ProbeEndpointsStayOpen(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithMiddleware(bearerAuthMiddleware("secret-token")))
	defer s.cleanup()

	for _, route := range []string{"/healthz", "/readyz", "/startupz"} {
		w := requestWithAuth(t, s.mux, route, false)
		if w.Code != http.StatusOK {
			t.Errorf(
				"unauthenticated %s: want 200 (kubelet cannot authenticate), got %d",
				route,
				w.Code,
			)
		}
	}
}

func TestMiddleware_HandlerReceivesUntouchedRequest(t *testing.T) {
	t.Parallel()

	var seenPath string

	recorder := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenPath = r.URL.Path

			w.Header().Set("X-Middleware", "ran")

			next.ServeHTTP(w, r)
		})
	}

	s := setupDashboard(t, dashboard.WithMiddleware(recorder))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if seenPath != "/health" {
		t.Errorf("handler saw path %q, want /health", seenPath)
	}

	if w.Header().Get("X-Middleware") != "ran" {
		t.Error("middleware header missing from response")
	}
}

func TestMiddleware_ComposedChain(t *testing.T) {
	t.Parallel()

	// compose wraps so that the first middleware listed runs first — the
	// last one listed runs closest to the handler.
	compose := func(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			for _, mw := range slices.Backward(mws) {
				next = mw(next)
			}

			return next
		}
	}

	var order []string

	tagging := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)

				next.ServeHTTP(w, r)
			})
		}
	}

	s := setupDashboard(t, dashboard.WithMiddleware(compose(tagging("first"), tagging("second"))))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("middleware order: want [first second], got %v", order)
	}
}

func TestMiddleware_ProtectsMetrics(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t,
		dashboard.WithMiddleware(bearerAuthMiddleware("secret-token")),
		dashboard.WithMetrics(true),
	)
	defer s.cleanup()

	w := requestWithAuth(t, s.mux, "/health/metrics", false)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated /health/metrics: want 401, got %d", w.Code)
	}

	w = requestWithAuth(t, s.mux, "/health/metrics", true)
	if w.Code != http.StatusOK {
		t.Errorf("authenticated /health/metrics: want 200, got %d", w.Code)
	}
}

func TestMiddleware_WithoutMiddlewareRoutesStayOpen(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	// /health/sse is deliberately absent: its streaming handler blocks until
	// the client disconnects, which a ResponseRecorder never does. SSE
	// rejection is covered by TestMiddleware_ProtectsSSERoute; the dashboard
	// and favicon handlers below prove middleware-free routes stay open.
	for _, route := range []string{"/health", "/favicon.svg"} {
		w := doRequest(t, s.mux, route)
		if w.Code != http.StatusOK {
			t.Errorf("%s: want 200 without middleware, got %d", route, w.Code)
		}
	}
}

// --- Per-IP rate limiting (M16) -----------------------------------------

// Per-IP limiting must isolate addresses: exhausting one IP's budget never
// touches another's, unlike the shared WithRateLimit bucket. Ports don't
// split the budget (the key is the host), and rejections feed the same
// ratelimit metrics family as the shared limiter (M14).
func TestPerIPRateLimit_IsolatesAddresses(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t,
		dashboard.WithMetrics(true),
		dashboard.WithPerIPRateLimit(2, time.Second),
	)
	defer s.cleanup()

	request := func(remote string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = remote

		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		return w
	}

	if w := request("10.0.0.1:1000"); w.Code != http.StatusOK {
		t.Fatalf("first request from .1: want 200, got %d", w.Code)
	}

	if w := request("10.0.0.1:1001"); w.Code != http.StatusOK {
		t.Fatalf("second request from .1 (new port, same host): want 200, got %d", w.Code)
	}

	rejected := request("10.0.0.1:1002")
	if rejected.Code != http.StatusTooManyRequests {
		t.Fatalf("third request from .1: want 429, got %d", rejected.Code)
	}

	if rejected.Header().Get("Retry-After") == "" {
		t.Fatal("429 missing Retry-After")
	}

	if w := request("10.0.0.2:1000"); w.Code != http.StatusOK {
		t.Fatalf("first request from .2: want 200 (budgets are per-address), got %d", w.Code)
	}

	// The rejection rode the shared M14 family (counter + histogram both
	// present because WithMetrics is on and a limiter is configured).
	metrics := doRequest(t, s.mux, "/health/metrics").Body.String()
	if !strings.Contains(metrics, "dashboard_ratelimit_429_total 1\n") {
		t.Errorf("metrics missing the per-IP rejection:\n%s", metrics)
	}
}

// The per-address map must not grow without bound: addresses beyond the
// sweep threshold with idle buckets are evicted, and an evicted address
// simply gets a fresh bucket.
func TestPerIPRateLimit_SweepsIdleBuckets(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithPerIPRateLimit(5, 30*time.Millisecond))
	defer s.cleanup()

	probe := func(remote string) int {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = remote

		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		return w.Code
	}

	for i := range 5000 {
		code := probe("10.1." + string(rune('0'+i%10)) + "." + string(rune('0'+i/10%10)) + ":9")
		if code == http.StatusTooManyRequests {
			t.Fatalf("request %d rejected before exhausting any single budget", i)
		}
	}
}

// Probes stay exempt from per-IP limiting: the kubelet cannot authenticate,
// and gating breaks liveness.
func TestPerIPRateLimit_ProbesExempt(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithPerIPRateLimit(1, time.Second))
	defer s.cleanup()

	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.RemoteAddr = "10.9.9.9:1"

		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("healthz with per-IP limiter exhausted elsewhere: want 200, got %d", w.Code)
		}
	}
}

// --- Request logging (M17) ----------------------------------------------

// WithRequestLogging emits one structured line per dashboard-owned request
// carrying method, path, status, and duration; it stays off by default and
// never logs probe endpoints.
func TestRequestLogging_RecordsDashboardRequests(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	s := setupDashboard(t, dashboard.WithRequestLogging(slog.New(slog.NewTextHandler(&buf, nil))))
	defer s.cleanup()

	doRequest(t, s.mux, "/health")

	line := buf.String()
	for _, want := range []string{"method=GET", "path=/health", "status=200", "duration="} {
		if !strings.Contains(line, want) {
			t.Errorf("log line missing %q: %s", want, line)
		}
	}

	buf.Reset()

	doRequest(t, s.mux, "/healthz")

	if buf.Len() != 0 {
		t.Errorf("probe endpoints must not be logged, got: %s", buf.String())
	}
}

// Rejections are logged too (the logger wraps the limiters), and the
// recorded status is the 429 the client actually received.
func TestRequestLogging_RecordsRejections(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	s := setupDashboard(t,
		dashboard.WithRequestLogging(slog.New(slog.NewTextHandler(&buf, nil))),
		dashboard.WithRateLimit(1, time.Second),
	)
	defer s.cleanup()

	doRequest(t, s.mux, "/health")

	rejected := doRequest(t, s.mux, "/health")
	if rejected.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: want 429, got %d", rejected.Code)
	}

	if !strings.Contains(buf.String(), "status=429") {
		t.Errorf("log line missing the rejection status: %s", buf.String())
	}
}

// SSE streaming must keep flushing through the logging recorder (the
// pusher's patches and heartbeats flush incrementally).
func TestRequestLogging_PreservesSSEFlush(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	s := setupDashboard(t,
		dashboard.WithRequestLogging(slog.New(slog.NewTextHandler(&buf, nil))),
		dashboard.WithMetrics(true),
	)
	defer s.cleanup()

	srv := httptest.NewServer(s.mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	chunk := []byte("scan-me: the initial SSE patch must arrive through the logger")

	if _, err := resp.Body.Read(chunk); err != nil {
		t.Fatalf("read initial patch: %v", err)
	}
}
