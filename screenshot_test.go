package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// TestCaptureREADME_Screenshot renders the dashboard in headless Chrome and
// writes a PNG to SCREENSHOT_OUTPUT. Skipped unless SCREENSHOT_OUTPUT is set:
//
//	SCREENSHOT_OUTPUT=docs/screenshot.png \
//	GO_HEALTH_DASHBOARD_CHROME=/path/to/chromium \
//	go test -run TestCaptureREADME_Screenshot -v .
//
// The Tailwind Play CDN is used on purpose so the capture needs no CSS build.
func TestCaptureREADME_Screenshot(t *testing.T) {
	t.Parallel()

	out := os.Getenv("SCREENSHOT_OUTPUT")
	if out == "" {
		t.Skip("SCREENSHOT_OUTPUT not set; screenshot capture is manual")
	}

	chromePath := findChrome(t)

	injector := do.New()
	provideHealthy(injector, "postgres")
	provideHealthy(injector, "api-gateway")
	provideUnhealthy(injector, "metrics-exporter", "exporter endpoint unreachable")
	invoke[*healthyService](t, injector, "postgres")
	invoke[*healthyService](t, injector, "api-gateway")
	invoke[*unhealthyService](t, injector, "metrics-exporter")

	probe := health.New(injector,
		health.WithVersion("1.2.3"),
		health.WithCriticalServices("postgres"),
		health.WithRefreshInterval(100*time.Millisecond),
	)

	dash := dashboard.New(probe,
		dashboard.WithTitle("Production Cluster"),
		dashboard.WithPushInterval(100*time.Millisecond),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithTrend(40),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	defer func() {
		dash.Shutdown()
		probe.Shutdown()
	}()

	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Pin the light theme before the dashboard loads: same-origin
	// localStorage survives across navigations.
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/healthz"),
		chromedp.Evaluate(`localStorage.setItem('theme', 'light')`, nil),
		chromedp.Navigate(server.URL+"/health"),
	); err != nil {
		t.Fatalf("navigate: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("SSE never connected during screenshot capture")
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Let the trend sparkline accumulate samples and a patch cycle run so
	// the captured page shows live data, not just the initial render.
	time.Sleep(600 * time.Millisecond)

	var png []byte

	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1280, 800),
		chromedp.FullScreenshot(&png, 95),
	); err != nil {
		t.Fatalf("screenshot: %v", err)
	}

	if err := os.WriteFile(
		out,
		png,
		0o600,
	); err != nil { //nolint:gosec // output path is the operator-provided SCREENSHOT_OUTPUT
		t.Fatalf("write screenshot: %v", err)
	}

	t.Logf("screenshot written to %s (%d bytes)", out, len(png))
}
