package dashboard

import (
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"

	dstarstatic "github.com/larsartmann/go-datastar/static"
	health "github.com/larsartmann/go-health"
)

// Handler returns an http.HandlerFunc that serves the health dashboard with
// content negotiation based on the Accept header:
//
//   - Accept: application/json → returns the probe's cached health response as
//     JSON. HTTP status is 503 when any check is failing, 200 otherwise.
//   - Any other Accept value (or none) → renders the full HTML dashboard page.
//
// Register it at your dashboard route (e.g. /health).
func (d *Dashboard) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if wantsJSON(r) {
			d.serveJSON(w)

			return
		}

		data := d.buildData(r)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")

		if err := View(data).Render(r.Context(), w); err != nil {
			http.Error(w, "dashboard: failed to render page", http.StatusInternalServerError)

			return
		}
	}
}

// wantsJSON reports whether the request prefers JSON over HTML based on
// the Accept header's quality values (RFC 7231 §5.3.2). Returns false when
// the header is empty, absent, or HTML is preferred. When both types have
// equal q-values, HTML wins (the dashboard default).
func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	var jsonQ, htmlQ, anyQ float64

	for part := range strings.SplitSeq(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.Split(part, ";")
		mediaType := strings.TrimSpace(strings.ToLower(segments[0]))
		q := 1.0

		for _, seg := range segments[1:] {
			seg = strings.TrimSpace(seg)
			if strings.HasPrefix(strings.ToLower(seg), "q=") {
				if v, err := strconv.ParseFloat(seg[2:], 64); err == nil {
					q = v
				}
			}
		}

		switch mediaType {
		case "application/json":
			jsonQ = max(jsonQ, q)
		case "text/html":
			htmlQ = max(htmlQ, q)
		case "application/*":
			jsonQ = max(jsonQ, q)
		case "text/*":
			htmlQ = max(htmlQ, q)
		case "*/*":
			anyQ = max(anyQ, q)
		}
	}

	jsonQ = max(jsonQ, anyQ)
	htmlQ = max(htmlQ, anyQ)

	return jsonQ > htmlQ
}

// serveJSON writes the probe's cached health response as JSON. The HTTP
// status code is 503 when the overall status is fail, 200 otherwise.
func (d *Dashboard) serveJSON(w http.ResponseWriter) {
	resp := d.currentResponse()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	code := http.StatusOK
	if resp.Status == health.StatusFail {
		code = http.StatusServiceUnavailable
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "dashboard: failed to encode health response", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(code)
	_, _ = w.Write(payload)
}

// SSEHandler returns an http.HandlerFunc that upgrades to an SSE connection
// and streams Datastar patches to the browser.
func (d *Dashboard) SSEHandler() http.HandlerFunc {
	return d.sseHandler
}

// SubscriberCount returns the number of active SSE connections. Returns 0
// when the pusher has not been started.
func (d *Dashboard) SubscriberCount() int64 {
	if p := d.push.Load(); p != nil {
		return p.connections.Load()
	}

	return 0
}

// buildData constructs the viewModel from the probe's cached response.
// When a NonceExtractor is configured, the nonce is read per-request;
// otherwise the fixed construction-time Nonce is used.
func (d *Dashboard) buildData(r *http.Request) viewModel {
	resp := d.currentResponse()
	vm := buildViewModel(resp, d.cfg.Title, d.cfg.Routes.SSE)

	nonce := d.cfg.Nonce
	if d.cfg.NonceExtractor != nil {
		if extracted := d.cfg.NonceExtractor(r); extracted != "" {
			nonce = extracted
		}
	}

	vm.DatastarNonce = nonce
	vm.TailwindNonce = nonce
	vm.CSSPath = d.cfg.CSSPath
	vm.DatastarSrc = d.cfg.DatastarSrc
	vm.FaviconURL = d.cfg.Routes.Favicon
	vm.ShowStatCards = !d.cfg.HideStatCards
	vm.Description = d.cfg.Description

	if d.cfg.PublicMode {
		anonymizeViewModel(&vm)
	}

	if p := d.push.Load(); p != nil && p.history != nil {
		populateHistory(&vm, p.history)
	}

	return vm
}

// RegisterRoutes registers all dashboard and probe endpoints on the given
// mux using the dashboard's configured routes (set via WithRoutes or
// WithBasePath, defaulting to DefaultRoutes).
//
// This wires up:
//   - Dashboard route (HTML page with Datastar SSE)
//   - SSE route (Datastar patch stream)
//   - Favicon route (SVG favicon)
//   - Metrics route (Prometheus exposition, when enabled via WithMetrics)
//   - Liveness, Readiness, Startup probe endpoints (JSON)
//
// Dashboard-owned routes (dashboard, SSE, favicon, metrics) pass through the
// middleware configured via WithMiddleware; the Kubernetes probe endpoints
// never do, so kubelet probes keep working without credentials.
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	routes := d.cfg.Routes

	mux.Handle(routes.Dashboard, d.wrap(d.applyRateLimit(d.Handler())))
	mux.Handle(routes.SSE, d.wrap(d.applyRateLimit(d.SSEHandler())))

	if routes.Favicon != "" {
		mux.Handle(routes.Favicon, d.wrap(d.applyRateLimit(d.FaviconHandler())))
	}

	if d.cfg.MetricsEnabled && routes.Metrics != "" {
		mux.Handle(routes.Metrics, d.wrap(d.applyRateLimit(d.MetricsHandler())))
	}

	if d.cfg.TrendSamples > 0 && routes.Trend != "" {
		mux.Handle(routes.Trend, d.wrap(d.applyRateLimit(d.TrendHandler())))
	}

	if d.cfg.TrendSamples > 0 && routes.Export != "" {
		mux.Handle(routes.Export, d.wrap(d.applyRateLimit(d.ExportHandler())))
	}

	if d.cfg.Introspection && routes.Introspect != "" {
		mux.Handle(routes.Introspect, d.wrap(d.applyRateLimit(d.IntrospectionHandler())))
	}

	if d.cfg.EmbeddedDatastarSDK && routes.DatastarJS != "" {
		mux.Handle(routes.DatastarJS, d.wrap(d.embeddedSDKHandler()))
	}

	mux.HandleFunc(routes.Liveness, d.probe.LivenessHandler())
	mux.HandleFunc(routes.Readiness, d.probe.ReadinessHandler())
	mux.HandleFunc(routes.Startup, d.probe.StartupHandler())
}

// wrap applies the configured middleware (WithMiddleware) to a
// dashboard-owned handler. Probe endpoints bypass it so kubelet probes
// keep working without credentials.
func (d *Dashboard) wrap(h http.Handler) http.Handler {
	if d.cfg.Middleware == nil {
		return h
	}

	return d.cfg.Middleware(h)
}

// embeddedSDKHandler serves the pinned Datastar SDK bundle from the
// go-datastar/static embed (WithEmbeddedDatastarSDK). Same-origin, so a
// strict CSP needs no CDN exception beyond the SDK's own 'unsafe-eval'.
func (d *Dashboard) embeddedSDKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != strings.TrimSuffix(d.cfg.Routes.DatastarJS, "/") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(dstarstatic.Bytes())
	})
}
