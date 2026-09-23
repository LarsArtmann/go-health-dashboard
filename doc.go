// Package dashboard renders a real-time, browser-friendly health dashboard
// from a [github.com/larsartmann/go-health] Probe. It composes go-health
// (health checking), [github.com/larsartmann/templ-components] (UI rendering),
// and [github.com/larsartmann/go-datastar] (SSE patch protocol) into a single
// drop-in handler.
//
// The dashboard lives at /health and uses Datastar SSE for real-time updates.
// It serves HTML by default but returns JSON when the client sends
// Accept: application/json. Kubernetes probe endpoints (/healthz, /readyz,
// /startupz) are wired separately as JSON-only. When check sources report
// timing (go-health v0.2.0 metadata), service-table rows render each check's
// state-entry time and execution duration, and a failure-evidence strip
// separates proven greens from unproven ones.
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
// # Common Option Combinations
//
// Public deployments that push status transitions to an operations
// webhook typically also anonymize the page: WithWebhook announces every
// status/fingerprint transition (independent of the SSE push mode), and
// WithPublicMode masks check names and error details in the HTML and
// metrics labels — the two compose without extra wiring:
//
//	dash := dashboard.New(probe,
//	    dashboard.WithWebhook("https://ops.example.invalid/hook"),
//	    dashboard.WithPublicMode(),
//	)
//
// Mounting under a sub-path (behind a reverse proxy or an admin mux) is
// one option: WithBasePath prefixes every route in Config.Routes, and the
// HTML-referenced SSE URL follows the registered handler automatically:
//
//	dash := dashboard.New(probe, dashboard.WithBasePath("/admin"))
//	dash.RegisterRoutes(mux) // serves /admin/health, /admin/health/sse, ...
//
// # Health Errors
//
// HealthCheck reports two sentinel-wrapped errors, both detectable via
// errors.Is(err, dashboard.ErrPusherNotActive):
//
//	dashboard.ErrPusherNotStarted // Start has never been called
//	dashboard.ErrPusherShutDown   // Shutdown has been called
package dashboard
