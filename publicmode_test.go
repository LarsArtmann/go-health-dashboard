package dashboard_test

import (
	"net/http"
	"strings"
	"testing"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

func TestPublicMode_AnonymizesHTML(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t,
		dashboard.WithMetrics(true),
		dashboard.WithPublicMode(),
		dashboard.WithDescription("Demo status page"),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	if !strings.Contains(body, "Demo status page") {
		t.Error("description missing from HTML")
	}

	if !strings.Contains(body, `property="og:description"`) {
		t.Error("og:description missing from HTML")
	}

	if strings.Contains(body, "connection refused") {
		t.Error("public mode leaked an error message into the HTML")
	}

	if !strings.Contains(body, "check-") {
		t.Error("public mode did not render generic check labels")
	}
}

func TestPublicMode_AnonymizesMetrics(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t,
		dashboard.WithMetrics(true),
		dashboard.WithPublicMode(),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/metrics")
	body := w.Body.String()

	if strings.Contains(body, `check="cache"`) || strings.Contains(body, `check="queue"`) {
		t.Error("public mode leaked real check names into metrics")
	}

	if !strings.Contains(body, `check="check-1"`) {
		t.Error("metrics missing generic check labels in public mode")
	}
}

func TestPublicMode_KeepsRealNamesWithoutOptIn(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	if !strings.Contains(w.Body.String(), "cache") {
		t.Error("default mode should show real check names")
	}
}

func TestDescription_OmittedByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	if strings.Contains(w.Body.String(), "og:description") {
		t.Error("og:description rendered without WithDescription")
	}
}

func TestWithMiddleware_NonWrappedProbes(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/healthz"); w.Code != http.StatusOK {
		t.Errorf("probe endpoint should keep working, got %d", w.Code)
	}
}

// TestPublicMode_LeakScanner sweeps every public surface — HTML, metrics,
// and the JSON health response — for the real service names and error
// strings that public mode must never disclose. This is the regression
// scanner for the 2026-09-04 pin incident class: presentation-layer leaks
// that only appear in rendered output.
func TestPublicMode_LeakScanner(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t,
		dashboard.WithMetrics(true),
		dashboard.WithPublicMode(),
	)
	defer s.cleanup()

	// The exact secrets public mode must keep private: the real service
	// names chosen by setupDashboardWithFailures and the error strings
	// the unhealthy services carry.
	secrets := []string{"database", "cache", "queue", "connection refused", "timeout"}

	for _, path := range []string{"/health", "/health/metrics"} {
		w := doRequest(t, s.mux, path)
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: unexpected status %d", path, w.Code)
		}

		body := w.Body.String()
		for _, secret := range secrets {
			if strings.Contains(body, secret) {
				t.Errorf("%s leaked %q in public mode:\n%.200s", path, secret, body)
			}
		}
	}

	// And the documented boundary holds: probes stay verbatim JSON in
	// public mode (the kubelet is trusted infrastructure).
	if w := doRequest(t, s.mux, "/readyz"); !strings.Contains(w.Body.String(), `"cache"`) {
		t.Error("readyz should stay verbatim in public mode, but cache name is missing")
	}
}
