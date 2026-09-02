package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// benchMux bundles a started dashboard's mux with its teardown.
type benchMux struct {
	serve   *http.ServeMux
	cleanup func()
}

// newBenchMux builds a fully started dashboard without *testing.T helpers
// so it works inside benchmarks.
func newBenchMux(b *testing.B, opts ...dashboard.Option) *benchMux {
	b.Helper()

	injector := do.New()
	do.ProvideNamed(injector, "db", func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
	do.ProvideNamed(injector, "cache", func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
	do.MustInvokeNamed[*healthyService](injector, "db")
	do.MustInvokeNamed[*healthyService](injector, "cache")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(50*time.Millisecond),
	)
	dash := dashboard.New(probe, append([]dashboard.Option{
		dashboard.WithPushInterval(50 * time.Millisecond),
	}, opts...)...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(context.Background())

	if err := probe.Start(ctx); err != nil {
		b.Fatal(err)
	}

	if err := dash.Start(ctx); err != nil {
		b.Fatal(err)
	}

	return &benchMux{
		serve: mux,
		cleanup: func() {
			cancel()
			dash.Shutdown()
			probe.Shutdown()
		},
	}
}

func BenchmarkMetrics_Exposition(b *testing.B) {
	bench := newBenchMux(b, dashboard.WithMetrics(true))
	defer bench.cleanup()

	r := httptest.NewRequest(http.MethodGet, "/health/metrics", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		bench.serve.ServeHTTP(w, r)
	}
}

func BenchmarkDashboard_PatchRender(b *testing.B) {
	bench := newBenchMux(b, dashboard.WithTrend(120))
	defer bench.cleanup()

	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		bench.serve.ServeHTTP(w, r)
	}
}

func BenchmarkDashboard_FullHTMLWithTrend(b *testing.B) {
	bench := newBenchMux(b, dashboard.WithTrend(120), dashboard.WithMetrics(true))
	defer bench.cleanup()

	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		bench.serve.ServeHTTP(w, r)
	}
}
