package dashboard

import (
	health "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

// Register creates a Dashboard wired to the given Probe and registers it in
// the injector so it participates in container lifecycle cascades.
//
// After registration:
//
//   - do.Shutdown(injector) calls Dashboard.Shutdown(), closing the SSE
//     broadcaster and releasing the pusher.
//   - do.HealthCheck[*Dashboard](injector) calls Dashboard.HealthCheck(),
//     reporting whether the SSE pusher is active.
//
// The returned *Dashboard is the same instance stored in the container.
// Call Start before serving HTTP traffic and RegisterRoutes to wire routes.
//
// Example:
//
//	injector := do.New()
//	probe := health.New(injector, health.WithCriticalServices("db"))
//	dash := dashboard.Register(injector, probe, dashboard.WithTitle("API"))
//	dash.Start(ctx)
//	dash.RegisterRoutes(mux)
//	// On shutdown: do.Shutdown(injector) cascades to dash.Shutdown().
func Register(injector do.Injector, probe *health.Probe, opts ...Option) *Dashboard {
	dash := New(probe, opts...)
	do.ProvideValue(injector, dash)

	return dash
}
