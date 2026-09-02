package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

type nopService struct{}

func (nopService) HealthCheck(context.Context) error { return nil }

func TestHealthCheck_WatchdogReportsStalePusher(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideNamed(injector, "svc", func(_ do.Injector) (nopService, error) {
		return nopService{}, nil
	})
	do.MustInvokeNamed[nopService](injector, "svc")

	probe := health.New(injector, health.WithRefreshInterval(20*time.Millisecond))
	dash := New(probe, WithPushInterval(20*time.Millisecond))

	ctx := t.Context()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer dash.Shutdown()

	push := dash.push.Load()

	deadline := time.Now().Add(2 * time.Second)

	for push.lastBroadcast.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("pusher never recorded a broadcast tick")
		}

		time.Sleep(5 * time.Millisecond)
	}

	if err := dash.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck with fresh pusher: want nil, got %v", err)
	}

	push.lastBroadcast.Store(time.Now().Add(-10 * time.Second).UnixNano())

	err := dash.HealthCheck(ctx)
	if !errors.Is(err, ErrPusherStale) {
		t.Fatalf("HealthCheck with stale pusher: want ErrPusherStale, got %v", err)
	}
}

func TestHealthCheck_WatchdogToleratesFirstTick(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideNamed(injector, "svc", func(_ do.Injector) (nopService, error) {
		return nopService{}, nil
	})
	do.MustInvokeNamed[nopService](injector, "svc")

	probe := health.New(injector, health.WithRefreshInterval(50*time.Millisecond))
	dash := New(probe, WithPushInterval(50*time.Millisecond))

	ctx := t.Context()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer dash.Shutdown()

	// Before the first tick lands, the watchdog must not report staleness:
	// there is no evidence yet, and a just-started dashboard would flake.
	if err := dash.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck immediately after Start: want nil, got %v", err)
	}
}
