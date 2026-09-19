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

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	remotes, err := parseRemotes(os.Getenv(remoteEnvVar))
	if err != nil {
		log.Fatalf("%s: %v", remoteEnvVar, err)
	}

	fetchExpiry := defaultFetchExpiry
	if raw := os.Getenv(timeoutEnvVar); raw != "" {
		fetchExpiry, err = parseTimeout(raw)
		if err != nil {
			log.Fatalf("%s: %v", timeoutEnvVar, err)
		}
	}

	fed, err := healthfederation.New(remotes, healthfederation.WithTimeout(fetchExpiry))
	if err != nil {
		log.Fatalf("federation.New: %v", err)
	}

	opts := append(
		[]dashboard.Option{dashboard.WithGrouping(dashboard.GroupBySource)},
		optionsFromEnv()...,
	)
	dash := dashboard.New(fed, opts...)

	if err := dash.Start(ctx); err != nil {
		log.Fatalf("dash.Start: %v", err)
	}
	defer dash.Shutdown()

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	addr := envOrDefault(addrEnvVar, ":"+envOrDefault(portEnvVar, defaultPort))
	log.Printf(
		"health-hub: serving /health on %s (build %s, %d remotes, fetch timeout %s)",
		addr,
		version.Version,
		len(remotes),
		fetchExpiry,
	)
	for _, remote := range remotes {
		parsed, parseErr := url.Parse(remote.URL)
		if parseErr != nil {
			log.Printf("remote: %s -> (unloggable URL)", remote.Name)

			continue
		}
		log.Printf("remote: %s -> %s", remote.Name, parsed.Redacted())
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server.Shutdown: %v", err)
	}
}

// parseTimeout validates a raw duration from the environment. Every
// failure names the offending value so a bad unit in a unit file is
// fixable from the log line alone.
func parseTimeout(raw string) (time.Duration, error) {
	expiry, err := time.ParseDuration(raw)
	if err != nil || expiry <= 0 {
		return 0, fmt.Errorf("want a positive duration (e.g. 3s, 500ms), got %q", raw)
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
		return nil, errors.New(
			"at least one name=url entry is required (comma-separated, e.g. cv=http://127.0.0.1:8080/health)",
		)
	}

	entries := strings.Split(spec, ",")
	remotes := make([]healthfederation.Remote, 0, len(entries))

	seen := make(map[string]bool, len(entries))

	for i, entry := range entries {
		name, rawURL, found := strings.Cut(entry, "=")
		name = strings.TrimSpace(name)
		rawURL = strings.TrimSpace(rawURL)

		if !found || name == "" || rawURL == "" {
			return nil, fmt.Errorf(
				"entry %d (%q): want name=url with a non-empty name and an absolute http(s) URL",
				i, entry,
			)
		}

		parsed, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("entry %d (%s): invalid URL: %w", i, name, err)
		}

		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, fmt.Errorf(
				"entry %d (%s): URL must be absolute http(s), got %q",
				i, name, parsed.Redacted(),
			)
		}

		if strings.Contains(name, "/") {
			return nil, fmt.Errorf(
				"entry %d: name must not contain %q (it prefixes every check key)", i, "/",
			)
		}

		if strings.ContainsFunc(name, unicode.IsSpace) {
			return nil, fmt.Errorf(
				"entry %d: name %q must not contain whitespace (it prefixes every check key, and systemd environment values cannot carry it)",
				i, name,
			)
		}

		if seen[name] {
			return nil, fmt.Errorf("entry %d: duplicate remote name %q", i, name)
		}
		seen[name] = true

		remotes = append(remotes, healthfederation.Remote{Name: name, URL: rawURL})
	}

	return remotes, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
