// Command health-hub serves a go-health-dashboard federating N remote
// go-health instances: one HTML dashboard, one JSON health document, one
// place to watch every service's existing go-health endpoint from.
//
// Remotes come from the environment and are never hardcoded:
//
//	HEALTH_HUB_REMOTES=cv=http://127.0.0.1:8080/health,forgejo=http://forgejo.home.lan:3000/health
//	HEALTH_HUB_TIMEOUT=5s        optional per-fetch deadline (default 5s)
//	HEALTH_HUB_TREND=1           enable the trend sparkline + timeline card
//	HEALTH_HUB_METRICS=1         serve Prometheus text at /health/metrics
//	HEALTH_HUB_PUSH_INTERVAL=10s  SSE push cadence (default 2s; every tick
//	                             fetches EVERY remote — raise this for LAN hubs)
//	HEALTH_HUB_SSE_DRAIN=5s      graceful SSE drain window on shutdown (default off)
//	HEALTH_HUB_CLIENT_TLS_CA=/path/ca.pem  trust this CA bundle for remote TLS
//	                             connections (self-signed/PKI remotes); validated
//	                             at startup, fails fast on a missing/PEM-less file
//	HEALTH_HUB_HTTP_PROXY=http://proxy:3128  route remote fetches through this
//	                             proxy (absolute http/https URL; credentials
//	                             allowed in the URL, never logged)
//	PORT=8080                    port to listen on
//	HEALTH_HUB_ADDR=127.0.0.1:8080  full listen address (overrides PORT)
//
// The hub serves /version (go-health VersionHandler, {"version":"..."}) so
// "which build am I hitting?" is answerable at the HTTP level. Per-process
// scalars deliberately do NOT ride the merged /health document: federation
// drops Version/Uptime/InstanceID in the merge (they would lie about a
// federated view), so WithInstanceID has nothing to attach to here — that
// surface belongs to the remotes themselves. The same asymmetry applies to
// ADR-0003's evaluation sink: federation.Prober has no evaluation hook, so
// probe-cadence Observe wiring belongs on the REMOTES (each remote's
// WithEvaluationHook), not on the hub.
//
// Checks land namespaced as "name/check" (worst-of across remotes); a
// dark remote surfaces as a "name/reachable" fail row instead of a silent
// freeze. The dashboard groups cards per remote, so the hub reads as one
// card per service.
//
// Drain-safety limitation: go-health's federation Prober has no
// MarkShuttingDown, so /readyz keeps answering from merge-on-read fetches
// until the listener closes — the hub cannot flip its own readiness to 503
// during a drain the way a single probe can. Shutdown therefore leads with
// the dashboard (SSE clients disconnected promptly instead of holding the
// server open for the full grace window); flipping readiness at the source
// is tracked as a go-health upstream proposal.
//
// The server root (/) redirects to the dashboard page — the vhost in
// front of the hub proxies every path, and a bare 404 on "/" reads as
// an outage to anyone typing the bare hostname. Unknown paths keep 404ing.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/larsartmann/go-health-dashboard/pkg/version"
	healthfederation "github.com/larsartmann/go-health/federation"
)

const (
	defaultPort        = "8080"
	trendSamples       = 1800
	remoteEnvVar       = "HEALTH_HUB_REMOTES"
	timeoutEnvVar      = "HEALTH_HUB_TIMEOUT"
	trendEnvVar        = "HEALTH_HUB_TREND"
	metricsEnvVar      = "HEALTH_HUB_METRICS"
	pushIntervalEnvVar = "HEALTH_HUB_PUSH_INTERVAL"
	sseDrainEnvVar     = "HEALTH_HUB_SSE_DRAIN"
	clientCAEnvVar     = "HEALTH_HUB_CLIENT_TLS_CA"
	proxyEnvVar        = "HEALTH_HUB_HTTP_PROXY"
	portEnvVar         = "PORT"
	addrEnvVar         = "HEALTH_HUB_ADDR"
	shutdownGrace      = 10 * time.Second
	readHeaderTimeout  = 5 * time.Second
	defaultFetchExpiry = 5 * time.Second
	startupGate        = 15 * time.Second
	startupPollStep    = 100 * time.Millisecond
)

var (
	errNoRemotesSpecified = errors.New(
		"at least one name=url entry is required (comma-separated, e.g. cv=http://127.0.0.1:8080/health)",
	)
	errMalformedEntry = errors.New(
		"want name=url with a non-empty name and an absolute http(s) URL",
	)
	errNonAbsoluteURL   = errors.New("URL must be absolute http(s)")
	errSlashInName      = errors.New(`name must not contain "/" (it prefixes every check key)`)
	errWhitespaceInName = errors.New(
		"name must not contain whitespace (it prefixes every check key, and systemd environment values cannot carry it)",
	)
	errDuplicateName      = errors.New("duplicate remote name")
	errNonPositiveTimeout = errors.New("want a positive duration (e.g. 3s, 500ms)")
)

// hubConfig is everything the environment contributes before the process
// starts doing anything irreversible.
type hubConfig struct {
	remotes      []healthfederation.Remote
	fetchExpiry  time.Duration
	pushInterval time.Duration // zero = library default
	sseDrain     time.Duration // zero = disabled (library default)
	tlsCAPath    string        // empty = system trust store
	httpProxy    string        // empty = no proxy override
}

// configFromEnv reads and validates all environment configuration up
// front, so a bad value fails before any goroutine, defer, or network
// exists.
func configFromEnv() (hubConfig, error) {
	remotes, err := parseRemotes(os.Getenv(remoteEnvVar))
	if err != nil {
		return hubConfig{}, fmt.Errorf("%s: %w", remoteEnvVar, err)
	}

	fetchExpiry := defaultFetchExpiry

	if raw := os.Getenv(timeoutEnvVar); raw != "" {
		fetchExpiry, err = parseTimeout(raw)
		if err != nil {
			return hubConfig{}, fmt.Errorf("%s: %w", timeoutEnvVar, err)
		}
	}

	pushInterval, err := parsePushInterval(os.Getenv(pushIntervalEnvVar))
	if err != nil {
		return hubConfig{}, fmt.Errorf("%s: %w", pushIntervalEnvVar, err)
	}

	var sseDrain time.Duration

	if raw := os.Getenv(sseDrainEnvVar); raw != "" {
		sseDrain, err = parseTimeout(raw)
		if err != nil {
			return hubConfig{}, fmt.Errorf("%s: %w", sseDrainEnvVar, err)
		}
	}

	return hubConfig{
		remotes:      remotes,
		fetchExpiry:  fetchExpiry,
		pushInterval: pushInterval,
		sseDrain:     sseDrain,
		tlsCAPath:    strings.TrimSpace(os.Getenv(clientCAEnvVar)),
		httpProxy:    strings.TrimSpace(os.Getenv(proxyEnvVar)),
	}, nil
}

// parsePushInterval validates the SSE push cadence from the environment.
// Empty means "library default" (2s today). Every tick fetches every
// remote once (merge-on-read), so a short cadence on a LAN hub multiplies
// into tens of thousands of fetches per remote per day — the value is
// surfaced in the startup log so the cost is visible, not buried.
func parsePushInterval(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}

	interval, err := time.ParseDuration(raw)
	if err != nil || interval <= 0 {
		return 0, fmt.Errorf("%w, got %q", errNonPositiveTimeout, raw)
	}

	return interval, nil
}

// resolvedPushInterval renders the effective push cadence for the startup
// log: the library default when the environment left it unset, otherwise
// the configured value.
func resolvedPushInterval(configured time.Duration) time.Duration {
	if configured != 0 {
		return configured
	}

	return dashboard.DefaultPushInterval
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := configFromEnv()
	if err != nil {
		return err
	}

	client, err := federationClient(cfg)
	if err != nil {
		return err
	}

	fedOpts := []healthfederation.Option{healthfederation.WithTimeout(cfg.fetchExpiry)}
	if client != nil {
		fedOpts = append(fedOpts, healthfederation.WithClient(client))
		log.Printf(
			"transport: custom HTTP client for remote fetches (TLS CA: %s, proxy: %s)",
			transportTLSCADescription(cfg),
			transportProxyDescription(cfg),
		)
	}

	fed, err := healthfederation.New(cfg.remotes, fedOpts...)
	if err != nil {
		return fmt.Errorf("federation.New: %w", err)
	}

	opts := append(
		[]dashboard.Option{dashboard.WithGrouping(dashboard.GroupBySource)},
		optionsFromEnv()...,
	)

	if cfg.pushInterval != 0 {
		opts = append(opts, dashboard.WithPushInterval(cfg.pushInterval))
	}

	if cfg.sseDrain > 0 {
		opts = append(opts, dashboard.WithShutdownDrain(cfg.sseDrain))
		log.Printf("sse drain: graceful SSE drain window %s on shutdown", cfg.sseDrain)
	}

	dash := dashboard.New(fed, opts...)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer cancel()

	if err := dash.Start(ctx); err != nil {
		return fmt.Errorf("dash.Start: %w", err)
	}

	defer dash.Shutdown()

	mux := newServeMux(dash)

	addr := envOrDefault(addrEnvVar, ":"+envOrDefault(portEnvVar, defaultPort))

	log.Printf(
		"health-hub: serving /health on %s (build %s, %d remotes, fetch timeout %s, push interval %s)",
		logSafe(addr),
		version.Version,
		len(cfg.remotes),
		cfg.fetchExpiry,
		resolvedPushInterval(cfg.pushInterval),
	)

	logRemotes(cfg.remotes)

	gateCtx, gateCancel := context.WithTimeout(ctx, startupGate)
	if waitForStartup(fed, gateCtx) {
		log.Println("startup gate: every remote has answered at least once")
	} else {
		log.Printf(
			"startup gate: not every remote answered within %s (serving anyway; dark remotes surface as reachable rows)",
			startupGate,
		)
	}
	gateCancel()

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	listenErr := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		// Drain-safe order: stop the dashboard BEFORE the HTTP server so
		// SSE clients are disconnected promptly (and, with a drain window,
		// new SSE connections already see 503) instead of their open
		// streams holding server.Shutdown open for the entire grace
		// window. The deferred dash.Shutdown stays as the early-exit
		// safety net and is idempotent. Federation cannot flip /readyz to
		// 503 (no MarkShuttingDown) - documented in the package comment.
		dash.Shutdown()
		return shutdown(server)
	case err := <-listenErr:
		return fmt.Errorf("server: %w", err)
	}
}

// waitForStartup polls the federation's startup latch (every remote has
// answered successfully at least once) until it completes or ctx is done.
func waitForStartup(fed *healthfederation.Prober, ctx context.Context) bool {
	for {
		if fed.StartupComplete() {
			return true
		}

		if ctx.Err() != nil {
			return false
		}

		time.Sleep(startupPollStep)
	}
}

// newServeMux wires the dashboard routes plus an exact-root redirect onto
// one handler. The hub never overrides Routes, so the library default IS
// the served dashboard path; if a route option ever lands here, read the
// resolved route from the dashboard instead of DefaultRoutes.
func newServeMux(dash *dashboard.Dashboard) *http.ServeMux {
	mux := http.NewServeMux()

	dash.RegisterRoutes(mux)
	mux.HandleFunc("/version", health.VersionHandler(version.Version))
	mux.Handle(
		"/{$}",
		http.RedirectHandler(dashboard.DefaultRoutes().Dashboard, http.StatusTemporaryRedirect),
	)

	return mux
}

// federationClient builds the HTTP client the federation fetches with when
// the transport knobs are set (TLS CA bundle, proxy); nil means "library
// default client". Validation is a startup concern: a missing CA file or a
// non-absolute proxy URL fails before any goroutine exists (the fleet's
// validate-before-use pattern).
func federationClient(cfg hubConfig) (*http.Client, error) {
	if cfg.tlsCAPath == "" && cfg.httpProxy == "" {
		return nil, nil
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()

	if cfg.httpProxy != "" {
		parsed, err := url.Parse(cfg.httpProxy)
		switch {
		case err != nil:
			return nil, fmt.Errorf("%s: %w", proxyEnvVar, err)
		case (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "":
			return nil, fmt.Errorf("%s: %w", proxyEnvVar, errNonAbsoluteURL)
		}

		transport.Proxy = http.ProxyURL(parsed)
	}

	if cfg.tlsCAPath != "" {
		pem, err := os.ReadFile(cfg.tlsCAPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", clientCAEnvVar, err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf(
				"%s: no PEM certificates found in %q",
				clientCAEnvVar,
				cfg.tlsCAPath,
			)
		}

		transport.TLSClientConfig = &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		}
	}

	// No client-level Timeout: federation bounds every fetch with its own
	// per-fetch deadline (HEALTH_HUB_TIMEOUT); a global cap here would only
	// duplicate it.
	return &http.Client{Transport: transport}, nil
}

// transportTLSCADescription renders the CA knob for the startup log.
func transportTLSCADescription(cfg hubConfig) string {
	if cfg.tlsCAPath == "" {
		return "system trust store"
	}

	return "custom bundle"
}

// transportProxyDescription renders the proxy knob redacted: the URL may
// embed credentials, and they never reach the log.
func transportProxyDescription(cfg hubConfig) string {
	if cfg.httpProxy == "" {
		return "none"
	}

	parsed, err := url.Parse(cfg.httpProxy)
	if err != nil {
		return "(unloggable proxy URL)"
	}

	return parsed.Redacted()
}

// logRemotes announces each remote with its credentials redacted, so an
// operator can verify the configuration from the log alone.
func logRemotes(remotes []healthfederation.Remote) {
	for _, remote := range remotes {
		parsed, parseErr := url.Parse(remote.URL)
		if parseErr != nil {
			log.Printf("remote: %s -> (unloggable URL)", logSafe(remote.Name))

			continue
		}

		log.Printf("remote: %s -> %s", logSafe(remote.Name), logSafe(parsed.Redacted()))
	}
}

// shutdown drains the server within the grace window after the context
// is cancelled.
func shutdown(server *http.Server) error {
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)

	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server.Shutdown: %v", err)
	}

	return nil
}

// logSafe strips control characters so a hostile environment value cannot
// forge or truncate log lines (the fleet's log-injection defense).
func logSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}

		return r
	}, s)
}

// parseTimeout validates a raw duration from the environment. Every
// failure names the offending value so a bad unit in a unit file is
// fixable from the log line alone.
func parseTimeout(raw string) (time.Duration, error) {
	expiry, err := time.ParseDuration(raw)
	if err != nil || expiry <= 0 {
		return 0, fmt.Errorf("%w, got %q", errNonPositiveTimeout, raw)
	}

	return expiry, nil
}

// optionsFromEnv assembles the optional dashboard features from the
// environment so a hub deployment turns capabilities on without code
// changes. Values are validated before any reaches a log line.
func optionsFromEnv() []dashboard.Option {
	var opts []dashboard.Option

	if os.Getenv(trendEnvVar) != "" {
		opts = append(opts, dashboard.WithTrend(trendSamples))

		log.Printf("trend: enabled (%d samples)", trendSamples)
	}

	if os.Getenv(metricsEnvVar) != "" {
		opts = append(opts, dashboard.WithMetrics(true))

		log.Printf("metrics: serving Prometheus text on /health/metrics")
	}

	return opts
}

// parseRemotes turns "name=url,name=url" into federation remotes. Every
// failure names the offending entry and the expected shape, so a typo in
// a long list is fixable from the log line alone.
func parseRemotes(spec string) ([]healthfederation.Remote, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, errNoRemotesSpecified
	}

	entries := strings.Split(spec, ",")
	remotes := make([]healthfederation.Remote, 0, len(entries))

	seen := make(map[string]bool, len(entries))

	for i, entry := range entries {
		remote, err := parseRemoteEntry(i, entry, seen)
		if err != nil {
			return nil, err
		}

		seen[remote.Name] = true

		remotes = append(remotes, remote)
	}

	return remotes, nil
}

// parseRemoteEntry validates one "name=url" entry against the shared
// seen-set. Every failure wraps a sentinel from this file with the entry
// index and the offending text, so errors.Is stays usable while the log
// line still pinpoints the entry.
func parseRemoteEntry(i int, entry string, seen map[string]bool) (healthfederation.Remote, error) {
	name, rawURL, found := strings.Cut(entry, "=")

	name = strings.TrimSpace(name)
	rawURL = strings.TrimSpace(rawURL)

	if !found || name == "" || rawURL == "" {
		return healthfederation.Remote{}, fmt.Errorf(
			"entry %d (%q): %w",
			i,
			entry,
			errMalformedEntry,
		)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return healthfederation.Remote{}, fmt.Errorf("entry %d (%s): invalid URL: %w", i, name, err)
	}

	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return healthfederation.Remote{}, fmt.Errorf(
			"entry %d (%s): %w, got %q",
			i,
			name,
			errNonAbsoluteURL,
			parsed.Redacted(),
		)
	}

	if strings.Contains(name, "/") {
		return healthfederation.Remote{}, fmt.Errorf("entry %d: %w", i, errSlashInName)
	}

	if strings.ContainsFunc(name, unicode.IsSpace) {
		return healthfederation.Remote{}, fmt.Errorf("entry %d: %w", i, errWhitespaceInName)
	}

	if seen[name] {
		return healthfederation.Remote{}, fmt.Errorf("entry %d: %w %q", i, errDuplicateName, name)
	}

	return healthfederation.Remote{Name: name, URL: rawURL}, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
