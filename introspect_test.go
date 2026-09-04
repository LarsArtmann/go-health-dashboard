package dashboard_test

import (
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// TestIntrospection_ServesResolvedConfig verifies the endpoint reports the
// dashboard's own version, the resolved routes, and the configured modes.
func TestIntrospection_ServesResolvedConfig(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t,
		dashboard.WithTrend(42),
		dashboard.WithMetrics(true),
		dashboard.WithRateLimit(10, 1<<20),
		dashboard.WithIntrospection(),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/introspect")
	if w.Code != http.StatusOK {
		t.Fatalf("introspection: want 200, got %d: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}

	var doc struct {
		Version   string            `json:"version"`
		GoVersion string            `json:"go_version"`
		Routes    map[string]string `json:"routes"`
		Limits    struct {
			MaxSSEConnections int    `json:"max_sse_connections"`
			RateLimitEnabled  bool   `json:"rate_limit_enabled"`
			ShutdownDrain     string `json:"shutdown_drain"`
		} `json:"limits"`
		Modes struct {
			PushMode      string `json:"push_mode"`
			PublicMode    bool   `json:"public_mode"`
			Metrics       bool   `json:"metrics"`
			Webhook       bool   `json:"webhook"`
			TrendSamples  int    `json:"trend_samples"`
			NonceStrategy string `json:"nonce_strategy"`
		} `json:"modes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("introspection is not valid JSON: %v\n%s", err, w.Body.String())
	}

	if doc.Version != dashboard.Version {
		t.Errorf("version: want %q, got %q", dashboard.Version, doc.Version)
	}

	if doc.GoVersion == "" {
		t.Error("go_version is empty")
	}

	for _, route := range []string{"dashboard", "sse", "trend", "export", "metrics"} {
		if doc.Routes[route] == "" {
			t.Errorf("routes.%s missing for an enabled feature", route)
		}
	}

	if doc.Modes.TrendSamples != 42 {
		t.Errorf("modes.trend_samples: want 42, got %d", doc.Modes.TrendSamples)
	}

	if doc.Modes.PushMode != "on-change" {
		t.Errorf("modes.push_mode: want on-change, got %q", doc.Modes.PushMode)
	}

	if !doc.Modes.Metrics {
		t.Error("modes.metrics: want true after WithMetrics(true)")
	}

	if !doc.Limits.RateLimitEnabled {
		t.Error("limits.rate_limit_enabled: want true after WithRateLimit")
	}

	if doc.Modes.NonceStrategy != "none" {
		t.Errorf("modes.nonce_strategy: want none, got %q", doc.Modes.NonceStrategy)
	}
}

// TestIntrospection_DisabledByDefault pins the opt-in contract: without
// WithIntrospection the route must not exist.
func TestIntrospection_DisabledByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health/introspect"); w.Code != http.StatusNotFound {
		t.Errorf("introspection without WithIntrospection: want 404, got %d", w.Code)
	}
}

// TestIntrospection_NeverLeaksCheckData verifies the document stays
// configuration-only even when the probe carries hostile check names.
func TestIntrospection_NeverLeaksCheckData(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t, dashboard.WithIntrospection())
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/introspect")
	if w.Code != http.StatusOK {
		t.Fatalf("introspection: want 200, got %d", w.Code)
	}

	body := w.Body.String()
	for _, leak := range []string{"cache", "queue", "error", "fail"} {
		if strings.Contains(body, leak) {
			t.Errorf("introspection body contains check-derived data %q:\n%s", leak, body)
		}
	}
}
