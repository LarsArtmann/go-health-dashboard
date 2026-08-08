// Command example demonstrates the go-health-dashboard with mock services.
//
// Run with:
//
//	GOEXPERIMENT=jsonv2 go run ./example
//
// Then open http://localhost:8080/health in a browser to see the live dashboard.
// Kubelet-style JSON is available at http://localhost:8080/readyz.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	injector := do.New()

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

	dash := dashboard.New(probe,
		dashboard.WithTitle("Demo Service"),
	)

	if err := dash.Start(ctx); err != nil {
		log.Fatalf("dash.Start: %v", err)
	}
	defer dash.Shutdown()

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	addr := ":8080"
	log.Printf("dashboard: http://localhost%s/health", addr)
	log.Printf("readiness: http://localhost%s/readyz", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
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
