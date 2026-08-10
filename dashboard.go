package dashboard

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

const (
	// defaultTitle is used when WithTitle is not provided.
	defaultTitle = "Health Dashboard"
	// defaultPushInterval is used when neither the probe's RefreshInterval
	// nor WithPushInterval is set (e.g. live mode with zero interval).
	defaultPushInterval = 2 * time.Second
	// defaultHeartbeatInterval is the SSE keepalive interval when
	// WithHeartbeatInterval is not set.
	defaultHeartbeatInterval = 15 * time.Second
)

// Version is the current package version.
const Version = "0.2.0"

// Config holds construction-only configuration for a Dashboard.
// It is populated by Option functions and consumed by New.
type Config struct {
	Title             string
	PushInterval      time.Duration
	PushMode          PushMode
	Routes            Routes
	Nonce             string
	NonceExtractor    func(*http.Request) string
	CSSPath           string
	DatastarSrc       string
	HeartbeatInterval time.Duration
	MaxSSEConnections int
	RetryInterval     time.Duration
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

// WithNonce sets a fixed CSP nonce used in script and style tags. Required
// when the host application uses a strict Content-Security-Policy but cannot
// provide per-request nonces (e.g. because the dashboard is constructed once
// at startup). For stronger security, prefer WithNonceExtractor.
func WithNonce(nonce string) Option {
	return func(c *Config) { c.Nonce = nonce }
}

// WithNonceExtractor provides a function that extracts the CSP nonce from
// each incoming request. This enables per-request nonces (more secure than
// a fixed construction-time nonce) when the host application uses middleware
// such as httputil.Nonce that stores a unique nonce in the request context.
//
// When set, the extractor takes precedence over WithNonce. If the extractor
// returns an empty string for a given request, the dashboard falls back to
// the fixed Nonce from WithNonce.
//
// Example wiring with httputil:
//
//	dashboard.New(probe, dashboard.WithNonceExtractor(httputil.NonceFromRequest))
func WithNonceExtractor(fn func(*http.Request) string) Option {
	return func(c *Config) { c.NonceExtractor = fn }
}

// WithRoutes overrides the default URL paths for dashboard and probe endpoints.
func WithRoutes(routes Routes) Option {
	return func(c *Config) { c.Routes = routes }
}

// WithCSSPath sets the URL path to a compiled CSS stylesheet. When set, the
// dashboard uses a <link> tag instead of the Tailwind Play CDN <script> tag.
// Use this in production to avoid the runtime overhead of the CDN.
func WithCSSPath(path string) Option {
	return func(c *Config) { c.CSSPath = path }
}

// WithDatastarSrc sets a self-hosted URL for the Datastar SDK script. When
// set, the dashboard renders <script src=...> pointing at this URL instead
// of the default jsdelivr CDN. Use this when the host application's
// Content-Security-Policy only allows 'self' scripts (e.g. the HTTP server
// serves a local copy of datastar.js).
func WithDatastarSrc(src string) Option {
	return func(c *Config) { c.DatastarSrc = src }
}

// WithHeartbeatInterval sets how often the SSE handler sends a comment-line
// keepalive to prevent proxy/load-balancer timeout. When zero (the default),
// the dashboard uses 15s.
func WithHeartbeatInterval(d time.Duration) Option {
	return func(c *Config) { c.HeartbeatInterval = d }
}

// WithMaxSSEConnections limits the number of concurrent SSE clients. When
// zero (the default), the number of connections is unlimited. Use this to
// prevent DoS via connection exhaustion.
func WithMaxSSEConnections(n int) Option {
	return func(c *Config) { c.MaxSSEConnections = n }
}

// WithRetryInterval sets the SSE retry field (in milliseconds) that tells
// the browser how long to wait before reconnecting after a disconnect. When
// zero (the default), the browser's built-in default (~3s) is used.
//
// A shorter interval means faster recovery from transient network blips;
// a longer interval reduces server load when many clients reconnect at once.
// Negative values are treated as zero.
func WithRetryInterval(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.RetryInterval = d
		}
	}
}

// WithBasePath prefixes all dashboard and probe routes with the given path.
// Use this when mounting the dashboard under a non-root path — for example
// WithBasePath("/admin") produces "/admin/health", "/admin/health/sse", etc.
//
// The prefix is applied to whatever routes are currently configured. When
// combined with WithRoutes, call WithBasePath last so it prefixes the custom
// routes; calling WithRoutes after WithBasePath replaces the prefixed set.
func WithBasePath(prefix string) Option {
	return func(cfg *Config) {
		prefix = strings.TrimSuffix(prefix, "/")
		if prefix == "" {
			return
		}

		r := cfg.Routes
		cfg.Routes = Routes{
			Dashboard: prefix + r.Dashboard,
			SSE:       prefix + r.SSE,
			Favicon:   prefix + r.Favicon,
			Liveness:  prefix + r.Liveness,
			Readiness: prefix + r.Readiness,
			Startup:   prefix + r.Startup,
		}
	}
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
	push  atomic.Pointer[pusher]
}

// Compile-time assertions that Dashboard satisfies samber/do lifecycle interfaces.
// These enable automatic participation in do.HealthCheck and do.Shutdown cascades
// when the Dashboard is registered in a DI container via Register.
var (
	_ do.HealthcheckerWithContext = (*Dashboard)(nil)
	_ do.Shutdowner               = (*Dashboard)(nil)
)

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
		Title:             defaultTitle,
		PushMode:          PushOnChange,
		Routes:            DefaultRoutes(),
		HeartbeatInterval: defaultHeartbeatInterval,
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
//   - Liveness, Readiness, Startup probe endpoints (JSON)
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	routes := d.cfg.Routes

	mux.HandleFunc(routes.Dashboard, d.Handler())
	mux.HandleFunc(routes.SSE, d.SSEHandler())

	if routes.Favicon != "" {
		mux.HandleFunc(routes.Favicon, d.FaviconHandler())
	}

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
	p := newPusher(d)

	d.push.Store(p)
	go p.start(ctx)

	return nil
}

// Shutdown stops the SSE pusher and closes all broadcaster connections.
// Safe to call multiple times.
func (d *Dashboard) Shutdown() {
	if p := d.push.Swap(nil); p != nil {
		p.broadcaster.Close()
	}
}

// ErrPusherNotActive is returned by HealthCheck when the SSE pusher has not
// been started or has been shut down.
var ErrPusherNotActive = errors.New("dashboard: SSE pusher is not active")

// HealthCheck reports whether the dashboard's real-time update mechanism is
// healthy. Returns an error when the SSE pusher has not been started or has
// been shut down.
//
// This method satisfies do.HealthcheckerWithContext, enabling the dashboard
// to participate in container-wide health checks when registered in a
// samber/do injector.
func (d *Dashboard) HealthCheck(_ context.Context) error {
	if d.push.Load() == nil {
		return ErrPusherNotActive
	}

	return nil
}
