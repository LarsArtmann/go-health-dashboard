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

func TestHealthCheck_WatchdogFloorProtectsFastPushers(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideNamed(injector, "svc", func(_ do.Injector) (nopService, error) {
		return nopService{}, nil
	})
	do.MustInvokeNamed[nopService](injector, "svc")

	probe := health.New(injector, health.WithRefreshInterval(20*time.Millisecond))
	dash := New(probe, WithPushInterval(20*time.Millisecond))

	// Hand-wire a never-started pusher so lastBroadcast stays exactly where
	// the test puts it: a running push loop would overwrite a backdated
	// stamp within one tick and could mask a missing floor under load.
	push := newPusher(dash)
	push.lastBroadcast.Store(time.Now().Add(-700 * time.Millisecond).UnixNano())
	dash.push.Store(push)

	ctx := t.Context()

	// 700ms is far beyond 3×20ms but inside pusherStaleFloor (1s): the
	// published v0.11.0 behavior keeps fast probes healthy here. Deleting
	// the floor re-arms the 60ms bound and fails this assertion.
	if err := dash.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck 700ms after last broadcast at 20ms interval: want nil (pusherStaleFloor must apply), got %v", err)
	}

	// The floor bounds detection, it does not disable it: well past the
	// floor, a silent fast pusher still reports stale.
	push.lastBroadcast.Store(time.Now().Add(-2 * time.Second).UnixNano())

	if err := dash.HealthCheck(ctx); !errors.Is(err, ErrPusherStale) {
		t.Fatalf("HealthCheck 2s after last broadcast at 20ms interval: want ErrPusherStale, got %v", err)
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
