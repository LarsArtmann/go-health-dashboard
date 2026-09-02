package dashboard_test

import (
	"net/http"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// --- Rate limiting (M12) ---

func TestRateLimit_ExceedsBudget(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithRateLimit(2, time.Second))
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
		t.Fatalf("request 1: want 200, got %d", w.Code)
	}

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
		t.Fatalf("request 2: want 200, got %d", w.Code)
	}

	w := doRequest(t, s.mux, "/health")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 3: want 429, got %d", w.Code)
	}

	if w.Header().Get("Retry-After") == "" {
		t.Error("429 response missing Retry-After header")
	}
}

func TestRateLimit_ProbesExempt(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithRateLimit(1, time.Hour))
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
		t.Fatalf("first dashboard request: want 200, got %d", w.Code)
	}

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("second dashboard request: want 429, got %d", w.Code)
	}

	for _, route := range []string{"/healthz", "/readyz", "/startupz"} {
		if w := doRequest(t, s.mux, route); w.Code != http.StatusOK {
			t.Errorf("probe %s: want 200 (rate limit must not apply), got %d", route, w.Code)
		}
	}
}

func TestRateLimit_RefillsOverTime(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithRateLimit(1, 150*time.Millisecond))
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
		t.Fatalf("request 1: want 200, got %d", w.Code)
	}

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 2: want 429, got %d", w.Code)
	}

	time.Sleep(200 * time.Millisecond)

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
		t.Fatalf("request 3 after refill: want 200, got %d", w.Code)
	}
}

func TestRateLimit_DisabledByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	for range 12 {
		if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
			t.Fatalf("rate limiting must be opt-in; got %d", w.Code)
		}
	}
}

// --- Shutdown drain (M9) ---

func TestShutdown_DrainWaitsForSubscribers(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(t, 50*time.Millisecond, svc,
		dashboard.WithShutdownDrain(400*time.Millisecond),
	)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer resp.Body.Close()

	stream.waitFor(t, isHealthyEvent, 5*time.Second)

	start := time.Now()

	dash.Shutdown()

	if elapsed := time.Since(start); elapsed < 300*time.Millisecond {
		t.Errorf(
			"Shutdown returned after %s; want it to drain ~400ms while a client is connected",
			elapsed,
		)
	}

	// After shutdown, new SSE connections are rejected immediately.
	next, err := http.Get(server.URL + "/health/sse")
	if err != nil {
		t.Fatalf("post-shutdown SSE connect: %v", err)
	}
	defer next.Body.Close()

	if next.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("post-shutdown SSE connect: want 503, got %d", next.StatusCode)
	}
}

func TestShutdown_WithoutDrainClosesImmediately(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer resp.Body.Close()

	stream.waitFor(t, isHealthyEvent, 5*time.Second)

	start := time.Now()

	dash.Shutdown()

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("Shutdown without drain took %s; want immediate return", elapsed)
	}
}

// --- Connection lifetime (M10) ---

func TestMaxConnectionLifetime_ClosesStream(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc,
		dashboard.WithMaxConnectionLifetime(300*time.Millisecond),
	)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer resp.Body.Close()

	stream.waitFor(t, isHealthyEvent, 5*time.Second)

	deadline := time.After(5 * time.Second)

	for {
		select {
		case _, ok := <-stream.events:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("SSE stream never closed after max connection lifetime")
		}
	}
}

func TestMaxConnectionLifetime_ZeroKeepsStreamOpen(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, _, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	resp, stream := connectSSE(t, server)
	defer resp.Body.Close()

	stream.waitFor(t, isHealthyEvent, 5*time.Second)

	// Receiving any event (or silence) both prove the stream stays open;
	// only a closed channel is a failure.
	select {
	case _, ok := <-stream.events:
		if !ok {
			t.Error("stream closed without lifetime cap configured")
		}
	case <-time.After(600 * time.Millisecond):
		// Still open, just quiet — expected with PushOnChange.
	}
}
