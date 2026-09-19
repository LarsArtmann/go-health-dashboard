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
//	PORT=8080                    port to listen on
//	HEALTH_HUB_ADDR=127.0.0.1:8080  full listen address (overrides PORT)
//
// Checks land namespaced as "name/check" (worst-of across remotes); a
// dark remote surfaces as a "name/reachable" fail row instead of a silent
// freeze. The dashboard groups cards per remote, so the hub reads as one
// card per service.
package main

import (
	"context"
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
	portEnvVar         = "PORT"
	addrEnvVar         = "HEALTH_HUB_ADDR"
	shutdownGrace      = 10 * time.Second
	readHeaderTimeout  = 5 * time.Second
	defaultFetchExpiry = 5 * time.Second
)

var (
	errNoRemotesSpecified = errors.New("at least one name=url entry is required (comma-separated, e.g. cv=http://127.0.0.1:8080/health)")
	errMalformedEntry     = errors.New("want name=url with a non-empty name and an absolute http(s) URL")
	errNonAbsoluteURL     = errors.New("URL must be absolute http(s)")
	errSlashInName        = errors.New(`name must not contain "/" (it prefixes every check key)`)
	errWhitespaceInName   = errors.New("name must not contain whitespace (it prefixes every check key, and systemd environment values cannot carry it)")
	errDuplicateName      = errors.New("duplicate remote name")
	errNonPositiveTimeout = errors.New("want a positive duration (e.g. 3s, 500ms)")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	remotes, err := parseRemotes(os.Getenv(remoteEnvVar))
	if err != nil {
		return fmt.Errorf("%s: %w", remoteEnvVar, err)
	}

	fetchExpiry := defaultFetchExpiry

	if raw := os.Getenv(timeoutEnvVar); raw != "" {
		fetchExpiry, err = parseTimeout(raw)

		if err != nil {
			return fmt.Errorf("%s: %w", timeoutEnvVar, err)
		}
	}

	fed, err := healthfederation.New(remotes, healthfederation.WithTimeout(fetchExpiry))
	if err != nil {
		return fmt.Errorf("federation.New: %w", err)
	}

	opts := append(
		[]dashboard.Option{dashboard.WithGrouping(dashboard.GroupBySource)},
		optionsFromEnv()...,
	)

	dash := dashboard.New(fed, opts...)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer cancel()

	if err := dash.Start(ctx); err != nil {
		return fmt.Errorf("dash.Start: %w", err)
	}

	defer dash.Shutdown()

	mux := http.NewServeMux()

	dash.RegisterRoutes(mux)

	addr := envOrDefault(addrEnvVar, ":"+envOrDefault(portEnvVar, defaultPort))

	log.Printf(
		"health-hub: serving /health on %s (build %s, %d remotes, fetch timeout %s)",
		logSafe(addr),
		version.Version,
		len(remotes),
		fetchExpiry,
	)

	for _, remote := range remotes {
		parsed, parseErr := url.Parse(remote.URL)

		if parseErr != nil {
			log.Printf("remote: %s -> (unloggable URL)", logSafe(remote.Name))

			continue
		}

		log.Printf("remote: %s -> %s", logSafe(remote.Name), logSafe(parsed.Redacted()))
	}

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
		log.Println("shutting down...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownGrace)

		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server.Shutdown: %v", err)
		}

		return nil
	case err := <-listenErr:
		return fmt.Errorf("server: %w", err)
	}
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
		return healthfederation.Remote{}, fmt.Errorf("entry %d (%q): %w", i, entry, errMalformedEntry)
	}

	parsed, err := url.Parse(rawURL)

	if err != nil {
		return healthfederation.Remote{}, fmt.Errorf("entry %d (%s): invalid URL: %w", i, name, err)
	}

	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return healthfederation.Remote{}, fmt.Errorf("entry %d (%s): %w, got %q", i, name, errNonAbsoluteURL, parsed.Redacted())
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
