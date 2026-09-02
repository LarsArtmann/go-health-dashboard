package dashboard

// Routes configures the URL paths for Dashboard.RegisterRoutes.
type Routes struct {
	Dashboard string // HTML dashboard page (default: /health)
	SSE       string // SSE push endpoint for real-time updates (default: /health/sse)
	Favicon   string // SVG favicon endpoint (default: /favicon.svg)
	Liveness  string // Kubernetes liveness probe — JSON (default: /healthz)
	Readiness string // Kubernetes readiness probe — JSON (default: /readyz)
	Startup   string // Kubernetes startup probe — JSON (default: /startupz)
	Metrics   string // Prometheus exposition endpoint (default: /health/metrics; leave empty to disable)
}

// DefaultRoutes returns conventional paths for the dashboard and Kubernetes
// health probes. The HTML dashboard lives at /health; kubelet endpoints use
// the standard /healthz, /readyz, /startupz paths.
func DefaultRoutes() Routes {
	return Routes{
		Dashboard: "/health",
		SSE:       "/health/sse",
		Favicon:   "/favicon.svg",
		Liveness:  "/healthz",
		Readiness: "/readyz",
		Startup:   "/startupz",
		Metrics:   "/health/metrics",
	}
}
