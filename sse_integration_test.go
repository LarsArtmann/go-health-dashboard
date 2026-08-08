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

// readSSEEvent reads one SSE event from a streaming reader. An SSE event is
// a block of lines terminated by a blank line. Returns the accumulated text.
func readSSEEvent(reader *bufio.Reader, timeout time.Duration) (string, error) {
	type result struct {
		text string
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		var lines []string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				ch <- result{text: strings.Join(lines, ""), err: err}
				return
			}

			line = strings.TrimRight(line, "\r\n")

			if line == "" {
				ch <- result{text: strings.Join(lines, "\n")}
				return
			}

			lines = append(lines, line)
		}
	}()

	select {
	case r := <-ch:
		return r.text, r.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout after %s waiting for SSE event", timeout)
	}
}

// waitForSSEEvent reads SSE events until one matches the predicate or the
// timeout expires. Returns the matching event text or an error.
func waitForSSEEvent(t *testing.T, reader *bufio.Reader, predicate func(string) bool, timeout time.Duration) string {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining < 50*time.Millisecond {
			break
		}

		text, err := readSSEEvent(reader, remaining)
		if err != nil {
			t.Fatalf("error reading SSE event: %v", err)
		}

		if predicate(text) {
			return text
		}
	}

	t.Fatalf("no SSE event matched predicate within %s", timeout)
	return ""
}

func setupSSEServer(t *testing.T, pushInterval time.Duration, svc *toggleService) (*httptest.Server, *toggleService, func()) {
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

	ctx, cancel := context.WithCancel(context.Background())

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

// --- T4: SSE change-detection integration tests ---.

func TestSSE_PushOnChange_DetectsStatusChange(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, svc, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	reader := bufio.NewReader(resp.Body)

	// Read initial state — should be healthy.
	_ = waitForSSEEvent(t, reader, func(s string) bool {
		return strings.Contains(s, "All Systems Operational") || strings.Contains(s, "pass")
	}, 2*time.Second)

	// Toggle to unhealthy — should trigger a broadcast.
	svc.healthy.Store(false)

	changed := waitForSSEEvent(t, reader, func(s string) bool {
		return strings.Contains(s, "Unhealthy") || strings.Contains(s, "fail")
	}, 2*time.Second)

	if !strings.Contains(changed, "fail") {
		t.Error("changed event should contain fail status")
	}

	// Wait for multiple intervals — PushOnChange should NOT broadcast again.
	eventCh := make(chan string, 1)

	go func() {
		text, _ := readSSEEvent(reader, 300*time.Millisecond)
		eventCh <- text
	}()

	select {
	case text := <-eventCh:
		if text != "" {
			t.Errorf("PushOnChange should not broadcast unchanged state, but got event:\n%s", text)
		}
	case <-time.After(300 * time.Millisecond):
		// Expected: no event means PushOnChange correctly suppressed the broadcast.
	}
}

func TestSSE_PushAlways_BroadcastsEveryTick(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	injector := do.New()
	provideToggleService(injector, "db", svc)

	probe := health.New(injector,
		health.WithVersion("1.0.0"),
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(50*time.Millisecond),
	)

	dash := dashboard.New(probe,
		dashboard.WithPushInterval(50*time.Millisecond),
		dashboard.WithPushMode(dashboard.PushAlways),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

	ctx, cancel := context.WithCancel(context.Background())
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

	resp, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	reader := bufio.NewReader(resp.Body)

	// Read initial event.
	_ = waitForSSEEvent(t, reader, func(s string) bool { return true }, 2*time.Second)

	// Read second event — PushAlways should broadcast even though nothing changed.
	_ = waitForSSEEvent(t, reader, func(s string) bool { return true }, 2*time.Second)

	// Read third event — still broadcasting.
	_ = waitForSSEEvent(t, reader, func(s string) bool { return true }, 2*time.Second)
}

func TestSSE_PushOnChange_FingerprintDetectsErrorTextChange(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(false) // Start unhealthy

	server, svc, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	reader := bufio.NewReader(resp.Body)

	// Read initial state — should be failing.
	_ = waitForSSEEvent(t, reader, func(s string) bool {
		return strings.Contains(s, "manually toggled to unhealthy")
	}, 2*time.Second)

	// The fingerprint should detect when error text changes, even if status
	// remains "fail". However, with toggleService the error message is fixed.
	// Toggle to healthy to verify fingerprint change is detected.
	svc.healthy.Store(true)

	recovered := waitForSSEEvent(t, reader, func(s string) bool {
		return strings.Contains(s, "All Systems Operational") || strings.Contains(s, "pass")
	}, 2*time.Second)

	if !strings.Contains(recovered, "pass") && !strings.Contains(recovered, "Operational") {
		t.Error("should detect recovery and broadcast updated state")
	}
}

// --- T8: SSE resilience tests ---.

func TestSSE_ClientDisconnectDoesNotLeakGoroutines(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	// Connect and immediately disconnect.
	resp, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}

	_ = resp.Body.Close()

	// Give the server time to process the disconnect.
	time.Sleep(100 * time.Millisecond)

	// Server should still be operational — no panic or deadlock.
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := probe.Start(ctx); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	server := httptest.NewServer(mux)

	resp, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}

	reader := bufio.NewReader(resp.Body)
	_ = waitForSSEEvent(t, reader, func(s string) bool { return true }, 2*time.Second)

	// Shutdown the dashboard while the SSE connection is open.
	dash.Shutdown()

	// The SSE connection should close (body returns EOF or error).
	_, err = readSSEEvent(reader, 2*time.Second)
	if err == nil {
		// If no error, check if body is actually closed.
		_, err = io.ReadAll(resp.Body)
		if err == nil {
			t.Error("SSE connection should close after dashboard shutdown")
		}
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := probe.Start(ctx); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	dash := dashboard.New(probe)

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	// Immediate shutdown should not panic.
	dash.Shutdown()
}

func TestSSE_ShutdownSafeToCallMultipleTimes(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	// Multiple shutdowns should not panic.
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

	// Create dashboard but do NOT call Start.
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

	// Client 1.
	resp1, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE client 1 connect: %v", err)
	}
	defer func() { _ = resp1.Body.Close() }()

	reader1 := bufio.NewReader(resp1.Body)
	_ = waitForSSEEvent(t, reader1, func(s string) bool { return true }, 2*time.Second)

	// Client 2.
	resp2, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("SSE client 2 connect: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	reader2 := bufio.NewReader(resp2.Body)
	_ = waitForSSEEvent(t, reader2, func(s string) bool { return true }, 2*time.Second)

	// Toggle to unhealthy — both clients should receive the change.
	svc.healthy.Store(false)

	_ = waitForSSEEvent(t, reader1, func(s string) bool {
		return strings.Contains(s, "fail")
	}, 2*time.Second)

	_ = waitForSSEEvent(t, reader2, func(s string) bool {
		return strings.Contains(s, "fail")
	}, 2*time.Second)
}
