package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			t.Errorf("unauthenticated %s: want 200 (kubelet cannot authenticate), got %d", route, w.Code)
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
			for i := len(mws) - 1; i >= 0; i-- {
				next = mws[i](next)
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
