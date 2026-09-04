package dashboard_test

import (
	"bufio"
	"context"
	"errors"
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

	return errors.New("manually toggled to unhealthy")
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
	dashOpts ...dashboard.Option,
) (*httptest.Server, *dashboard.Dashboard, func()) {
	t.Helper()

	injector := do.New()
	provideToggleService(injector, "db", svc)

	probe := health.New(injector,
		health.WithVersion("1.0.0"),
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(pushInterval),
	)

	dash := dashboard.New(probe,
		append([]dashboard.Option{dashboard.WithPushInterval(pushInterval)}, dashOpts...)...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

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

	return server, dash, cleanup
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

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
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

	server, _, cleanup := setupSSEServer(
		t,
		50*time.Millisecond,
		svc,
		dashboard.WithPushMode(dashboard.PushAlways),
	)
	defer cleanup()

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

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	// Initial state is unhealthy.
	stream.waitFor(t, isUnhealthyEvent, 2*time.Second)

	// Recover — should trigger a broadcast.
	svc.healthy.Store(true)
	stream.waitFor(t, isHealthyEvent, 2*time.Second)
}

// --- Connection limit tests ---.

// WithMaxSSEConnections(0) is the default: unlimited. Several clients must
// all be admitted (HTTP 200 + a first patch), never rejected with 503.
func TestWithMaxSSEConnections_ZeroAllowsUnlimited(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(
		t,
		50*time.Millisecond,
		svc,
		dashboard.WithMaxSSEConnections(0),
	)
	defer cleanup()

	const clients = 3
	resps := make([]*http.Response, clients)
	streams := make([]*sseStream, clients)

	for i := range resps {
		resp, stream := connectSSE(t, server)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("client %d: want 200, got %d", i, resp.StatusCode)
		}

		resps[i] = resp
		streams[i] = stream
	}

	defer func() {
		for _, resp := range resps {
			_ = resp.Body.Close()
		}
	}()

	// Every client receives its initial patch.
	for _, stream := range streams {
		stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
	}

	if count := dash.SubscriberCount(); count != clients {
		t.Fatalf("SubscriberCount with %d clients: want %d, got %d", clients, clients, count)
	}
}

// A dashboard whose probe was never started still serves SSE: the zero-value
// response renders as an unknown/degraded status instead of panicking or 500.
func TestSSE_ProbeNotStarted_ServesDegradedRender(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideToggleService(injector, "db", &toggleService{})

	// Intentionally NOT calling probe.Start.
	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(50*time.Millisecond),
	)
	defer probe.Shutdown()

	dash := dashboard.New(probe, dashboard.WithPushInterval(50*time.Millisecond))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	defer dash.Shutdown()

	server := httptest.NewServer(mux)
	defer server.Close()

	// HTML path renders degraded instead of erroring.
	healthResp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health with unstarted probe: %v", err)
	}

	body, _ := io.ReadAll(healthResp.Body)
	_ = healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		t.Errorf("GET /health with unstarted probe: want 200, got %d", healthResp.StatusCode)
	}

	if !strings.Contains(string(body), "Updated") {
		t.Error("degraded HTML render missing the Updated stamp")
	}

	// SSE path still streams an initial (degraded) patch.
	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
}

// SSE patches replace the health region via inner HTML — they must never
// carry inline style attributes (CSP-clean output invariant).
func TestSSE_PatchContentHasNoInlineStyles(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	// PushAlways so the stream carries multiple patches to inspect, not
	// just the initial one (PushOnChange stays silent without changes).
	server, _, cleanup := setupSSEServer(
		t,
		50*time.Millisecond,
		svc,
		dashboard.WithPushMode(dashboard.PushAlways),
	)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	for range 3 {
		evt := stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
		if strings.Contains(evt, "style=") {
			t.Errorf("SSE patch contains an inline style attribute:\n%.500s", evt)
		}
	}
}

// --- T8: SSE resilience tests ---.

func TestSSE_ClientDisconnectDoesNotLeakGoroutines(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, _ := connectSSE(t, server)
	_ = resp.Body.Close()

	// Event-driven: wait until the server has actually released the
	// connection instead of guessing with a fixed sleep.
	deadline := time.Now().Add(3 * time.Second)
	for dash.SubscriberCount() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("SSE connection was not released after client disconnect")
		}
		time.Sleep(10 * time.Millisecond)
	}

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
	dash.RegisterRoutes(mux)

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

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
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

func TestSSE_ConnectionLimitRejectsExcessClients(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(
		t,
		50*time.Millisecond,
		svc,
		dashboard.WithMaxSSEConnections(1),
	)
	defer cleanup()

	// First client connects successfully.
	resp1, stream1 := connectSSE(t, server)
	defer func() { _ = resp1.Body.Close() }()
	stream1.waitFor(t, func(string) bool { return true }, 2*time.Second)

	// Second client should be rejected with 503.
	resp2, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("second connect attempt: %v", err)
	}

	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("excess client: want 503, got %d", resp2.StatusCode)
	}
}

// --- SubscriberCount tests ---.

func TestSSE_SubscriberCount_TracksConnections(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	if count := dash.SubscriberCount(); count != 0 {
		t.Fatalf("initial SubscriberCount: want 0, got %d", count)
	}

	resp1, stream1 := connectSSE(t, server)
	defer func() { _ = resp1.Body.Close() }()
	stream1.waitFor(t, func(string) bool { return true }, 2*time.Second)

	if count := dash.SubscriberCount(); count != 1 {
		t.Fatalf("after 1 client: want 1, got %d", count)
	}

	resp2, stream2 := connectSSE(t, server)
	defer func() { _ = resp2.Body.Close() }()
	stream2.waitFor(t, func(string) bool { return true }, 2*time.Second)

	if count := dash.SubscriberCount(); count != 2 {
		t.Fatalf("after 2 clients: want 2, got %d", count)
	}

	_ = resp1.Body.Close()

	// Event-driven: poll the counter instead of sleeping a fixed interval.
	deadline := time.Now().Add(3 * time.Second)
	for dash.SubscriberCount() != 1 {
		if time.Now().After(deadline) {
			t.Fatalf("after disconnecting 1: want 1, got %d", dash.SubscriberCount())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// --- Heartbeat tests ---.

func TestSSE_HeartbeatInterval_SendsKeepalive(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(
		t,
		50*time.Millisecond,
		svc,
		dashboard.WithHeartbeatInterval(100*time.Millisecond),
	)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	stream.waitFor(t, func(string) bool { return true }, 2*time.Second)

	stream.waitFor(t, func(s string) bool {
		return strings.Contains(s, ": heartbeat")
	}, 3*time.Second)
}

// --- Retry interval tests ---.

func TestWithRetryInterval_EventsCarryRetry(t *testing.T) {
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
		dashboard.WithRetryInterval(2*time.Second),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

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

	evt := stream.waitFor(t, func(string) bool { return true }, 2*time.Second)

	if !strings.Contains(evt, "retry: 2000") {
		t.Errorf("SSE event should contain 'retry: 2000', got:\n%s", evt)
	}
}

func TestWithRetryInterval_DefaultOmitsRetryField(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	// No WithRetryInterval — should default to zero (browser default).
	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer func() { _ = resp.Body.Close() }()

	evt := stream.waitFor(t, func(string) bool { return true }, 2*time.Second)

	if strings.Contains(evt, "retry:") {
		t.Errorf(
			"SSE event should NOT contain 'retry:' when WithRetryInterval is zero, got:\n%s",
			evt,
		)
	}
}

// --- Reconnection tests ---.

func TestSSE_Reconnect_ReceivesCurrentState(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp1, stream1 := connectSSE(t, server)
	defer func() { _ = resp1.Body.Close() }()
	stream1.waitFor(t, isHealthyEvent, 2*time.Second)

	// Toggle to unhealthy and keep stream1 open until it confirms the probe
	// has actually refreshed. This eliminates the need for a fixed sleep.
	svc.healthy.Store(false)
	stream1.waitFor(t, isUnhealthyEvent, 2*time.Second)

	// Reconnect — the new connection's initial patch must reflect current state.
	resp2, stream2 := connectSSE(t, server)
	defer func() { _ = resp2.Body.Close() }()

	stream2.waitFor(t, isUnhealthyEvent, 2*time.Second)
}

// --- SSE CSP nonce flow tests ---.

func TestSSE_PatchesContainNoInlineScripts(t *testing.T) {
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
		dashboard.WithNonce("test-nonce-abc"),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

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

	for range 3 {
		evt := stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
		if strings.Contains(strings.ToLower(evt), "<script") {
			t.Errorf(
				"SSE patch must not contain <script> tags (CSP-safe inner-HTML), got:\n%s",
				evt,
			)
		}
	}
}
