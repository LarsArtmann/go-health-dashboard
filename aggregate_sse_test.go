package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/go-health/aggregate"
	"github.com/samber/do/v2"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// TestAggregateSSE_ServesInitialStateStream is a diagnostic for the
// aggregate browser test: the SSE endpoint must serve the initial Datastar
// patch for an aggregate prober over plain HTTP before any browser is
// involved.
func TestAggregateSSE_ServesInitialStateStream(t *testing.T) {
	apiInjector := do.New()
	provideHealthy(apiInjector, "postgres")
	invoke[*healthyService](t, apiInjector, "postgres")

	apiProbe := health.New(apiInjector, health.WithRefreshInterval(50*time.Millisecond))
	if err := apiProbe.Start(context.Background()); err != nil {
		t.Fatalf("api probe start: %v", err)
	}
	defer apiProbe.Shutdown()

	agg, err := aggregate.New(aggregate.Source{Name: "api", Probe: apiProbe})
	if err != nil {
		t.Fatalf("aggregate.New: %v", err)
	}

	dash := dashboard.New(agg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dash.Start(ctx); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	defer dash.Shutdown()

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/health/sse", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE request: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type: want text/event-stream, got %q", ct)
	}

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)

	if !strings.Contains(string(buf[:n]), "datastar-patch-elements") {
		t.Fatalf("no datastar patch event in first read: %q", string(buf[:n]))
	}

	if !strings.Contains(string(buf[:n]), "api/postgres") {
		t.Errorf("initial patch missing namespaced check: %q", string(buf[:n]))
	}
}
