package dashboard_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	healthfederation "github.com/larsartmann/go-health/federation"
)

// Compile-time guarantee that the federation prober (go-health v0.4.0)
// satisfies the dashboard's consumer-side Prober interface: the hub in
// cmd/health-hub and every direct federation consumer depend on this
// structural fit, so a widened dashboard.Prober or a changed federation
// surface must break here, loudly, instead of in a consumer's build.
var _ dashboard.Prober = (*healthfederation.Prober)(nil)

// serveHealthyRemote spins up an httptest server speaking the go-health
// wire shape with a single passing check, standing in for a live remote.
func serveHealthyRemote(t *testing.T) *httptest.Server {
	t.Helper()

	const body = `{"status":"pass","checks":{"api":{"status":"pass"}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

// closedPort returns a host:port with no listener, so the remote is
// unreachable by instant connection-refused rather than by a slow timeout.
func closedPort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}

	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release reserved port: %v", err)
	}

	return addr
}

// TestFederation_UnreachableRemoteCountsAsEvidence drives the real
// federation prober against one live remote and one dead one, and verifies
// the evidence strip counts the synthetic "dark/reachable" fail row as a
// non-pass observation: a dark remote must make the dashboard less
// proven, never silently green.
func TestFederation_UnreachableRemoteCountsAsEvidence(t *testing.T) {
	t.Parallel()

	upstream := serveHealthyRemote(t)

	fed, err := healthfederation.New([]healthfederation.Remote{
		{Name: "api", URL: upstream.URL + "/health"},
		{Name: "dark", URL: "http://" + closedPort(t) + "/health"},
	})
	if err != nil {
		t.Fatalf("federation.New: %v", err)
	}

	dash := dashboard.New(fed,
		dashboard.WithPushInterval(5*time.Millisecond),
		dashboard.WithEmbeddedDatastarSDK(),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	t.Cleanup(dash.Shutdown)

	// Poll for the strip wording itself (not the row): the reachable row
	// renders from the live response immediately, while the evidence strip
	// reflects the pusher's observation log, which needs its first tick.
	body := waitForBody(t, mux, "/health", "Failure evidence: 1 of 2 checks")

	if !strings.Contains(body, "dark/reachable") {
		t.Error("the synthetic reachable row must render in the problems group")
	}

	if !strings.Contains(body, "unproven") {
		t.Errorf(
			"the live remote's green must stay unproven: %s",
			extractLine(body, "Failure evidence"),
		)
	}
}

// TestFederation_HostileRemoteNameEscapesThroughMetrics pins the untrusted
// seam: a remote name is consumer input that reaches the hand-rolled
// Prometheus exposition via the synthetic "name/reachable" key, and the
// label escaping must hold for exposition-hostile bytes.
func TestFederation_HostileRemoteNameEscapesThroughMetrics(t *testing.T) {
	t.Parallel()

	fed, err := healthfederation.New([]healthfederation.Remote{
		{Name: `evil"remote`, URL: "http://" + closedPort(t) + "/health"},
	})
	if err != nil {
		t.Fatalf("federation.New: %v", err)
	}

	dash := dashboard.New(fed, dashboard.WithMetrics(true))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	body := doRequest(t, mux, "/health/metrics").Body.String()

	if !strings.Contains(body, `check="evil\"remote/reachable"`) {
		t.Errorf("hostile remote name must be escaped in metric labels; got:\n%s", body)
	}
}

// TestFederation_UnreachableRemoteFlowsThroughWebhook pins the webhook
// seam: the initial-state announce carries the synthetic reachable row with
// the untrusted remote name, and the JSON encoding must escape it.
func TestFederation_UnreachableRemoteFlowsThroughWebhook(t *testing.T) {
	t.Parallel()

	var payload atomic.Value

	receiver := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read webhook body: %v", err)

			return
		}
		payload.Store(string(body))
	}))
	t.Cleanup(receiver.Close)

	fed, err := healthfederation.New([]healthfederation.Remote{
		{Name: `evil"remote`, URL: "http://" + closedPort(t) + "/health"},
	})
	if err != nil {
		t.Fatalf("federation.New: %v", err)
	}

	dash := dashboard.New(fed,
		dashboard.WithWebhook(receiver.URL),
		dashboard.WithPushInterval(5*time.Millisecond),
	)

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	t.Cleanup(dash.Shutdown)

	deadline := time.Now().Add(3 * time.Second)
	for payload.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}

	got, _ := payload.Load().(string)
	if got == "" {
		t.Fatal("webhook announce never arrived")
	}

	// The name and the synthesized row must survive the wire; the quote in
	// the remote name must be JSON-escaped, never raw.
	if !strings.Contains(got, `evil\"remote/reachable`) {
		t.Errorf("webhook payload must carry the escaped reachable key; got: %s", got)
	}

	if !strings.Contains(got, string(health.StatusFail)) {
		t.Errorf("webhook payload must carry the reachable row as fail; got: %s", got)
	}
}

// TestFederation_HealthzCapabilityNotForwarded completes the capability
// matrix: the federation prober does NOT implement go-health v0.4.0's
// Healthz (only Probe and aggregate do), so hub deployments get no
// combined-traffic route even when Routes.Healthz is configured.
func TestFederation_HealthzCapabilityNotForwarded(t *testing.T) {
	t.Parallel()

	remote := serveHealthyRemote(t)

	fed, err := healthfederation.New([]healthfederation.Remote{
		{Name: "edge", URL: remote.URL},
	})
	if err != nil {
		t.Fatalf("federation.New: %v", err)
	}

	if _, ok := any(fed).(dashboard.Healthzer); ok {
		t.Fatal("federation prober must not expose a Healthz capability")
	}

	routes := dashboard.DefaultRoutes()
	routes.Healthz = "/livez"

	dash := dashboard.New(fed, dashboard.WithRoutes(routes))

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if code := doRequest(t, mux, "/livez").Code; code != http.StatusNotFound {
		t.Errorf("combined-traffic route must stay unregistered through federation, got %d", code)
	}
}
