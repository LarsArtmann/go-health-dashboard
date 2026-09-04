package dashboard

import (
	"net/http"
	"strings"
	"time"
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
	Introspection     bool
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

	// BasePath is stored by WithBasePath and applied to Routes once after
	// all options run (see resolveRoutes). Empty means no prefix.
	BasePath string

	// EmbeddedDatastarSDK registers Routes.DatastarJS serving the pinned
	// SDK bundle from the go-datastar/static embed, and points the HTML's
	// script src at it. Set via WithEmbeddedDatastarSDK.
	EmbeddedDatastarSDK bool

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
// The prefix is stored and applied once after all options run, so the order
// of WithBasePath relative to WithRoutes does not matter (the historical
// ordering footgun is gone).
func WithBasePath(prefix string) Option {
	return func(cfg *Config) {
		cfg.BasePath = strings.TrimSuffix(prefix, "/")
	}
}

// resolveRoutes applies cfg.BasePath to cfg.Routes. Empty routes stay
// empty ("disabled") and never become the bare prefix.
func (c *Config) resolveRoutes() {
	if c.BasePath == "" {
		return
	}

	prefix := c.BasePath
	r := c.Routes

	out := Routes{
		Dashboard:  prefix + r.Dashboard,
		SSE:        prefix + r.SSE,
		Favicon:    prefix + r.Favicon,
		Liveness:   prefix + r.Liveness,
		Readiness:  prefix + r.Readiness,
		Startup:    prefix + r.Startup,
		Introspect: prefix + r.Introspect,
	}

	// Empty Metrics route is meaningful ("disabled") — don't turn it into
	// the bare prefix. Same for Trend/Export/Introspect.
	if r.Metrics != "" {
		out.Metrics = prefix + r.Metrics
	}

	if r.Trend != "" {
		out.Trend = prefix + r.Trend
	}

	if r.Export != "" {
		out.Export = prefix + r.Export
	}

	if r.Introspect != "" {
		out.Introspect = prefix + r.Introspect
	}

	if r.DatastarJS != "" {
		out.DatastarJS = prefix + r.DatastarJS
	}

	c.Routes = out
}

// WithIntrospection enables the introspection endpoint served at
// Routes.Introspect (default /health/introspect) by RegisterRoutes. The
// endpoint returns the dashboard's resolved configuration — routes,
// limits, modes, versions — as JSON. It exposes route paths and feature
// flags but never check results or check names; gate it with
// WithMiddleware if that disclosure matters in your environment.
func WithIntrospection() Option {
	return func(c *Config) {
		c.Introspection = true
	}
}

// WithEmbeddedDatastarSDK serves the pinned Datastar SDK from the
// go-datastar/static embed at Routes.DatastarJS (default
// /health/datastar.js) and points the dashboard's script tag at it. This
// removes both the CDN dependency and the CSP exception it would need: the
// script becomes same-origin, so `script-src 'self'` (plus the
// 'unsafe-eval' the SDK itself requires for expression compilation)
// is sufficient. The served bytes are exactly the audited bundle from the
// pinned go-datastar version.
func WithEmbeddedDatastarSDK() Option {
	return func(c *Config) {
		c.EmbeddedDatastarSDK = true
	}
}
