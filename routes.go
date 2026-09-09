package dashboard

// Routes configures the URL paths for Dashboard.RegisterRoutes.
type Routes struct {
	Dashboard  string // HTML dashboard page (default: /health)
	SSE        string // SSE push endpoint for real-time updates (default: /health/sse)
	Favicon    string // SVG favicon endpoint (default: /favicon.svg)
	Liveness   string // Kubernetes liveness probe — JSON (default: /healthz)
	Readiness  string // Kubernetes readiness probe — JSON (default: /readyz)
	Startup    string // Kubernetes startup probe — JSON (default: /startupz)
	Metrics    string // Prometheus exposition endpoint (default: /health/metrics; leave empty to disable)
	Trend      string // Status history JSON endpoint (default: /health/trend; only registered with WithTrend)
	Export     string // Status history export endpoint, JSON/CSV (default: /health/export; only registered with WithTrend)
	Introspect string // Introspection endpoint, JSON config document (default: /health/introspect; only registered with WithIntrospection)
	DatastarJS string // Self-hosted Datastar SDK script served from the embedded bundle (default: /health/datastar.js; only registered with WithEmbeddedDatastarSDK)
}

// DefaultRoutes returns conventional paths for the dashboard and Kubernetes
// health probes. The HTML dashboard lives at /health; kubelet endpoints use
// the standard /healthz, /readyz, /startupz paths.
func DefaultRoutes() Routes {
	return Routes{
		Dashboard:  "/health",
		SSE:        "/health/sse",
		Favicon:    "/favicon.svg",
		Liveness:   "/healthz",
		Readiness:  "/readyz",
		Startup:    "/startupz",
		Metrics:    "/health/metrics",
		Trend:      "/health/trend",
		Export:     "/health/export",
		Introspect: "/health/introspect",
		DatastarJS: "/health/datastar.js",
	}
}

// exportURL returns the status-history export endpoint when the trend is
// enabled and the route is configured, otherwise empty (endpoint disabled).
func (d *Dashboard) exportURL() string {
	if d.cfg.TrendSamples > 0 && d.cfg.Routes.Export != "" {
		return d.cfg.Routes.Export
	}

	return ""
}

// trendURL returns the trend-history endpoint when the trend is enabled and
// the route is configured, otherwise empty (endpoint disabled).
func (d *Dashboard) trendURL() string {
	if d.cfg.TrendSamples > 0 && d.cfg.Routes.Trend != "" {
		return d.cfg.Routes.Trend
	}

	return ""
}

// metricsURL returns the Prometheus metrics endpoint when metrics are
// enabled and the route is configured, otherwise empty (endpoint disabled).
func (d *Dashboard) metricsURL() string {
	if d.cfg.MetricsEnabled && d.cfg.Routes.Metrics != "" {
		return d.cfg.Routes.Metrics
	}

	return ""
}
