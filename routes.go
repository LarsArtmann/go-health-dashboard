package dashboard

// Routes configures the URL paths for Dashboard.RegisterRoutes.
type Routes struct {
	Dashboard string // Content-negotiated HTML/JSON dashboard (default: /health)
	Partial   string // HTMX polling partial endpoint (default: /health/partial)
	SSE       string // SSE push endpoint (default: /health/sse)
	Liveness  string // Probe liveness — JSON only (default: /healthz)
	Readiness string // Probe readiness — JSON only (default: /readyz)
	Startup   string // Probe startup — JSON only (default: /startupz)
}

// DefaultRoutes returns conventional paths for the dashboard and Kubernetes
// health probes. The dashboard lives at /health (content-negotiated), while
// kubelet endpoints use the standard /healthz, /readyz, /startupz paths.
func DefaultRoutes() Routes {
	return Routes{
		Dashboard: "/health",
		Partial:   "/health/partial",
		SSE:       "/health/sse",
		Liveness:  "/healthz",
		Readiness: "/readyz",
		Startup:   "/startupz",
	}
}
