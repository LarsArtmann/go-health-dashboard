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
//	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())
//	http.ListenAndServe(":8080", mux)
//
// Browser visits http://localhost:8080/health and sees a live status dashboard
// that updates in real-time via SSE. Kubelet hits http://localhost:8080/readyz
// and gets the JSON readiness response.
package dashboard
