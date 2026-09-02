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
//	PORT=8080                    listen address
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"strings"
	"os/signal"
	"syscall"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	injector := do.New()
	defer func() { _ = injector.Shutdown() }()

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
	defer probe.Shutdown()

	// Assemble the option set from environment toggles so every feature can
	// be demonstrated without code changes.
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
		log.Printf("auth: bearer token required on dashboard routes (DEMO_AUTH set, %d chars)", len(token))
	}

	if spec := os.Getenv("DEMO_RATELIMIT"); spec != "" {
		maxReqs, window, err := parseRateLimit(spec)
		if err != nil {
			log.Fatalf("DEMO_RATELIMIT: %v", err)
		}

		opts = append(opts, dashboard.WithRateLimit(maxReqs, window))
		log.Printf("rate limit: %d requests per %s on dashboard routes", maxReqs, window)
	}

	// Register the dashboard in the injector so it participates in
	// do.Shutdown and do.HealthCheck cascades automatically.
	dash := dashboard.Register(injector, probe, opts...)

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
		log.Fatalf("%s: invalid duration %q", key, spec)
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
	if err != nil || maxRequestsInvalid(maxReqs) {
		return 0, 0, fmt.Errorf("invalid request count %q", countStr)
	}

	window, err := time.ParseDuration(windowStr)
	if err != nil || window <= 0 {
		return 0, 0, fmt.Errorf("invalid window %q", windowStr)
	}

	return maxReqs, window, nil
}

func maxRequestsInvalid(n int) bool { return n < 1 }
