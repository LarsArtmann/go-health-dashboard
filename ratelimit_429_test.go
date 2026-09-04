package dashboard_test

import (
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// TestRateLimit_429JSONBody verifies JSON-negotiating clients get a
// parseable 429 document with retry_after, while HTML clients keep the
// plain-text body.
func TestRateLimit_429JSONBody(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithRateLimit(1, 2*time.Second))
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health"); w.Code != http.StatusOK {
		t.Fatalf("first request within budget: want 200, got %d", w.Code)
	}

	// Exhausted: HTML client keeps the text body.
	wHTML := doRequest(t, s.mux, "/health")
	if wHTML.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: want 429, got %d", wHTML.Code)
	}

	if ct := wHTML.Header().Get("Content-Type"); strings.HasPrefix(ct, "application/json") {
		t.Errorf("HTML client should get text/plain 429, got %q", ct)
	}

	if ra := wHTML.Header().Get("Retry-After"); ra == "" {
		t.Error("Retry-After header missing on 429")
	}

	// JSON client gets the document.
	wJSON := doRequestWithAccept(t, s.mux, "/health", "application/json")
	if wJSON.Code != http.StatusTooManyRequests {
		t.Fatalf("JSON request: want 429, got %d", wJSON.Code)
	}

	if ct := wJSON.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("JSON client Content-Type: want application/json, got %q", ct)
	}

	var body struct {
		Error      string `json:"error"`
		RetryAfter int    `json:"retry_after"`
	}
	if err := json.Unmarshal(wJSON.Body.Bytes(), &body); err != nil {
		t.Fatalf("429 body is not valid JSON: %v\n%s", err, wJSON.Body.String())
	}

	if body.Error == "" {
		t.Error("429 JSON error field empty")
	}

	if body.RetryAfter <= 0 {
		t.Errorf("429 retry_after: want > 0, got %d", body.RetryAfter)
	}
}

// TestSSE_Drain503CarriesRetryAfter verifies that after Shutdown (with a
// configured drain window), new SSE connections receive 503 with a
// Retry-After hinting the drain duration, so clients reconnect after the
// drain completes instead of hammering a draining server.
func TestSSE_Drain503CarriesRetryAfter(t *testing.T) {
	t.Parallel()

	const drain = 2 * time.Second

	s := setupDashboard(t, dashboard.WithShutdownDrain(drain))
	defer s.cleanup()

	s.dash.Shutdown()

	w := doRequest(t, s.mux, "/health/sse")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("SSE after shutdown: want 503, got %d", w.Code)
	}

	retryAfterHeader := w.Header().Get("Retry-After")
	if retryAfterHeader == "" {
		t.Fatal("Retry-After missing on drain-window 503")
	}

	seconds, err := strconv.Atoi(retryAfterHeader)
	if err != nil {
		t.Fatalf("Retry-After not integer seconds: %q", retryAfterHeader)
	}

	if seconds < 1 || seconds > int(drain.Seconds()) {
		t.Errorf("Retry-After %d outside drain window [1, %d]", seconds, int(drain.Seconds()))
	}
}
