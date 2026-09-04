// Command example demonstrates the go-health-dashboard with mock services.
//
// Run with:
//
//	GOEXPERIMENT=jsonv2 go run ./example
//
// Then open http://localhost:8080/health in a browser to see the live dashboard.
// Kubelet-style JSON is available at http://localhost:8080/readyz.
//
// Optional environment toggles showcase the v0.3.x features:
//
//	DEMO_TREND=1                 enable the health trend sparkline (WithTrend)
//	DEMO_METRICS=1               enable the Prometheus endpoint (/health/metrics)
//	DEMO_AUTH=<token>            require "Authorization: Bearer <token>" on dashboard routes
//	DEMO_RATELIMIT=<n>/<window>  e.g. 30/1m — token-bucket limit on dashboard routes
//	DEMO_DRAIN=5s                graceful SSE drain window on shutdown
//	DEMO_PUBLIC=1                public status-page mode (WithPublicMode)
//	DEMO_BASE_PATH=/status       mount the dashboard under a sub-path (WithBasePath)
//	DEMO_AGGREGATE=1             serve a two-probe aggregate instead of one probe (go-health aggregate)
//	DEMO_WEBHOOK=<url>           POST transitions to this receiver (WithWebhook)
//	PORT=8080                    listen address
package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/go-health/aggregate"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	injector := do.New()
	defer func() { _ = injector.Shutdown() }()

	probe, shutdownProbe := buildProbe(ctx, injector)
	defer shutdownProbe()

	// Assemble the option set from environment toggles so every feature can
	// be demonstrated without code changes.
	dash := dashboard.Register(injector, probe, buildOptions()...)

	if err := dash.Start(ctx); err != nil {
		log.Fatalf("dash.Start: %v", err)
	}

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	addr := ":" + envOrDefault("PORT", "8080")
	log.Printf("dashboard: http://localhost%s/health", addr)
	log.Printf("readiness: http://localhost%s/readyz", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server until ctx is cancelled (SIGINT/SIGTERM), then shut down
	// gracefully: stop accepting new connections, wait for in-flight requests,
	// then let the deferred injector.Shutdown() cascade to all services.
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server.Shutdown: %v", err)
	}
}

// buildProbe constructs the health probe the dashboard renders. With
// DEMO_AGGREGATE=1 it builds two independent probes (api + worker service
// groups) and merges them with go-health's aggregate — demonstrating the
// multi-service dashboard from AGENTS.md — otherwise it builds the classic
// single probe. The returned shutdown func stops whichever probe was built.
func buildProbe(ctx context.Context, injector *do.RootScope) (dashboard.Prober, func()) {
	if os.Getenv("DEMO_AGGREGATE") == "" {
		registerService(injector, "postgres", &alwaysHealthy{})
		registerService(injector, "redis", &flappingService{failEvery: 15 * time.Second})
		registerService(
			injector,
			"metrics-exporter",
			&alwaysFailing{reason: "exporter endpoint unreachable"},
		)

		probe := health.New(injector,
			health.WithVersion("1.2.3"),
			health.WithCriticalServices("postgres", "redis"),
			health.WithRefreshInterval(2*time.Second),
		)

		if err := probe.Start(ctx); err != nil {
			log.Fatalf("probe.Start: %v", err)
		}

		return probe, probe.Shutdown
	}

	// Aggregate mode: two probes over disjoint service groups. Sources must
	// have unique, slash-free names (go-health v0.1.3 contract).
	apiInjector := do.New()
	registerService(apiInjector, "postgres", &alwaysHealthy{})
	registerService(apiInjector, "redis", &flappingService{failEvery: 15 * time.Second})

	workerInjector := do.New()
	registerService(workerInjector, "metrics-exporter", &alwaysFailing{reason: "exporter endpoint unreachable"})

	apiProbe := health.New(apiInjector,
		health.WithVersion("1.2.3"),
		health.WithCriticalServices("postgres"),
		health.WithRefreshInterval(2*time.Second),
	)
	workerProbe := health.New(workerInjector,
		health.WithRefreshInterval(2*time.Second),
	)

	agg, err := aggregate.New(
		aggregate.NamedSource("api", apiProbe),
		aggregate.NamedSource("worker", workerProbe),
	)
	if err != nil {
		log.Fatalf("aggregate.New: %v", err)
	}

	if err := apiProbe.Start(ctx); err != nil {
		log.Fatalf("api probe.Start: %v", err)
	}
	if err := workerProbe.Start(ctx); err != nil {
		log.Fatalf("worker probe.Start: %v", err)
	}

	return agg, func() {
		apiProbe.Shutdown()
		workerProbe.Shutdown()
	}
}

// buildOptions assembles the demo option set from environment toggles so
// every feature can be demonstrated without code changes.
func buildOptions() []dashboard.Option {
	opts := []dashboard.Option{
		dashboard.WithTitle("Demo Service"),
		dashboard.WithShutdownDrain(parseDuration("DEMO_DRAIN")),
	}

	if os.Getenv("DEMO_TREND") != "" {
		opts = append(opts, dashboard.WithTrend(120))
		log.Println("trend: enabled (health trend sparkline)")
	}

	if os.Getenv("DEMO_METRICS") != "" {
		opts = append(opts, dashboard.WithMetrics(true))
		log.Printf("metrics: http://localhost%s/health/metrics", ":"+envOrDefault("PORT", "8080"))
	}

	if token := os.Getenv("DEMO_AUTH"); token != "" {
		opts = append(opts, dashboard.WithMiddleware(bearerAuth(token)))
		log.Printf("auth: bearer token required on dashboard routes (DEMO_AUTH set)")
	}

	if spec := os.Getenv("DEMO_RATELIMIT"); spec != "" {
		maxReqs, window, err := parseRateLimit(spec)
		if err != nil {
			log.Fatalf(
				"DEMO_RATELIMIT: %v",
				err,
			)
		}

		opts = append(opts, dashboard.WithRateLimit(maxReqs, window))
		log.Printf("rate limit: enabled on dashboard routes")
	}

	if url := os.Getenv("DEMO_WEBHOOK"); url != "" {
		opts = append(opts, dashboard.WithWebhook(url))
		log.Printf("webhook: transitions POST to %s (DEMO_WEBHOOK set)", url)
	}

	if os.Getenv("DEMO_PUBLIC") != "" {
		opts = append(opts, dashboard.WithPublicMode())
		log.Println("public mode: check names and errors anonymized (DEMO_PUBLIC set)")
	}

	if spec := os.Getenv("DEMO_BASE_PATH"); spec != "" {
		basePath, err := safeBasePath(spec)
		if err != nil {
			log.Fatalf("DEMO_BASE_PATH: %v", err)
		}

		opts = append(opts, dashboard.WithBasePath(basePath))
		log.Println("base path: dashboard routes mounted under the DEMO_BASE_PATH prefix")
	}

	return opts
}

// safeBasePath validates an env-provided base path: it must be a plain URL
// path prefix (leading slash, no whitespace or control characters) so a
// hostile DEMO_BASE_PATH cannot inject anything into logs or routes.
func safeBasePath(spec string) (string, error) {
	if !strings.HasPrefix(spec, "/") {
		return "", fmt.Errorf("want a path starting with /, got %q", spec)
	}

	if strings.ContainsFunc(spec, func(r rune) bool { return !isBasePathRune(r) }) {
		return "", fmt.Errorf("unsupported character in %q (want [A-Za-z0-9/_.-])", spec)
	}

	return spec, nil
}

// isBasePathRune reports whether r may appear in a base path prefix.
func isBasePathRune(r rune) bool {
	switch {
	case r == '/' || r == '_' || r == '-' || r == '.':
		return true
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return false
	}
}

// healthChecker is the interface samber/do uses for health checks.
type healthChecker interface {
	HealthCheck(ctx context.Context) error
}

// registerService registers a named service in the injector. The service must
// implement healthChecker so it shows up in health-check batches.
func registerService(injector do.Injector, name string, svc healthChecker) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (healthChecker, error) {
		return svc, nil
	})

	if _, err := do.InvokeNamed[healthChecker](injector, name); err != nil {
		log.Fatalf("invoke %s: %v", name, err)
	}
}

// --- Mock services ---.

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultVal
}

type alwaysHealthy struct{}

func (*alwaysHealthy) HealthCheck(_ context.Context) error { return nil }

type alwaysFailing struct{ reason string }

func (s *alwaysFailing) HealthCheck(_ context.Context) error {
	return fmt.Errorf("unhealthy: %s", s.reason)
}

// flappingService alternates between healthy and unhealthy on a cycle.
type flappingService struct {
	failEvery time.Duration
	startedAt time.Time
}

func (s *flappingService) HealthCheck(_ context.Context) error {
	if s.startedAt.IsZero() {
		s.startedAt = time.Now()
	}

	elapsed := time.Since(s.startedAt)
	if int(elapsed/s.failEvery)%2 == 1 {
		return fmt.Errorf("flapping: in failure window (elapsed=%s)", elapsed.Round(time.Second))
	}

	return nil
}

// bearerAuth returns middleware enforcing "Authorization: Bearer <token>".
// This is demo scaffolding; production services should use their existing
// auth middleware.
func bearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="demo"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseDuration reads an env var as a time.Duration, returning 0 when unset.
func parseDuration(key string) time.Duration {
	spec := os.Getenv(key)
	if spec == "" {
		return 0
	}

	d, err := time.ParseDuration(spec)
	if err != nil || d < 0 {
		log.Fatalf("%s: invalid duration", key)
	}

	return d
}

// parseRateLimit parses "30/1m", "5/s", "100/1h" style specifications.
func parseRateLimit(spec string) (int, time.Duration, error) {
	countStr, windowStr, found := strings.Cut(spec, "/")
	if !found {
		return 0, 0, fmt.Errorf("want <requests>/<window> (e.g. 30/1m), got %q", spec)
	}

	maxReqs, err := strconv.Atoi(countStr)
	if err != nil || maxReqs < 1 {
		return 0, 0, fmt.Errorf("invalid request count %q", countStr)
	}

	window, err := time.ParseDuration(windowStr)
	if err != nil || window <= 0 {
		return 0, 0, fmt.Errorf("invalid window %q", windowStr)
	}

	return maxReqs, window, nil
}
