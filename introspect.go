package dashboard

import (
	"encoding/json/v2"
	"net/http"
	"runtime"
	"time"
)

// introspection is the JSON document served by IntrospectionHandler. It
// exposes the dashboard's resolved configuration — routes, limits, and
// modes — so operators can verify a running instance without reading its
// source or config. Durations are rendered as Go duration strings.
type introspection struct {
	Version   string            `json:"version"`
	GoVersion string            `json:"go_version"`
	Routes    map[string]string `json:"routes"`
	Limits    introspectLimits  `json:"limits"`
	Modes     introspectModes   `json:"modes"`
}

type introspectLimits struct {
	MaxSSEConnections     int    `json:"max_sse_connections"` // 0 = unlimited
	RateLimitEnabled      bool   `json:"rate_limit_enabled"`
	ShutdownDrain         string `json:"shutdown_drain"`
	MaxConnectionLifetime string `json:"max_connection_lifetime"`
	HeartbeatInterval     string `json:"heartbeat_interval"`
	TimelineMaxAge        string `json:"timeline_max_age"` // "0s" = keep all entries
}

type introspectModes struct {
	PushMode      string `json:"push_mode"`
	PublicMode    bool   `json:"public_mode"`
	Metrics       bool   `json:"metrics"`
	Webhook       bool   `json:"webhook"`
	TrendSamples  int    `json:"trend_samples"` // 0 = trend disabled
	NonceStrategy string `json:"nonce_strategy"`

	// Presentation modes introduced with the 0.7.x UI.
	HealthyGroupCollapseThreshold int  `json:"healthy_group_collapse_threshold"` // 0 = always expanded
	PersistCollapse               bool `json:"persist_collapse"`
	HideStatCards                 bool `json:"hide_stat_cards"`
	EmbeddedDatastarSDK           bool `json:"embedded_datastar_sdk"`
	PushOnChangeTTL               int  `json:"push_on_change_ttl"` // ticks between re-asserts; 0 = disabled
}

// IntrospectionHandler serves the resolved configuration as JSON. Enabled
// together with Routes.Introspect via WithIntrospection. Introspection is
// configuration metadata only — it never includes check results, check
// names, or probe payloads — but it does disclose route paths, so it passes
// through the same middleware and rate limiting as the other
// dashboard-owned routes.
func (d *Dashboard) IntrospectionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		routes := map[string]string{}

		for name, path := range map[string]string{
			"dashboard":   d.cfg.Routes.Dashboard,
			"sse":         d.cfg.Routes.SSE,
			"favicon":     d.cfg.Routes.Favicon,
			"metrics":     d.cfg.Routes.Metrics,
			"trend":       d.cfg.Routes.Trend,
			"export":      d.cfg.Routes.Export,
			"datastar_js": d.cfg.Routes.DatastarJS,
			"introspect":  d.cfg.Routes.Introspect,
		} {
			if path != "" {
				routes[name] = path
			}
		}

		strategy := "none"

		switch {
		case d.cfg.NonceExtractor != nil:
			strategy = "per-request"
		case d.cfg.Nonce != "":
			strategy = "static"
		}

		doc := introspection{
			Version:   Version,
			GoVersion: runtime.Version(),
			Routes:    routes,
			Limits: introspectLimits{
				MaxSSEConnections:     d.cfg.MaxSSEConnections,
				RateLimitEnabled:      d.cfg.RateLimitRequests > 0 && d.cfg.RateLimitWindow > 0,
				ShutdownDrain:         formatDuration(d.cfg.ShutdownDrain),
				MaxConnectionLifetime: formatDuration(d.cfg.MaxConnectionLifetime),
				HeartbeatInterval:     formatDuration(d.cfg.HeartbeatInterval),
				TimelineMaxAge:        formatDuration(d.cfg.TimelineMaxAge),
			},
			Modes: introspectModes{
				PushMode:      string(d.cfg.PushMode),
				PublicMode:    d.cfg.PublicMode,
				Metrics:       d.cfg.MetricsEnabled,
				Webhook:       d.cfg.WebhookURL != "",
				TrendSamples:  d.cfg.TrendSamples,
				NonceStrategy: strategy,

				HealthyGroupCollapseThreshold: d.cfg.HealthyGroupCollapseThreshold,
				PersistCollapse:               d.cfg.PersistCollapse,
				HideStatCards:                 d.cfg.HideStatCards,
				EmbeddedDatastarSDK:           d.cfg.EmbeddedDatastarSDK,
				PushOnChangeTTL:               d.cfg.PushOnChangeTTL,
			},
		}

		if err := json.MarshalWrite(w, doc, json.Deterministic(true)); err != nil {
			http.Error(
				w,
				"dashboard: failed to encode introspection",
				http.StatusInternalServerError,
			)
		}
	})
}

// formatDuration renders a duration for the introspection document. Zero
// durations render as "0s" rather than "0" so the field stays unambiguous.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	return d.String()
}
