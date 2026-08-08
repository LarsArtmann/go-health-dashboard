package dashboard

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	health "github.com/larsartmann/go-health"
)

const (
	// defaultTitle is used when WithTitle is not provided.
	defaultTitle = "Health Dashboard"
	// defaultPushInterval is used when neither the probe's RefreshInterval
	// nor WithPushInterval is set (e.g. live mode with zero interval).
	defaultPushInterval = 2 * time.Second
)

// Version is the current package version.
const Version = "0.1.0"

// Config holds construction-only configuration for a Dashboard.
// It is populated by Option functions and consumed by New.
type Config struct {
	Title        string
	PushInterval time.Duration
	PushMode     PushMode
	Routes       Routes
	Nonce        string
}

// Option configures a Dashboard. Use the With* functions to create options.
type Option func(*Config)

// WithTitle sets the page title and heading displayed in the dashboard.
func WithTitle(title string) Option {
	return func(c *Config) { c.Title = title }
}

// WithPushInterval sets the SSE push cadence. When zero (the default),
// the dashboard uses the probe's configured RefreshInterval, falling back
// to 2s when the probe is in live mode (interval == 0).
func WithPushInterval(d time.Duration) Option {
	return func(c *Config) { c.PushInterval = d }
}

// WithPushMode selects when the pusher sends updates: only on change
// (default) or on every tick.
func WithPushMode(mode PushMode) Option {
	return func(c *Config) { c.PushMode = mode }
}

// WithNonce sets the CSP nonce used in script and style tags. Required when
// the host application uses a strict Content-Security-Policy.
func WithNonce(nonce string) Option {
	return func(c *Config) { c.Nonce = nonce }
}

// WithRoutes overrides the default URL paths for dashboard and probe endpoints.
func WithRoutes(routes Routes) Option {
	return func(c *Config) { c.Routes = routes }
}

// Dashboard renders a browser-friendly health dashboard from a go-health
// Probe using Datastar SSE for real-time updates.
//
// The dashboard lives at a dedicated HTML route (default /health). It serves
// HTML by default but returns JSON when the client sends Accept:
// application/json. Kubernetes probe endpoints (/healthz, /readyz,
// /startupz) are wired separately as JSON-only.
//
// Dashboard is safe for concurrent use by multiple goroutines.
type Dashboard struct {
	probe *health.Probe
	cfg   Config
	push  *pusher
}

// New creates a Dashboard wired to the given Probe. The Probe provides
// health data (via CachedResponse) and JSON handlers (via ReadinessHandler,
// LivenessHandler, StartupHandler).
//
// Default configuration:
//   - Title: "Health Dashboard"
//   - PushInterval: probe's RefreshInterval, or 2s if probe is live
//   - PushMode: PushOnChange
//   - Routes: DefaultRoutes()
func New(probe *health.Probe, opts ...Option) *Dashboard {
	cfg := Config{
		Title:    defaultTitle,
		PushMode: PushOnChange,
		Routes:   DefaultRoutes(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.PushInterval = resolvePushInterval(cfg.PushInterval, probe)

	return &Dashboard{probe: probe, cfg: cfg}
}

// resolvePushInterval determines the effective push interval. When the
// caller provides a positive interval via WithPushInterval, it wins.
// Otherwise we fall back to the probe's configured interval, or the default
// when the probe is in live mode (interval == 0).
func resolvePushInterval(configured time.Duration, probe *health.Probe) time.Duration {
	if configured > 0 {
		return configured
	}

	if probeInterval := probe.RefreshInterval(); probeInterval > 0 {
		return probeInterval
	}

	return defaultPushInterval
}

// currentResponse returns the probe's cached health snapshot.
func (d *Dashboard) currentResponse() health.Response {
	return d.probe.CachedResponse()
}

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

		data := d.buildData()

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

	for _, part := range strings.Split(accept, ",") {
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

		switch {
		case mediaType == "application/json":
			jsonQ = max(jsonQ, q)
		case mediaType == "text/html":
			htmlQ = max(htmlQ, q)
		case mediaType == "application/*":
			jsonQ = max(jsonQ, q)
		case mediaType == "text/*":
			htmlQ = max(htmlQ, q)
		case mediaType == "*/*":
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

// buildData constructs the viewModel from the probe's cached response.
func (d *Dashboard) buildData() viewModel {
	resp := d.currentResponse()
	vm := buildViewModel(resp, d.cfg.Title, d.cfg.Routes.SSE)
	vm.DatastarNonce = d.cfg.Nonce
	vm.TailwindNonce = d.cfg.Nonce
	return vm
}

// RegisterRoutes registers all dashboard and probe endpoints on the given
// mux using the provided routes. Pass DefaultRoutes for conventional paths.
//
// This wires up:
//   - Dashboard route (HTML page with Datastar SSE)
//   - SSE route (Datastar patch stream)
//   - Liveness, Readiness, Startup probe endpoints (JSON)
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux, routes Routes) {
	mux.HandleFunc(routes.Dashboard, d.Handler())
	mux.HandleFunc(routes.SSE, d.SSEHandler())
	mux.HandleFunc(routes.Liveness, d.probe.LivenessHandler())
	mux.HandleFunc(routes.Readiness, d.probe.ReadinessHandler())
	mux.HandleFunc(routes.Startup, d.probe.StartupHandler())
}

// Start launches the SSE pusher goroutine that broadcasts health updates to
// connected clients. Call before serving HTTP traffic.
//
// The ctx controls the lifetime of the pusher goroutine. Call Shutdown to
// stop it cleanly.
func (d *Dashboard) Start(ctx context.Context) error {
	d.push = newPusher(d)
	go d.push.start(ctx)

	return nil
}

// Shutdown stops the SSE pusher and closes all broadcaster connections.
// Safe to call multiple times.
func (d *Dashboard) Shutdown() {
	if d.push != nil {
		d.push.broadcaster.Close()
		d.push = nil
	}
}
