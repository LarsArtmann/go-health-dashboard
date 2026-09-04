package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

// Version is the current package version.
//
// CI enforces that this matches the latest git tag (the version-guard job
// in .github/workflows/ci.yml). When cutting a release, bump this const in
// the same commit you tag.
const Version = "0.6.0"

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

	// started records whether Start has ever been called, so HealthCheck
	// can distinguish "never started" from "shut down" (both have a nil
	// pusher pointer).
	started atomic.Bool
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

	cfg.resolveRoutes()

	if cfg.EmbeddedDatastarSDK {
		cfg.DatastarSrc = cfg.Routes.DatastarJS
	}

	cfg.PushInterval = resolvePushInterval(cfg.PushInterval, probe)

	d := &Dashboard{
		probe:   probe,
		cfg:     cfg,
		latency: newLatencyHistogram(),
		notify:  newWebhookNotifier(cfg),
	}

	if cfg.RateLimitRequests > 0 && cfg.RateLimitWindow > 0 {
		d.limiter = newRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	}

	return d
}

// Routes returns the dashboard's fully resolved routes: defaults or
// WithRoutes, then the WithBasePath prefix applied once after all options
// ran. Empty string values mean "endpoint disabled" and were not prefixed.
func (d *Dashboard) Routes() Routes {
	return d.cfg.Routes
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

// currentResponse returns the probe's cached health snapshot, sanitized for
// wire consumption: go-health's SanitizeResponse replaces invalid UTF-8 (a
// real possibility in service-supplied error strings) with U+FFFD, keeping
// every downstream write seam — JSON responses, webhook payloads, SSE
// patches, metrics labels, CSV export — valid under jsonv2 semantics.
func (d *Dashboard) currentResponse() health.Response {
	return health.SanitizeResponse(d.probe.CachedResponse())
}

// Start launches the SSE pusher goroutine that broadcasts health updates to
// connected clients. Call before serving HTTP traffic.
//
// The ctx controls the lifetime of the pusher goroutine. Call Shutdown to
// stop it cleanly.
func (d *Dashboard) Start(ctx context.Context) error {
	p := newPusher(d)

	d.push.Store(p)
	d.started.Store(true)

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

// ErrPusherNotActive is the parent sentinel for all "the real-time pusher is
// not running" errors returned by HealthCheck. Use it when you only care
// that the pusher is down; use ErrPusherNotStarted or ErrPusherShutDown when
// the distinction matters.
var ErrPusherNotActive = errors.New("dashboard: SSE pusher is not active")

// ErrPusherNotStarted is returned by HealthCheck when Start has never been
// called. It wraps ErrPusherNotActive.
var ErrPusherNotStarted = fmt.Errorf("%w: Start has not been called", ErrPusherNotActive)

// ErrPusherShutDown is returned by HealthCheck after Shutdown has stopped
// the pusher. It wraps ErrPusherNotActive.
var ErrPusherShutDown = fmt.Errorf("%w: Shutdown has been called", ErrPusherNotActive)

// ErrPusherStale is returned by HealthCheck when the pusher goroutine has
// not completed a broadcast tick recently (more than three push intervals).
// The watchdog is report-only: a wedged pusher is surfaced to container
// health checks, not restarted, so operators see the underlying problem.
var ErrPusherStale = errors.New("dashboard: SSE pusher stopped ticking")

// HealthCheck reports whether the dashboard's real-time update mechanism is
// healthy. Returns ErrPusherNotStarted when Start has never been called and
// ErrPusherShutDown after Shutdown (both detectable via
// errors.Is(err, ErrPusherNotActive)), and reports staleness when the push
// loop stopped ticking (watchdog; report-only, never restarts the
// goroutine).
//
// This method satisfies do.HealthcheckerWithContext, enabling the dashboard
// to participate in container-wide health checks when registered in a
// samber/do injector.
func (d *Dashboard) HealthCheck(_ context.Context) error {
	push := d.push.Load()
	if push == nil {
		if d.started.Load() {
			return ErrPusherShutDown
		}

		return ErrPusherNotStarted
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
