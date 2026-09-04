package dashboard_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	aggregate "github.com/larsartmann/go-health/aggregate"
	"github.com/samber/do/v2"
)

// --- Stub prober ---.

// stubProber is a hand-rolled dashboard.Prober used to prove that the
// dashboard renders any source implementing the interface, not just
// *health.Probe. The response is swappable so tests can trigger transitions.
type stubProber struct {
	resp atomic.Pointer[health.Response]
}

func newStubProber(resp health.Response) *stubProber {
	p := &stubProber{}
	p.resp.Store(&resp)

	return p
}

func (s *stubProber) set(resp health.Response) { s.resp.Store(&resp) }

func (s *stubProber) CachedResponse() health.Response { return *s.resp.Load() }

func (s *stubProber) RefreshInterval() time.Duration { return 20 * time.Millisecond }

func (s *stubProber) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (s *stubProber) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (s *stubProber) StartupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

// webhookCapture collects webhook deliveries from an httptest server. The
// body is read inside the handler because the request is closed when the
// handler returns.
type webhookCapture struct {
	ch chan capturedRequest
}

type capturedRequest struct {
	headers http.Header
	body    []byte
}

func newWebhookCapture() *webhookCapture {
	return &webhookCapture{ch: make(chan capturedRequest, 32)}
}

func (c *webhookCapture) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.ch <- capturedRequest{headers: r.Header.Clone(), body: body}
		w.WriteHeader(http.StatusOK)
	}
}

// next awaits the next delivery or fails the test after a generous deadline.
func (c *webhookCapture) next(t *testing.T) capturedRequest {
	t.Helper()

	select {
	case r := <-c.ch:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")

		return capturedRequest{}
	}
}

// decodeWebhook unmarshals a delivery body into the payload shape.
func decodeWebhook(t *testing.T, captured capturedRequest) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(captured.body, &payload); err != nil {
		t.Fatalf("decode webhook payload: %v", err)
	}

	return payload
}

func passResponse() health.Response {
	return health.Response{
		Status: health.StatusPass,
		Checks: map[string]health.Check{
			"api/database": {Status: health.StatusPass},
		},
	}
}

// --- Prober interface ---.

func TestNew_AcceptsCustomProber(t *testing.T) {
	t.Parallel()

	dash := dashboard.New(newStubProber(passResponse()))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health status = %d, want 200", resp.StatusCode)
	}
}

// --- Webhook behavior ---.

func TestWebhook_AnnouncesInitialStateThenTransitions(t *testing.T) {
	t.Parallel()

	capture := newWebhookCapture()
	srv := httptest.NewServer(capture.handler())
	defer srv.Close()

	prober := newStubProber(passResponse())

	dash := dashboard.New(prober,
		dashboard.WithWebhook(srv.URL),
		dashboard.WithWebhookHeaders(map[string]string{"Authorization": "Bearer test-token"}),
		dashboard.WithPushInterval(20*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	firstReq := capture.next(t)
	first := decodeWebhook(t, firstReq)
	if first["status"] != string(health.StatusPass) {
		t.Fatalf("initial webhook status = %v, want pass", first["status"])
	}

	if got := firstReq.headers.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("Authorization header = %q, want bearer token passthrough", got)
	}

	if got := firstReq.headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	// Trigger a transition: pass → warn.
	prober.set(health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{
			"api/database": {Status: health.StatusWarn, Error: "connection refused"},
		},
	})

	second := decodeWebhook(t, capture.next(t))
	if second["status"] != string(health.StatusWarn) {
		t.Fatalf("transition webhook status = %v, want warn", second["status"])
	}
	checks, ok := second["checks"].(map[string]any)
	if !ok {
		t.Fatalf("webhook checks missing or wrong shape: %v", second["checks"])
	}

	check, ok := checks["api/database"].(map[string]any)
	if !ok {
		t.Fatalf("namespaced check missing: %v", checks)
	}

	if check["error"] != "connection refused" {
		t.Fatalf("check error = %v, want detail in non-public mode", check["error"])
	}

	if _, ok := second["changed_at"].(string); !ok {
		t.Fatalf("changed_at missing or not a string: %v", second["changed_at"])
	}
}

func TestWebhook_SilentWhenNothingChanges(t *testing.T) {
	t.Parallel()

	capture := newWebhookCapture()
	srv := httptest.NewServer(capture.handler())
	defer srv.Close()

	dash := dashboard.New(newStubProber(passResponse()),
		dashboard.WithWebhook(srv.URL),
		dashboard.WithPushInterval(20*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	// Exactly one delivery: the initial announcement.
	capture.next(t)

	// Many push ticks later, no further deliveries may arrive.
	select {
	case r := <-capture.ch:
		t.Fatalf("unexpected extra webhook delivery: %s", r.body)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestWebhook_PublicModeMasksCheckNamesAndErrors(t *testing.T) {
	t.Parallel()

	capture := newWebhookCapture()
	srv := httptest.NewServer(capture.handler())
	defer srv.Close()

	prober := newStubProber(passResponse())

	dash := dashboard.New(prober,
		dashboard.WithWebhook(srv.URL),
		dashboard.WithPublicMode(),
		dashboard.WithPushInterval(20*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	decodeWebhook(t, capture.next(t)) // initial announcement

	prober.set(health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{
			"api/database": {Status: health.StatusWarn, Error: "connection refused"},
		},
	})

	payload := decodeWebhook(t, capture.next(t))

	checks, ok := payload["checks"].(map[string]any)
	if !ok {
		t.Fatalf("webhook checks missing or wrong shape: %v", payload["checks"])
	}

	if _, ok := checks["api/database"]; ok {
		t.Fatalf("public mode leaked internal check name; got %v", checks)
	}

	masked, ok := checks["check-1"].(map[string]any)
	if !ok {
		t.Fatalf("masked check key missing; got %v", checks)
	}

	if errText, ok := masked["error"]; ok && errText != "" {
		t.Fatalf("public mode leaked error detail: %v", errText)
	}
}

// --- Aggregate end-to-end ---.

func TestDashboard_RendersAggregateOfTwoProbes(t *testing.T) {
	t.Parallel()

	newProbe := func(name string, critical bool, unhealthy bool) *health.Probe {
		injector := do.New()

		if unhealthy {
			do.ProvideNamed(injector, "dependency", func(_ do.Injector) (*unhealthyService, error) {
				return &unhealthyService{reason: "connection refused"}, nil
			})
			do.MustInvokeNamed[*unhealthyService](injector, "dependency")
		} else {
			do.ProvideNamed(injector, "dependency", func(_ do.Injector) (*healthyService, error) {
				return &healthyService{}, nil
			})
			do.MustInvokeNamed[*healthyService](injector, "dependency")
		}

		opts := []health.Option{health.WithRefreshInterval(20 * time.Millisecond)}
		if critical {
			opts = append(opts, health.WithCriticalServices("dependency"))
		}

		probe := health.New(injector, opts...)

		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		if err := probe.Start(ctx); err != nil {
			t.Fatalf("probe.Start: %v", err)
		}

		return probe
	}

	agg, err := aggregate.New(
		aggregate.Source{Name: "api", Probe: newProbe("api", true, false)},
		aggregate.Source{Name: "web", Probe: newProbe("web", false, true)},
	)
	if err != nil {
		t.Fatalf("aggregate.New: %v", err)
	}

	dash := dashboard.New(agg)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// The HTML dashboard must show both sources' namespaced checks and the
	// worst-of overall status (warn, from the degraded web source).
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read dashboard HTML: %v", err)
	}

	for _, want := range []string{"api/dependency", "web/dependency"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("dashboard HTML missing %q", want)
		}
	}

	if !strings.Contains(string(body), "Degraded") {
		t.Fatal("dashboard HTML missing degraded status banner")
	}

	// The readiness endpoint serves the merged JSON with 200 (warn).
	ready, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer ready.Body.Close()

	if ready.StatusCode != http.StatusOK {
		t.Fatalf("GET /readyz status = %d, want 200 (warn stays 200)", ready.StatusCode)
	}
}

// TestJSON_SanitizesInvalidUTF8FromStubProber proves the dashboard sanitizes
// the probe snapshot before any write seam consumes it: service-supplied
// fields (check errors, version strings) may contain invalid UTF-8, and
// go-health's SanitizeResponse replaces those bytes with U+FFFD. The
// dashboard's response choke point (currentResponse) applies it, so the JSON
// response — and, by the same path, webhook payloads, SSE patches, metrics,
// and CSV export — stays valid under jsonv2 semantics even for hostile
// upstream bytes.
func TestJSON_SanitizesInvalidUTF8FromStubProber(t *testing.T) {
	t.Parallel()

	garbage := "connection reset by \xff\xfe peer"
	resp := health.Response{
		Status:  health.StatusFail,
		Version: "1.\xc3(0",
		Checks: map[string]health.Check{
			"db": {Status: health.StatusFail, Error: garbage},
		},
	}

	dash := dashboard.New(newStubProber(resp))

	w := doRequestWithAccept(t, dash.Handler(), "/health", "application/json")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 for fail", w.Code)
	}

	body := w.Body.String()
	if !utf8.ValidString(body) {
		t.Error("served JSON contains invalid UTF-8; want sanitized output")
	}
	if !strings.Contains(body, "\uFFFD") {
		t.Error("served JSON does not contain U+FFFD replacement for invalid bytes")
	}
	if strings.Contains(body, "\xff") || strings.Contains(body, "\xfe") {
		t.Error("served JSON leaks raw invalid bytes")
	}

	var back health.Response
	if err := json.Unmarshal([]byte(body), &back); err != nil {
		t.Fatalf("served JSON does not round-trip: %v", err)
	}
	if got := back.Checks["db"].Error; !strings.Contains(got, "\uFFFD") {
		t.Errorf("check error after round-trip = %q, want U+FFFD substitution", got)
	}
}
