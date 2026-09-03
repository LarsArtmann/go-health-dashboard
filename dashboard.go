package dashboard

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
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
const Version = "0.4.0"

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
	Middleware        func(http.Handler) http.Handler
	MetricsEnabled    bool
	TrendSamples      int
	HideStatCards     bool

	// ShutdownDrain bounds how long Shutdown waits for connected SSE
	// clients to disconnect before closing the broadcaster. Zero closes
	// immediately (default).
	ShutdownDrain time.Duration

	// MaxConnectionLifetime caps how long a single SSE connection may stay
	// open. Zero means unlimited (default). Clients reconnect automatically
	// via the SSE retry field.
	MaxConnectionLifetime time.Duration

	// Description is rendered as the meta description plus Open Graph
	// og:title/og:description tags. Empty (default) omits the tags entirely.
	Description string

	// PublicMode anonymizes the dashboard for untrusted audiences: check
	// names and error details are replaced with generic labels in the HTML
	// and the metrics endpoint. Health JSON and probe endpoints are
	// unaffected.
	PublicMode bool

	// RateLimitRequests and RateLimitWindow configure a shared token
	// bucket across all dashboard-owned routes. Zero disables rate
	// limiting (default). Probe endpoints are never limited.
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// WebhookURL receives a JSON status snapshot on every health-state
	// transition (change-only, independent of PushMode). Empty disables the
	// webhook (default). Deliveries are best-effort with a 10s timeout and
	// no retry — the receiver owns alert thresholds. The URL may embed a
	// secret; it is never logged.
	WebhookURL string

	// WebhookHeaders are sent with every webhook delivery, typically for
	// receiver authentication (e.g. {"Authorization": "Bearer ..."}).
	// Nil/empty (default) sends only Content-Type.
	WebhookHeaders map[string]string
}

// WithMiddleware wraps every dashboard-owned handler (dashboard HTML, SSE,
// favicon, and the metrics endpoint when enabled) with the given middleware.
// Use it to protect the dashboard with authentication — for example Basic
// Auth, a bearer token check, or a session middleware from the host
// application:
//
//	dash := dashboard.New(probe,
//		dashboard.WithMiddleware(myAuthMiddleware),
//	)
//
// Kubernetes probe endpoints (/healthz, /readyz, /startupz) are deliberately
// NOT wrapped: the kubelet cannot authenticate, and gating them breaks
// liveness and readiness gates. Register probe handlers separately on a
// middleware-free mux if your environment requires protecting them too.
//
// When multiple middlewares are needed, compose them explicitly — the
// last one registered runs closest to the handler:
//
//	dashboard.WithMiddleware(chain(mwOne, mwTwo))
func WithMiddleware(mw func(http.Handler) http.Handler) Option {
	return func(c *Config) { c.Middleware = mw }
}

// WithMetrics enables the Prometheus metrics endpoint served at
// Routes.Metrics (default /health/metrics) by RegisterRoutes. The endpoint
// is disabled by default — it exposes per-check detail, so opt in and
// protect it like the dashboard itself (WithMiddleware applies to it too).
func WithMetrics(enabled bool) Option {
	return func(c *Config) { c.MetricsEnabled = enabled }
}

// WithTrend enables the health trend sparkline and sets how many status
// samples it retains. One sample is recorded per push interval (pass=1,
// warn=0.5, fail=0); the card appears once at least two samples exist.
// A sample count of 60 at the default 2s interval covers the last two
// minutes. Non-positive values leave the trend disabled (the default).
func WithTrend(samples int) Option {
	return func(c *Config) {
		if samples > 0 {
			c.TrendSamples = samples
		}
	}
}

// WithHideStatCards hides the version/uptime/latency stat card grid.
// Use this for compact dashboards where only the status banner and service
// tables matter.
func WithHideStatCards() Option {
	return func(c *Config) { c.HideStatCards = true }
}

// WithShutdownDrain makes Shutdown wait up to d for connected SSE clients
// to disconnect before closing the broadcaster. New connections are rejected
// immediately during the drain window, so load balancers draining traffic
// see fast 503s while existing browsers keep their streams. Zero (default)
// closes all streams immediately.
func WithShutdownDrain(d time.Duration) Option {
	return func(c *Config) {
		c.ShutdownDrain = d
	}
}

// WithMaxConnectionLifetime caps how long a single SSE connection may stay
// open. When the cap hits, the server closes the stream; the browser
// reconnects automatically (SSE semantics, optionally tuned via
// WithRetryInterval). Useful behind load balancers that recycle
// long-lived connections.
func WithMaxConnectionLifetime(d time.Duration) Option {
	return func(c *Config) {
		c.MaxConnectionLifetime = d
	}
}

// WithRateLimit caps dashboard-owned routes (dashboard HTML, SSE, favicon,
// metrics) with a shared token bucket: bursts up to maxRequests, refilling
// continuously at maxRequests per window. Excess requests receive 429 with
// a Retry-After header. Kubernetes probe endpoints are never limited.
func WithRateLimit(maxRequests int, window time.Duration) Option {
	return func(c *Config) {
		c.RateLimitRequests = maxRequests
		c.RateLimitWindow = window
	}
}

// WithDescription sets the meta description and Open Graph tags for the
// dashboard page. Empty by default, which omits the tags.
func WithDescription(description string) Option {
	return func(c *Config) {
		c.Description = description
	}
}

// WithPublicMode renders the dashboard for untrusted audiences: check names
// and error details become generic labels in the HTML, and the metrics
// endpoint labels checks opaquely. The /health JSON response and the
// Kubernetes probe endpoints are unaffected.
func WithPublicMode() Option {
	return func(c *Config) {
		c.PublicMode = true
	}
}

// WithWebhook pushes a JSON health snapshot to url on every status
// transition — including the initial state on Start — so egress-restricted
// deployments (NAT, serverless) can feed event ingests and monitoring
// pipelines without a scraper. Change detection is independent of PushMode:
// a PushAlways dashboard still fires the webhook only on transitions.
//
// Deliveries are best-effort: one goroutine per fire with a 10s timeout, no
// retries, no logging. Use WithWebhookHeaders for receiver authentication.
// Combine with WithPublicMode to mask check names and error details before
// they leave the process.
func WithWebhook(url string) Option {
	return func(cfg *Config) {
		cfg.WebhookURL = url
	}
}

// WithWebhookHeaders sets headers sent with every webhook delivery, usually
// receiver authentication:
//
//	dashboard.WithWebhookHeaders(map[string]string{
//		"Authorization": "Bearer " + token,
//	})
func WithWebhookHeaders(headers map[string]string) Option {
	return func(cfg *Config) {
		cfg.WebhookHeaders = headers
	}
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
		out := Routes{
			Dashboard: prefix + r.Dashboard,
			SSE:       prefix + r.SSE,
			Favicon:   prefix + r.Favicon,
			Liveness:  prefix + r.Liveness,
			Readiness: prefix + r.Readiness,
			Startup:   prefix + r.Startup,
		}

		// An empty Metrics route is meaningful ("disabled") — don't turn it
		// into the bare prefix.
		if r.Metrics != "" {
			out.Metrics = prefix + r.Metrics
		}

		// Same for Trend/Export: empty means disabled, and they only
		// register when WithTrend is configured anyway.
		if r.Trend != "" {
			out.Trend = prefix + r.Trend
		}

		if r.Export != "" {
			out.Export = prefix + r.Export
		}

		cfg.Routes = out
	}
}

// Prober is the minimal health-probe surface the dashboard renders. It is
// satisfied by *health.Probe (the common case) and by go-health's
// aggregate.Aggregate, so one dashboard can serve a merged multi-service
// view without the dashboard importing or knowing the concrete source type.
// The interface lives on the consumer side by Go convention: providers stay
// concrete, consumers define what they need.
type Prober interface {
	// CachedResponse returns the current health snapshot without running
	// dependency checks. The pusher reads it once per push interval.
	CachedResponse() health.Response
	// RefreshInterval reports the probe's cache refresh cadence. Used as the
	// default push interval; zero means the probe evaluates live.
	RefreshInterval() time.Duration
	// LivenessHandler serves the JSON liveness probe (always 200).
	LivenessHandler() http.HandlerFunc
	// ReadinessHandler serves the JSON readiness probe (503 on fail).
	ReadinessHandler() http.HandlerFunc
	// StartupHandler serves the JSON startup probe (503 until latched).
	StartupHandler() http.HandlerFunc
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
	probe   Prober
	cfg     Config
	push    atomic.Pointer[pusher]
	limiter *rateLimiter
	latency *latencyHistogram
	notify  *webhookNotifier
}

// Compile-time assertions that Dashboard satisfies samber/do lifecycle interfaces.
// These enable automatic participation in do.HealthCheck and do.Shutdown cascades
// when the Dashboard is registered in a DI container via Register.
var (
	_ do.HealthcheckerWithContext = (*Dashboard)(nil)
	_ do.Shutdowner               = (*Dashboard)(nil)
)

// New creates a Dashboard wired to the given Prober. The Prober provides
// health data (via CachedResponse) and JSON handlers (via ReadinessHandler,
// LivenessHandler, StartupHandler). Both *health.Probe and go-health's
// aggregate.Aggregate satisfy it — pass an aggregate to render several
// in-process probes as one multi-service dashboard.
//
// Default configuration:
//   - Title: "Health Dashboard"
//   - PushInterval: probe's RefreshInterval, or 2s if probe is live
//   - PushMode: PushOnChange
//   - Routes: DefaultRoutes()
func New(probe Prober, opts ...Option) *Dashboard {
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

	d := &Dashboard{probe: probe, cfg: cfg, latency: newLatencyHistogram(), notify: newWebhookNotifier(cfg)}

	if cfg.RateLimitRequests > 0 && cfg.RateLimitWindow > 0 {
		d.limiter = newRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	}

	return d
}

// resolvePushInterval determines the effective push interval. When the
// caller provides a positive interval via WithPushInterval, it wins.
// Otherwise we fall back to the probe's configured interval, or the default
// when the probe is in live mode (interval == 0).
func resolvePushInterval(configured time.Duration, probe Prober) time.Duration {
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

	mux.HandleFunc(routes.Liveness, d.probe.LivenessHandler())
	mux.HandleFunc(routes.Readiness, d.probe.ReadinessHandler())
	mux.HandleFunc(routes.Startup, d.probe.StartupHandler())
}

// populateHistory fills the sparkline values and the recent status-change
// timeline from the trend history. Shared by the initial HTML render and
// the SSE patches so both always agree.
const maxTimelineEntries = 5

func populateHistory(vm *viewModel, buffer *historyBuffer) {
	samples := buffer.snapshot()

	values := make([]float64, 0, len(samples))
	for _, s := range samples {
		values = append(values, s.Value)
	}

	vm.History = values

	transitions := buffer.transitions()
	if len(transitions) > maxTimelineEntries {
		transitions = transitions[len(transitions)-maxTimelineEntries:]
	}

	for _, tr := range transitions {
		vm.Timeline = append(vm.Timeline, TimelineEntry{
			At:       tr.At.Format("15:04:05"),
			Status:   tr.To,
			Degraded: tr.To != string(health.StatusPass),
		})
	}
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
// When WithShutdownDrain is configured, new connections are rejected
// immediately and existing clients get up to the configured window to
// disconnect before the broadcaster closes. Safe to call multiple times.
func (d *Dashboard) Shutdown() {
	p := d.push.Swap(nil)
	if p == nil {
		return
	}

	if drain := d.cfg.ShutdownDrain; drain > 0 {
		deadline := time.Now().Add(drain)

		for p.connections.Load() > 0 && time.Now().Before(deadline) {
			time.Sleep(2 * time.Millisecond)
		}
	}

	p.broadcaster.Close()
}

// ErrPusherNotActive is returned by HealthCheck when the SSE pusher has not
// been started or has been shut down.
var ErrPusherNotActive = errors.New("dashboard: SSE pusher is not active")

// ErrPusherStale is returned by HealthCheck when the pusher goroutine has
// not completed a broadcast tick recently (more than three push intervals).
// The watchdog is report-only: a wedged pusher is surfaced to container
// health checks, not restarted, so operators see the underlying problem.
var ErrPusherStale = errors.New("dashboard: SSE pusher stopped ticking")

// HealthCheck reports whether the dashboard's real-time update mechanism is
// healthy. Returns an error when the SSE pusher has not been started or has
// been shut down, and reports staleness when the push loop stopped ticking
// (watchdog; report-only, never restarts the goroutine).
//
// This method satisfies do.HealthcheckerWithContext, enabling the dashboard
// to participate in container-wide health checks when registered in a
// samber/do injector.
func (d *Dashboard) HealthCheck(_ context.Context) error {
	push := d.push.Load()
	if push == nil {
		return ErrPusherNotActive
	}

	if last := push.lastBroadcast.Load(); last != 0 && push.interval > 0 {
		const stalenessFactor = 3

		staleAfter := stalenessFactor * push.interval

		if elapsed := time.Since(time.Unix(0, last)); elapsed > staleAfter {
			return fmt.Errorf("%w: last broadcast %s ago, stale after %s",
				ErrPusherStale, elapsed.Round(time.Millisecond), staleAfter)
		}
	}

	return nil
}
