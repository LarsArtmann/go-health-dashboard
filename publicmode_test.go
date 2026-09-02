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
