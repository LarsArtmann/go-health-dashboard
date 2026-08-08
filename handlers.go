package dashboard

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// contentNegotiationHandler inspects the Accept header and dispatches:
//   - text/html (or */* from browsers) → render the full dashboard page
//   - application/json (or no Accept header) → delegate to probe readiness
//
// This follows HTTP content negotiation semantics. The dashboard lives at
// the same path as the readiness endpoint; the Accept header determines
// the representation.
func (d *Dashboard) contentNegotiationHandler(w http.ResponseWriter, r *http.Request) {
	if acceptsHTML(r) {
		d.renderHTML(w, r)
		return
	}

	d.probe.ReadinessHandler()(w, r)
}

// partialHandler renders just the dashboard content (alert + table) without
// the full HTML document. This endpoint is called by HTMX polling on each
// refresh cycle.
func (d *Dashboard) partialHandler(w http.ResponseWriter, r *http.Request) {
	data := d.buildData(r)
	component := Partial(data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "dashboard: failed to render partial", http.StatusInternalServerError)
		return
	}
}

// renderHTML renders the full dashboard page as HTML.
func (d *Dashboard) renderHTML(w http.ResponseWriter, r *http.Request) {
	data := d.buildData(r)
	component := View(data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "dashboard: failed to render page", http.StatusInternalServerError)
		return
	}
}

// buildData constructs the viewModel from the probe's cached response.
func (d *Dashboard) buildData(r *http.Request) viewModel {
	resp := d.currentResponse()

	every := d.pollIntervalString()
	if d.cfg.RefreshMode == RefreshModeOff {
		every = ""
	}

	partialURL := d.cfg.Routes.Partial
	if strings.HasPrefix(partialURL, "/") {
		// Keep as-is — HTMX resolves relative to the page origin.
	}

	return buildViewModel(resp, d.cfg.Title, partialURL, every)
}

// acceptsHTML reports whether the request prefers HTML over JSON based on
// the Accept header. Browsers send "Accept: text/html, */* ..." or just "*/*".
// When no Accept header is present, we default to JSON (kubelet doesn't send
// Accept headers).
func acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	// Fast path: check for explicit text/html or */*
	if strings.Contains(accept, "text/html") {
		return true
	}

	// Browsers often send "*/*" as their first (or only) Accept value.
	// Treat */* as HTML so the dashboard renders when visited in a browser.
	if strings.Contains(accept, "*/*") {
		return true
	}

	return false
}

// renderComponent is a helper that renders a templ.Component to the
// ResponseWriter with the given content type.
func renderComponent(w http.ResponseWriter, r *http.Request, contentType string, component templ.Component) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "dashboard: failed to render", http.StatusInternalServerError)
	}
}
