// Package dashboard renders a real-time, browser-friendly health dashboard
// from a [github.com/larsartmann/go-health] Probe. It composes go-health
// (health checking), [github.com/larsartmann/templ-components] (UI rendering),
// and [github.com/larsartmann/go-datastar] (SSE patch protocol) into a single
// drop-in handler.
//
// The dashboard lives at /health and uses Datastar SSE for real-time updates.
// It serves HTML by default but returns JSON when the client sends
// Accept: application/json. Kubernetes probe endpoints (/healthz, /readyz,
// /startupz) are wired separately as JSON-only.
//
// # Quick Start
//
//	probe := health.New(injector, health.WithVersion("1.2.3"))
//	_ = probe.Start(ctx)
//
//	dash := dashboard.New(probe,
//	    dashboard.WithTitle("My Service"),
//	)
//	_ = dash.Start(ctx)
//	defer dash.Shutdown()
//
//	mux := http.NewServeMux()
//	dash.RegisterRoutes(mux)
//	http.ListenAndServe(":8080", mux)
//
// Browser visits http://localhost:8080/health and sees a live status dashboard
// that updates in real-time via SSE. Kubelet hits http://localhost:8080/readyz
// and gets the JSON readiness response.
//
// # DI Container Integration
//
// When the probe already runs in a samber/do injector (go-health requires
// one), Register wires the dashboard into the same container. The dashboard
// then participates in container-wide Shutdown and HealthCheck cascades —
// no manual Start/Shutdown bookkeeping beyond starting the pusher:
//
//	dash := dashboard.Register(injector, probe)
//	_ = dash.Start(ctx)
//	defer do.Shutdown(injector) // cascades to the dashboard
//
// # Health Errors
//
// HealthCheck reports two sentinel-wrapped errors, both detectable via
// errors.Is(err, dashboard.ErrPusherNotActive):
//
//	dashboard.ErrPusherNotStarted // Start has never been called
//	dashboard.ErrPusherShutDown   // Shutdown has been called
package dashboard
