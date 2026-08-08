// Package dashboard renders a browser-friendly health dashboard from a
// [github.com/larsartmann/go-health] Probe. It sits between go-health (health
// checking) and [github.com/larsartmann/templ-components] (UI rendering),
// performing HTTP content negotiation: JSON requests are delegated to
// go-health's existing handlers; HTML requests get a rich browser dashboard
// with status banners, tables, and badges.
//
// # Quick Start
//
//	probe := health.New(injector, health.WithVersion("1.2.3"))
//	_ = probe.Start(ctx)
//
//	dash := dashboard.New(probe,
//	    dashboard.WithTitle("My Service"),
//	)
//
//	mux := http.NewServeMux()
//	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())
//	http.ListenAndServe(":8080", mux)
//
// Browser visits http://localhost:8080/health and sees a live status dashboard.
// Kubelet hits http://localhost:8080/readyz and gets the JSON readiness response.
package dashboard
