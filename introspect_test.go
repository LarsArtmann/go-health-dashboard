package dashboard_test

import (
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// introspectionDoc mirrors the JSON shape served by IntrospectionHandler.
type introspectionDoc struct {
	Version   string            `json:"version"`
	GoVersion string            `json:"go_version"`
	Routes    map[string]string `json:"routes"`
	Limits struct {
		MaxSSEConnections int    `json:"max_sse_connections"`
		RateLimitEnabled  bool   `json:"rate_limit_enabled"`
		ShutdownDrain     string `json:"shutdown_drain"`
		TimelineMaxAge    string `json:"timeline_max_age"`
	} `json:"limits"`
	Modes struct {
		PushMode      string `json:"push_mode"`
		PublicMode    bool   `json:"public_mode"`
		Metrics       bool   `json:"metrics"`
		Webhook       bool   `json:"webhook"`
		TrendSamples  int    `json:"trend_samples"`
		NonceStrategy string `json:"nonce_strategy"`

		HealthyGroupCollapseThreshold int  `json:"healthy_group_collapse_threshold"`
		PersistCollapse               bool `json:"persist_collapse"`
		HideStatCards                 bool `json:"hide_stat_cards"`
		EmbeddedDatastarSDK           bool `json:"embedded_datastar_sdk"`
		PushOnChangeTTL               int  `json:"push_on_change_ttl"`
	} `json:"modes"`
}

func decodeIntrospection(t *testing.T, body []byte) introspectionDoc {
	t.Helper()

	var doc introspectionDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("introspection is not valid JSON: %v\n%s", err, body)
	}

	return doc
}

// TestIntrospection_ServesResolvedConfig verifies the endpoint reports the
// dashboard's own version, the resolved routes, and the configured modes.
func TestIntrospection_ServesResolvedConfig(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t,
		dashboard.WithTrend(42),
		dashboard.WithMetrics(true),
		dashboard.WithRateLimit(10, 1<<20),
		dashboard.WithIntrospection(),
		dashboard.WithHealthyGroupCollapse(3),
		dashboard.WithPersistCollapse(),
		dashboard.WithPushOnChangeTTL(2),
		dashboard.WithTimelineMaxAge(time.Hour),
		dashboard.WithEmbeddedDatastarSDK(),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/introspect")
	if w.Code != http.StatusOK {
		t.Fatalf("introspection: want 200, got %d: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}

	doc := decodeIntrospection(t, w.Body.Bytes())

	if doc.Version != dashboard.Version {
		t.Errorf("version: want %q, got %q", dashboard.Version, doc.Version)
	}

	for _, route := range []string{"dashboard", "sse", "trend", "export", "metrics", "datastar_js", "introspect"} {
		if doc.Routes[route] == "" {
			t.Errorf("routes.%s missing for an enabled feature", route)
		}
	}

	switch {
	case doc.Modes.TrendSamples != 42:
		t.Errorf("modes.trend_samples: want 42, got %d", doc.Modes.TrendSamples)
	case doc.Modes.PushMode != "on-change":
		t.Errorf("modes.push_mode: want on-change, got %q", doc.Modes.PushMode)
	case !doc.Modes.Metrics:
		t.Error("modes.metrics: want true after WithMetrics(true)")
	case !doc.Limits.RateLimitEnabled:
		t.Error("limits.rate_limit_enabled: want true after WithRateLimit")
	case doc.Modes.NonceStrategy != "none":
		t.Errorf("modes.nonce_strategy: want none, got %q", doc.Modes.NonceStrategy)
	case doc.Modes.HealthyGroupCollapseThreshold != 3:
		t.Errorf("modes.healthy_group_collapse_threshold: want 3, got %d", doc.Modes.HealthyGroupCollapseThreshold)
	case !doc.Modes.PersistCollapse:
		t.Error("modes.persist_collapse: want true after WithPersistCollapse")
	case doc.Modes.PushOnChangeTTL != 2:
		t.Errorf("modes.push_on_change_ttl: want 2, got %d", doc.Modes.PushOnChangeTTL)
	case doc.Limits.TimelineMaxAge != "1h0m0s":
		t.Errorf("limits.timeline_max_age: want 1h0m0s, got %q", doc.Limits.TimelineMaxAge)
	case !doc.Modes.EmbeddedDatastarSDK:
		t.Error("modes.embedded_datastar_sdk: want true after WithEmbeddedDatastarSDK")
	case doc.Modes.HideStatCards:
		t.Error("modes.hide_stat_cards: want false by default")
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

	for _, leak := range []string{"cache", "queue", `"error"`, `"fail"`} {
		if strings.Contains(w.Body.String(), leak) {
			t.Errorf(
				"introspection body contains check-derived data %q:\n%s",
				leak,
				w.Body.String(),
			)
		}
	}
}
