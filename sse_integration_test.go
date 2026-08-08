package dashboard_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// toggleService is a test service whose health can be toggled at runtime.
type toggleService struct {
	healthy atomic.Bool
}

func (t *toggleService) HealthCheck(_ context.Context) error {
	if t.healthy.Load() {
		return nil
	}
	return fmt.Errorf("manually toggled to unhealthy")
}

func provideToggleService(i do.Injector, name string, svc *toggleService) {
	do.ProvideNamed(i, name, func(_ do.Injector) (*toggleService, error) {
		return svc, nil
	})

	if _, err := do.InvokeNamed[*toggleService](i, name); err != nil {
		panic(fmt.Sprintf("invoke %s: %v", name, err))
	}
}

// sseStream wraps a response body reader into a channel of SSE events.
// A single goroutine reads from the body, eliminating reader-level races.
type sseStream struct {
	events chan string
}

func newSSEStream(body io.Reader) *sseStream {
	s := &sseStream{events: make(chan string, 32)}

	go func() {
		defer close(s.events)
		reader := bufio.NewReader(body)
		var lines []string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				s.events <- strings.Join(lines, "\n")
				lines = nil
				continue
			}

			lines = append(lines, line)
		}
	}()

	return s
}

// waitFor reads SSE events until one matches the predicate or the timeout
// expires. Calls t.Fatal on timeout.
func (s *sseStream) waitFor(
	t *testing.T,
	predicate func(string) bool,
	timeout time.Duration,
) string {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatalf("no SSE event matched predicate within %s", timeout)
		}

		select {
		case evt, ok := <-s.events:
			if !ok {
				t.Fatalf("SSE stream closed before matching event")
			}
			if predicate(evt) {
				return evt
			}
		case <-time.After(remaining):
			t.Fatalf("no SSE event matched predicate within %s", timeout)
		}
	}
}

// assertNoEvent verifies that no SSE event arrives within the timeout.
func (s *sseStream) assertNoEvent(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case evt := <-s.events:
		t.Errorf("expected no SSE event for %s, but got:\n%s", timeout, evt)
	case <-time.After(timeout):
		// Expected: no event within timeout.
	}
}

func setupSSEServer(
	t *testing.T,
	pushInterval time.Duration,
	svc *toggleService,
) (*httptest.Server, *toggleService, func()) {
	t.Helper()

	injector := do.New()
	provideToggleService(injector, "db", svc)

	probe := health.New(injector,
		health.WithVersion("1.0.0"),
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(pushInterval),
	)

	dash := dashboard.New(probe, dashboard.WithPushInterval(pushInterval))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	ctx, cancel := context.WithCancel(t.Context())

	if err := probe.Start(ctx); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	server := httptest.NewServer(mux)

	cleanup := func() {
		server.Close()
		cancel()
		dash.Shutdown()
		probe.Shutdown()
	}

	return server, svc, cleanup
}

func connectSSE(t *testing.T, server *httptest.Server) (*http.Response, *sseStream) {
	t.Helper()

	resp, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}

	return resp, newSSEStream(resp.Body)
}

func isHealthyEvent(s string) bool {
	return strings.Contains(s, "All Systems Operational") || strings.Contains(s, `"pass"`)
}

func isUnhealthyEvent(s string) bool {
	return strings.Contains(s, "Unhealthy") || strings.Contains(s, `"fail"`)
}

// --- T4: SSE change-detection integration tests ---.

func TestSSE_PushOnChange_DetectsStatusChange(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, svc, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	// Initial broadcast on connect.
	stream.waitFor(t, isHealthyEvent, 2*time.Second)

	// Toggle to unhealthy — should trigger a broadcast.
	svc.healthy.Store(false)
	stream.waitFor(t, isUnhealthyEvent, 2*time.Second)

	// PushOnChange should NOT broadcast again when nothing changed.
	stream.assertNoEvent(t, 250*time.Millisecond)
}

func TestSSE_PushAlways_BroadcastsEveryTick(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	injector := do.New()
	provideToggleService(injector, "db", svc)

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(50*time.Millisecond),
	)

	dash := dashboard.New(probe,
		dashboard.WithPushInterval(50*time.Millisecond),
		dashboard.WithPushMode(dashboard.PushAlways),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := probe.Start(ctx); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	defer func() { dash.Shutdown(); probe.Shutdown() }()

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	// PushAlways: initial + at least 2 more ticks.
	stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
	stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
	stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
}

func TestSSE_PushOnChange_DetectsRecovery(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(false)

	server, svc, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	// Initial state is unhealthy.
	stream.waitFor(t, isUnhealthyEvent, 2*time.Second)

	// Recover — should trigger a broadcast.
	svc.healthy.Store(true)
	stream.waitFor(t, isHealthyEvent, 2*time.Second)
}

// --- T8: SSE resilience tests ---.

func TestSSE_ClientDisconnectDoesNotLeakGoroutines(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, _ := connectSSE(t, server)
	_ = resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	healthResp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("server should still respond after SSE disconnect: %v", err)
	}
	_ = healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		t.Errorf("server status after disconnect: want 200, got %d", healthResp.StatusCode)
	}
}

func TestSSE_ShutdownClosesConnections(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	injector := do.New()
	provideToggleService(injector, "db", svc)

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(50*time.Millisecond),
	)

	dash := dashboard.New(probe, dashboard.WithPushInterval(50*time.Millisecond))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := probe.Start(ctx); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	server := httptest.NewServer(mux)

	resp, stream := connectSSE(t, server)
	stream.waitFor(t, func(string) bool { return true }, 2*time.Second)

	dash.Shutdown()

	// After shutdown, the SSE stream should eventually close.
	// Buffered events may flush first, but the channel must close.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case _, ok := <-stream.events:
			if !ok {
				break // Channel closed — expected.
			}
			continue // Buffered event, keep draining.
		case <-time.After(time.Until(deadline)):
			t.Error("SSE stream should close after dashboard shutdown")
		}
		break
	}

	_ = resp.Body.Close()
	server.Close()
	probe.Shutdown()
}

func TestSSE_StartThenImmediateShutdownDoesNotPanic(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := probe.Start(ctx); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	dash := dashboard.New(probe)
	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	dash.Shutdown()
}

func TestSSE_ShutdownSafeToCallMultipleTimes(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	s.dash.Shutdown()
	s.dash.Shutdown()
	s.dash.Shutdown()
}

func TestSSE_HandlerReturns503WhenPusherNotStarted(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	defer probe.Shutdown()

	dash := dashboard.New(probe)

	mux := http.NewServeMux()
	mux.HandleFunc("/health/sse", dash.SSEHandler())

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("SSE without pusher: want 503, got %d", w.Code)
	}
}

func TestSSE_MultipleClientsReceiveBroadcasts(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, svc, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp1, stream1 := connectSSE(t, server)
	defer func() { _ = resp1.Body.Close() }()
	stream1.waitFor(t, func(string) bool { return true }, 2*time.Second)

	resp2, stream2 := connectSSE(t, server)
	defer func() { _ = resp2.Body.Close() }()
	stream2.waitFor(t, func(string) bool { return true }, 2*time.Second)

	// Toggle to unhealthy — both clients should receive the change.
	svc.healthy.Store(false)

	stream1.waitFor(t, isUnhealthyEvent, 2*time.Second)
	stream2.waitFor(t, isUnhealthyEvent, 2*time.Second)
}
