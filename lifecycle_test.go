package dashboard_test

import (
	"context"
	"errors"
	"testing"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

func TestHealthCheck_NotStarted_ReturnsError(t *testing.T) {
	t.Parallel()

	injector := do.New()
	defer injector.Shutdown()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector)
	dash := dashboard.New(probe)

	if err := dash.HealthCheck(t.Context()); err == nil {
		t.Fatal("expected error when pusher is not started, got nil")
	}
}

func TestHealthCheck_Started_ReturnsNil(t *testing.T) {
	t.Parallel()

	setup := setupDashboard(t)
	defer setup.cleanup()

	if err := setup.dash.HealthCheck(t.Context()); err != nil {
		t.Fatalf("expected nil after Start, got %v", err)
	}
}

func TestHealthCheck_AfterShutdown_ReturnsError(t *testing.T) {
	t.Parallel()

	setup := setupDashboard(t)
	setup.cleanup()

	if err := setup.dash.HealthCheck(t.Context()); err == nil {
		t.Fatal("expected error after Shutdown, got nil")
	}
}

func TestRegister_HealthCheckViaContainer(t *testing.T) {
	t.Parallel()

	injector := do.New()
	defer injector.Shutdown()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector)
	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	defer probe.Shutdown()

	dash := dashboard.Register(injector, probe, dashboard.WithTitle("Test"))
	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	if err := do.HealthCheck[*dashboard.Dashboard](injector); err != nil {
		t.Fatalf("do.HealthCheck returned error after Start: %v", err)
	}
}

func TestRegister_HealthCheckBeforeStart_ReturnsError(t *testing.T) {
	t.Parallel()

	injector := do.New()
	defer injector.Shutdown()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector)
	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	defer probe.Shutdown()

	dashboard.Register(injector, probe)
	// Intentionally not calling dash.Start

	if err := do.HealthCheck[*dashboard.Dashboard](injector); err == nil {
		t.Fatal("expected do.HealthCheck to fail when pusher is not started")
	}
}

func TestRegister_ShutdownCascadeStopsPusher(t *testing.T) {
	t.Parallel()

	injector := do.New()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector)
	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	dash := dashboard.Register(injector, probe)
	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	// do.Shutdown should cascade to Dashboard.Shutdown.
	report := injector.Shutdown()
	if !report.Succeed {
		t.Fatalf("injector.Shutdown failed: %v", report)
	}

	// After container shutdown, HealthCheck should fail.
	if err := dash.HealthCheck(t.Context()); err == nil {
		t.Fatal("expected HealthCheck to fail after do.Shutdown cascade")
	}
}

func TestShutdown_IsIdempotent(t *testing.T) {
	t.Parallel()

	setup := setupDashboard(t)
	defer setup.cleanup()

	// Multiple Shutdown calls should not panic.
	setup.dash.Shutdown()
	setup.dash.Shutdown()
}

func TestHealthCheckWithContext_InterfaceSatisfied(t *testing.T) {
	t.Parallel()

	var _ do.HealthcheckerWithContext = (*dashboard.Dashboard)(nil)
	var _ do.Shutdowner = (*dashboard.Dashboard)(nil)
}

func TestHealthCheck_WithCancelledContext(t *testing.T) {
	t.Parallel()

	setup := setupDashboard(t)
	defer setup.cleanup()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// HealthCheck should still succeed — it's a fast atomic check,
	// not dependent on context cancellation.
	if err := setup.dash.HealthCheck(ctx); err != nil {
		t.Fatalf("expected nil even with cancelled context, got %v", err)
	}
}

func TestRegister_ReturnsSameInstance(t *testing.T) {
	t.Parallel()

	injector := do.New()
	defer injector.Shutdown()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector)
	dash := dashboard.Register(injector, probe)

	resolved, err := do.Invoke[*dashboard.Dashboard](injector)
	if err != nil {
		t.Fatalf("do.Invoke: %v", err)
	}

	if resolved != dash {
		t.Fatal("Register should return the same instance stored in the container")
	}
}

// Ensure errors.Is works on the HealthCheck error for consumers who want to
// distinguish dashboard-push-down from other health check failures.
func TestHealthCheck_ErrorIsDetectable(t *testing.T) {
	t.Parallel()

	injector := do.New()
	defer injector.Shutdown()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector)
	dash := dashboard.New(probe)

	err := dash.HealthCheck(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, dashboard.ErrPusherNotActive) {
		t.Fatalf("errors.Is(err, ErrPusherNotActive) = false, want true (err: %v)", err)
	}
}

func TestHealthCheck_NotStarted_ReturnsNotStartedSentinel(t *testing.T) {
	t.Parallel()

	injector := do.New()
	defer injector.Shutdown()

	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	dash := dashboard.New(health.New(injector))

	err := dash.HealthCheck(t.Context())
	if err == nil {
		t.Fatal("expected error before Start, got nil")
	}

	if !errors.Is(err, dashboard.ErrPusherNotStarted) {
		t.Errorf("errors.Is(err, ErrPusherNotStarted) = false, want true (err: %v)", err)
	}

	if errors.Is(err, dashboard.ErrPusherShutDown) {
		t.Errorf("not-started error must not match ErrPusherShutDown (err: %v)", err)
	}
}

func TestHealthCheck_AfterShutdown_ReturnsShutDownSentinel(t *testing.T) {
	t.Parallel()

	setup := setupDashboard(t)
	setup.cleanup()

	err := setup.dash.HealthCheck(t.Context())
	if err == nil {
		t.Fatal("expected error after Shutdown, got nil")
	}

	if !errors.Is(err, dashboard.ErrPusherShutDown) {
		t.Errorf("errors.Is(err, ErrPusherShutDown) = false, want true (err: %v)", err)
	}

	if errors.Is(err, dashboard.ErrPusherNotStarted) {
		t.Errorf("shut-down error must not match ErrPusherNotStarted (err: %v)", err)
	}
}
