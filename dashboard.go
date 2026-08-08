package dashboard

import (
	"net/http"
	"time"

	health "github.com/larsartmann/go-health"
)

const (
	// defaultTitle is used when WithTitle is not provided.
	defaultTitle = "Health Dashboard"
	// defaultPollInterval is used when neither the probe's RefreshInterval
	// nor WithRefreshInterval is set (e.g. live mode with zero interval).
	defaultPollInterval = 2 * time.Second
)

// RefreshMode controls how the dashboard auto-updates in the browser.
type RefreshMode string

const (
	// RefreshModePoll uses HTMX polling (default). The browser sends a GET
	// request to the partial endpoint on each interval. Polls read the
	// atomic cache pointer — zero dependency calls per poll.
	RefreshModePoll RefreshMode = "poll"

	// RefreshModeSSE uses Server-Sent Events for sub-second push updates.
	// The server pushes changes to all connected clients via a broadcaster.
	// Use this for NOC monitors where sub-second freshness matters.
	RefreshModeSSE RefreshMode = "sse"

	// RefreshModeOff disables auto-refresh. The dashboard renders once and
	// requires a full page reload to update.
	RefreshModeOff RefreshMode = "off"
)

// Config holds construction-only configuration for a Dashboard.
// It is populated by Option functions and consumed by New.
type Config struct {
	Title           string
	RefreshInterval time.Duration
	RefreshMode     RefreshMode
	Routes          Routes
	Nonce           string
}

// Option configures a Dashboard. Use the With* functions to create options.
type Option func(*Config)

// WithTitle sets the page title and heading displayed in the dashboard.
func WithTitle(title string) Option {
	return func(c *Config) { c.Title = title }
}

// WithRefreshInterval sets the browser auto-refresh cadence. When zero (the
// default), the dashboard uses the probe's configured RefreshInterval,
// falling back to 2s when the probe is in live mode (interval == 0).
func WithRefreshInterval(d time.Duration) Option {
	return func(c *Config) { c.RefreshInterval = d }
}

// WithRefreshMode selects the auto-refresh mechanism: HTMX polling (default),
// SSE push, or disabled.
func WithRefreshMode(mode RefreshMode) Option {
	return func(c *Config) { c.RefreshMode = mode }
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

// WithSSEPush enables SSE push mode for real-time updates. This is a
// convenience alias for WithRefreshMode(RefreshModeSSE).
func WithSSEPush() Option {
	return WithRefreshMode(RefreshModeSSE)
}

// Dashboard renders a browser-friendly health dashboard from a go-health
// Probe. It performs HTTP content negotiation: HTML requests get a rich
// dashboard with status banners, tables, and badges; JSON requests are
// delegated to go-health's existing readiness handler.
//
// Dashboard is safe for concurrent use by multiple goroutines.
type Dashboard struct {
	probe *health.Probe
	cfg   Config
	push  *ssePusher
}

// New creates a Dashboard wired to the given Probe. The Probe provides
// health data (via CachedResponse or Evaluate) and JSON handlers (via
// ReadinessHandler, LivenessHandler, StartupHandler).
//
// Default configuration:
//   - Title: "Health Dashboard"
//   - RefreshInterval: probe's RefreshInterval, or 2s if probe is live
//   - RefreshMode: HTMX polling
//   - Routes: DefaultRoutes()
func New(probe *health.Probe, opts ...Option) *Dashboard {
	cfg := Config{
		Title:       defaultTitle,
		RefreshMode: RefreshModePoll,
		Routes:      DefaultRoutes(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.RefreshInterval = resolvePollInterval(cfg.RefreshInterval, probe)

	return &Dashboard{probe: probe, cfg: cfg}
}

// resolvePollInterval determines the effective polling interval. When the
// caller provides a positive interval via WithRefreshInterval, it wins.
// Otherwise we fall back to the probe's configured interval, or the default
// when the probe is in live mode (interval == 0).
func resolvePollInterval(configured time.Duration, probe *health.Probe) time.Duration {
	if configured > 0 {
		return configured
	}

	if probeInterval := probe.RefreshInterval(); probeInterval > 0 {
		return probeInterval
	}

	return defaultPollInterval
}

// pollIntervalString returns the interval formatted for HTMX's
// hx-trigger="every Ns" attribute (e.g. "2s").
func (d *Dashboard) pollIntervalString() string {
	return d.cfg.RefreshInterval.String()
}

// currentResponse returns the best available health Response: the cached
// value when the background loop is active, or a live evaluation otherwise.
func (d *Dashboard) currentResponse() health.Response {
	return d.probe.CachedResponse()
}

// Handler returns an http.HandlerFunc that performs content negotiation:
// HTML requests render the full dashboard page; JSON requests delegate to
// the probe's readiness handler. This is the main entry point — register
// it at your health endpoint (e.g. /health).
func (d *Dashboard) Handler() http.HandlerFunc {
	return d.contentNegotiationHandler
}

// PartialHandler returns an http.HandlerFunc that renders just the dashboard
// content (alert + table), without the full HTML document. This endpoint is
// called by HTMX polling on each refresh cycle.
func (d *Dashboard) PartialHandler() http.HandlerFunc {
	return d.partialHandler
}

// SSEHandler returns an http.HandlerFunc that upgrades to an SSE connection
// and streams dashboard updates. Only meaningful when RefreshMode is SSE.
func (d *Dashboard) SSEHandler() http.HandlerFunc {
	return d.sseConnectionHandler
}

// RegisterRoutes registers all dashboard and probe endpoints on the given
// mux using the provided routes. Pass DefaultRoutes for conventional paths.
//
// This wires up:
//   - Dashboard route (content-negotiated HTML/JSON)
//   - Partial route (HTMX polling endpoint)
//   - SSE route (when SSE mode is active)
//   - Liveness, Readiness, Startup probe endpoints
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux, routes Routes) {
	mux.HandleFunc(routes.Dashboard, d.Handler())
	mux.HandleFunc(routes.Partial, d.PartialHandler())

	if d.cfg.RefreshMode == RefreshModeSSE {
		mux.HandleFunc(routes.SSE, d.SSEHandler())
	}

	mux.HandleFunc(routes.Liveness, d.probe.LivenessHandler())
	mux.HandleFunc(routes.Readiness, d.probe.ReadinessHandler())
	mux.HandleFunc(routes.Startup, d.probe.StartupHandler())
}

// Start launches background goroutines. When RefreshMode is SSE, this starts
// the pusher goroutine that broadcasts updates to connected clients. Call
// before serving HTTP traffic. For HTMX polling mode (default), Start is a
// no-op — the probe's own Start method handles the cache refresh loop.
//
// The ctx controls the lifetime of background goroutines. Call Shutdown to
// stop them cleanly.
func (d *Dashboard) Start(ctx context.Context) error {
	if d.cfg.RefreshMode == RefreshModeSSE {
		d.push = newSSEPusher(d)
		go d.push.start(ctx)
	}

	return nil
}

// Shutdown stops background goroutines. Safe to call multiple times.
func (d *Dashboard) Shutdown() {
	if d.push != nil {
		d.push.broadcaster.Close()
		d.push = nil
	}
}
