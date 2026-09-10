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
	Limits    struct {
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

	for _, check := range []struct {
		name string
		ok   bool
		msg  string
	}{
		{"trend_samples", doc.Modes.TrendSamples == 42, "want 42 after WithTrend(42)"},
		{"push_mode", doc.Modes.PushMode == "on-change", "want on-change"},
		{"metrics", doc.Modes.Metrics, "want true after WithMetrics(true)"},
		{"rate_limit_enabled", doc.Limits.RateLimitEnabled, "want true after WithRateLimit"},
		{"nonce_strategy", doc.Modes.NonceStrategy == "none", "want none"},
		{
			"healthy_group_collapse_threshold",
			doc.Modes.HealthyGroupCollapseThreshold == 3,
			"want 3 after WithHealthyGroupCollapse(3)",
		},
		{"persist_collapse", doc.Modes.PersistCollapse, "want true after WithPersistCollapse"},
		{"push_on_change_ttl", doc.Modes.PushOnChangeTTL == 2, "want 2 after WithPushOnChangeTTL(2)"},
		{"timeline_max_age", doc.Limits.TimelineMaxAge == "1h0m0s", "want 1h0m0s after WithTimelineMaxAge(time.Hour)"},
		{"embedded_datastar_sdk", doc.Modes.EmbeddedDatastarSDK, "want true after WithEmbeddedDatastarSDK"},
		{"hide_stat_cards", !doc.Modes.HideStatCards, "want false by default"},
	} {
		if !check.ok {
			t.Errorf("%s: %s", check.name, check.msg)
		}
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
