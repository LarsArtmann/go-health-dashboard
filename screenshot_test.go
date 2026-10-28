package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/larsartmann/go-health/aggregate"
	"github.com/larsartmann/go-health/checks"
	"github.com/samber/do/v2"
)

// screenshotFileMode is the docs-file permission captures normalize to:
// git only tracks the executable bit, but a 0600 docs file trips readers
// and diff tools.
const screenshotFileMode = 0o644

// screenshotQuality is the image quality the README captures are rendered at.
const screenshotQuality = 95

// captureScreenshot renders the full page at the documented viewport.
func captureScreenshot(ctx context.Context) ([]byte, error) {
	if err := chromedp.Do(ctx, chromedp.EmulateViewport(1280, 800)); err != nil {
		return nil, err
	}

	return chromedp.Run(ctx, chromedp.FullScreenshot(screenshotQuality))
}

// timedScreenshotRecorder routes the capture probe through the detailed
// seam so the README screenshots show the full v0.2.0 metadata row:
// per-check since/age AND real execution durations (plain injector checks
// carry Since only, so the duration column would read as unknown).
type timedScreenshotRecorder struct{}

var (
	_ health.DetailedHealthRecorder = timedScreenshotRecorder{}
	_ health.HealthRecorder         = timedScreenshotRecorder{}
)

func (timedScreenshotRecorder) RecordDetailedHealthCheckWithContext(
	ctx context.Context,
	injector do.Injector,
) map[string]health.CheckDetail {
	services := injector.ListInvokedServices()
	details := make(map[string]health.CheckDetail, len(services))

	for _, svc := range services {
		start := time.Now()
		err := do.HealthCheckNamedWithContext(ctx, injector, svc.Service)
		details[svc.Service] = health.CheckDetail{Err: err, Duration: time.Since(start)}
	}

	return details
}

func (r timedScreenshotRecorder) RecordHealthCheckWithContext(
	ctx context.Context,
	injector do.Injector,
) map[string]error {
	details := r.RecordDetailedHealthCheckWithContext(ctx, injector)
	errs := make(map[string]error, len(details))

	for name, detail := range details {
		errs[name] = detail.Err
	}

	return errs
}

// normalizeScreenshotPerms chmods a capture to the docs convention. The
// out path comes from an operator-provided env variable, so the file
// permission change is annotated in one place instead of per call site.
func normalizeScreenshotPerms(t *testing.T, out string) {
	t.Helper()

	if err := os.Chmod(out, screenshotFileMode); err != nil {
		t.Fatalf("chmod screenshot: %v", err)
	}
}

// offService is a deliberately-unconfigured dependency: its check returns
// health.Off so the README captures show the intentional-absence row next
// to the verdict states.
type offService struct{ detail string }

func (s offService) HealthCheck(_ context.Context) error { return health.Off(s.detail) }

func provideOff(i do.Injector, name, detail string) {
	do.ProvideNamed(i, name, func(_ do.Injector) (offService, error) {
		return offService{detail: detail}, nil
	})
}

// batteryService adapts a go-health checks battery func to the injector
// service surface, so the README captures show a real resource row.
type batteryService struct{ fn func(context.Context) error }

func (s batteryService) HealthCheck(ctx context.Context) error { return s.fn(ctx) }

func provideBattery(i do.Injector, name string, fn func(context.Context) error) {
	do.ProvideNamed(i, name, func(_ do.Injector) (batteryService, error) {
		return batteryService{fn: fn}, nil
	})
}

// captureThemeScreenshot renders the dashboard in the requested theme and
// writes a PNG to out. Skipped when envVar is unset: screenshot capture is
// a manual documentation tool.
//
// The Tailwind Play CDN is used on purpose so the capture needs no CSS
// build. The theme is pinned via same-origin localStorage before the
// dashboard loads, mirroring the UI's own persistence.
func captureThemeScreenshot(t *testing.T, envVar, theme, out string) {
	t.Helper()

	if os.Getenv(envVar) == "" {
		t.Skipf("%s not set; screenshot capture is manual", envVar)
	}

	chromePath := findChrome(t)

	injector := do.New()
	provideHealthy(injector, "postgres")
	provideHealthy(injector, "api-gateway")
	provideUnhealthy(injector, "metrics-exporter", "exporter endpoint unreachable")
	provideOff(
		injector,
		"analytics",
		"not configured: set DEMO_ANALYTICS_URL=<url> to enable analytics",
	)
	provideBattery(injector, "memory-pressure", checks.Memory(1<<20))
	invoke[*healthyService](t, injector, "postgres")
	invoke[*healthyService](t, injector, "api-gateway")
	invoke[*unhealthyService](t, injector, "metrics-exporter")
	invoke[offService](t, injector, "analytics")
	invoke[batteryService](t, injector, "memory-pressure")

	probe := health.New(injector,
		health.WithVersion("1.2.3"),
		health.WithCriticalServices("postgres"),
		health.WithRefreshInterval(100*time.Millisecond),
		health.WithHealthRecorder(timedScreenshotRecorder{}),
	)

	s := startDashboardFixture(t, probe,
		dashboard.WithTitle("Production Cluster"),
		dashboard.WithPushInterval(100*time.Millisecond),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithTrend(40),
	)
	defer s.cleanup()

	server := httptest.NewServer(s.mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	allocCtx, allocCancel := remote.NewAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	pin := `localStorage.setItem('theme', '` + theme + `')`

	if err := chromedp.Do(ctx,
		chromedp.Navigate(server.URL+"/healthz"),
		evalDiscard(pin),
		chromedp.Navigate(server.URL+"/health"),
	); err != nil {
		t.Fatalf("navigate: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)

	for s.dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("SSE never connected during screenshot capture")
		}

		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(600 * time.Millisecond)

	png, err := captureScreenshot(ctx)
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}

	if err := os.WriteFile(out, png, 0o600); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}

	normalizeScreenshotPerms(t, out)

	t.Logf("screenshot written to %s (%d bytes)", out, len(png))
}

// TestCaptureREADME_Screenshot renders the dashboard in light mode for the
// README:
//
//	SCREENSHOT_OUTPUT=docs/screenshot.png \
//	GO_HEALTH_DASHBOARD_CHROME=/path/to/chromium \
//	go test -run TestCaptureREADME_Screenshot -v .
func TestCaptureREADME_Screenshot(t *testing.T) {
	t.Parallel()

	captureThemeScreenshot(t, "SCREENSHOT_OUTPUT", "light", os.Getenv("SCREENSHOT_OUTPUT"))
}

// TestCaptureREADME_ScreenshotDegraded renders the failure state — a
// critical service down, so the banner, jump-to-problems link, and the
// Critical Failures card all show:
//
//	SCREENSHOT_OUTPUT_DEGRADED=docs/screenshot-degraded.png \
//	GO_HEALTH_DASHBOARD_CHROME=/path/to/chromium \
//	go test -run TestCaptureREADME_ScreenshotDegraded -v .
func TestCaptureREADME_ScreenshotDegraded(t *testing.T) {
	t.Parallel()

	if os.Getenv("SCREENSHOT_OUTPUT_DEGRADED") == "" {
		t.Skip("SCREENSHOT_OUTPUT_DEGRADED not set; screenshot capture is manual")
	}

	chromePath := findChrome(t)

	injector := do.New()
	provideHealthy(injector, "api-gateway")
	provideUnhealthy(injector, "postgres", "connection refused (db-1:5432)")
	provideUnhealthy(injector, "metrics-exporter", "exporter endpoint unreachable")
	invoke[*healthyService](t, injector, "api-gateway")
	invoke[*unhealthyService](t, injector, "postgres")
	invoke[*unhealthyService](t, injector, "metrics-exporter")

	probe := health.New(injector,
		health.WithVersion("1.2.3"),
		health.WithCriticalServices("postgres"),
		health.WithRefreshInterval(100*time.Millisecond),
		health.WithHealthRecorder(timedScreenshotRecorder{}),
	)

	s := startDashboardFixture(t, probe,
		dashboard.WithTitle("Production Cluster"),
		dashboard.WithPushInterval(100*time.Millisecond),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithTrend(40),
	)
	defer s.cleanup()

	server := httptest.NewServer(s.mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	allocCtx, allocCancel := remote.NewAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if err := chromedp.Do(ctx,
		chromedp.Navigate(server.URL+"/healthz"),
		evalDiscard(`localStorage.setItem('theme', 'light')`),
		chromedp.Navigate(server.URL+"/health"),
	); err != nil {
		t.Fatalf("navigate: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)

	for s.dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("SSE never connected during screenshot capture")
		}

		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(600 * time.Millisecond)

	png, err := captureScreenshot(ctx)
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}

	out := os.Getenv("SCREENSHOT_OUTPUT_DEGRADED")

	if err := os.WriteFile(out, png, 0o600); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}

	normalizeScreenshotPerms(t, out)

	t.Logf("degraded screenshot written to %s (%d bytes)", out, len(png))
}

// TestCaptureREADME_ScreenshotGrouped renders the source-grouped page —
// the view a federation hub (cmd/health-hub) serves, one card per remote:
//
//	SCREENSHOT_OUTPUT_GROUPED=docs/screenshot-grouped.png \
//	GO_HEALTH_DASHBOARD_CHROME=/path/to/chromium \
//	go test -run TestCaptureREADME_ScreenshotGrouped -v .
func TestCaptureREADME_ScreenshotGrouped(t *testing.T) {
	t.Parallel()

	if os.Getenv("SCREENSHOT_OUTPUT_GROUPED") == "" {
		t.Skip("SCREENSHOT_OUTPUT_GROUPED not set; screenshot capture is manual")
	}

	chromePath := findChrome(t)

	apiInjector := do.New()
	provideHealthy(apiInjector, "postgres")
	provideHealthy(apiInjector, "api-gateway")
	invoke[*healthyService](t, apiInjector, "postgres")
	invoke[*healthyService](t, apiInjector, "api-gateway")

	apiProbe := health.New(apiInjector,
		health.WithRefreshInterval(100*time.Millisecond),
		health.WithHealthRecorder(timedScreenshotRecorder{}),
	)

	workerInjector := do.New()
	provideHealthy(workerInjector, "redis")
	provideUnhealthy(workerInjector, "queue", "connection refused (worker-1:5672)")
	invoke[*healthyService](t, workerInjector, "redis")
	invoke[*unhealthyService](t, workerInjector, "queue")

	workerProbe := health.New(workerInjector,
		health.WithRefreshInterval(100*time.Millisecond),
		health.WithHealthRecorder(timedScreenshotRecorder{}),
	)

	agg, err := aggregate.New(
		aggregate.Source{Name: "api", Probe: apiProbe},
		aggregate.Source{Name: "worker", Probe: workerProbe},
	)
	if err != nil {
		t.Fatalf("aggregate.New: %v", err)
	}

	dash := dashboard.New(agg,
		dashboard.WithTitle("Federation Hub"),
		dashboard.WithPushInterval(100*time.Millisecond),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithTrend(40),
		dashboard.WithGrouping(dashboard.GroupBySource),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := apiProbe.Start(t.Context()); err != nil {
		t.Fatalf("api probe start: %v", err)
	}

	if err := workerProbe.Start(t.Context()); err != nil {
		t.Fatalf("worker probe start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	defer func() {
		dash.Shutdown()
		workerProbe.Shutdown()
		apiProbe.Shutdown()
	}()

	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	allocCtx, allocCancel := remote.NewAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if err := chromedp.Do(ctx,
		chromedp.Navigate(server.URL+"/healthz"),
		evalDiscard(`localStorage.setItem('theme', 'light')`),
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

	time.Sleep(600 * time.Millisecond)

	png, err := captureScreenshot(ctx)
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}

	out := os.Getenv("SCREENSHOT_OUTPUT_GROUPED")

	if err := os.WriteFile(out, png, 0o600); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}

	normalizeScreenshotPerms(t, out)

	t.Logf("grouped screenshot written to %s (%d bytes)", out, len(png))
}
