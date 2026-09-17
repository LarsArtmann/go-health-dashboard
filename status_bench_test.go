package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// benchMetadataChecks builds a severity-flat response of n checks with full
// go-health v0.2.0 metadata (state-entry time + execution duration) — the
// worst case for the per-tick stamping loop in buildViewModelAt.
func benchMetadataChecks(n int, withMetadata bool) health.Response {
	since := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	checks := make(map[string]health.Check, n)
	for i := range n {
		check := health.Check{Status: health.StatusPass}
		if withMetadata {
			check.Since = since
			check.DurationNanos = int64(823 * time.Microsecond)
		}

		checks[fmt.Sprintf("svc-%02d.DatabaseService", i)] = check
	}

	return health.Response{Status: health.StatusPass, Checks: checks}
}

// BenchmarkBuildViewModelAt_Stamping isolates the per-tick stamping loop:
// derive SinceText/DurationText for every row, 60 checks with full
// metadata versus the same table with none. The delta is the cost the
// pusher pays per tick for the v0.2.0 metadata line.
func BenchmarkBuildViewModelAt_Stamping(b *testing.B) {
	now := time.Date(2026, 9, 17, 12, 17, 0, 0, time.UTC)

	for _, testCase := range []struct {
		name         string
		withMetadata bool
	}{
		{name: "no_metadata", withMetadata: false},
		{name: "with_metadata", withMetadata: true},
	} {
		resp := benchMetadataChecks(60, testCase.withMetadata)

		b.Run(testCase.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				vm := buildViewModelAt(resp, "bench", "/health/sse", GroupBySeverity, now)
				if len(vm.Groups) == 0 {
					b.Fatal("no groups built")
				}
			}
		})
	}
}

// benchRenderPusher builds a pusher for render benchmarks: the cfg drives
// every renderPatch input, and the retry value isolates the retry-stamping
// delta (zero = no retry field on the event).
func benchRenderPusher(retry time.Duration) *pusher {
	d := &Dashboard{cfg: Config{
		Title:         "bench",
		Routes:        DefaultRoutes(),
		PushInterval:  2 * time.Second,
		RetryInterval: retry,
	}}

	return newPusher(d)
}

// BenchmarkRenderPatch measures the full pusher tick render (view model →
// templ → Datastar patch → SSE event) for a 60-check metadata-bearing
// response, with and without retry stamping: the delta between the
// sub-benchmarks is the retry-stamping overhead the plan asked to size.
func BenchmarkRenderPatch(b *testing.B) {
	resp := benchMetadataChecks(60, true)

	for _, retry := range []time.Duration{0, 2 * time.Second} {
		p := benchRenderPusher(retry)

		b.Run(fmt.Sprintf("retry_%s", retry), func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				if _, ok := p.renderPatch(resp); !ok {
					b.Fatal("renderPatch failed")
				}
			}
		})
	}
}

// benchProber is the minimal Prober for lifecycle benchmarks.
type benchProber struct {
	resp health.Response
}

func (p benchProber) CachedResponse() health.Response { return p.resp }

func (p benchProber) RefreshInterval() time.Duration { return 2 * time.Second }

func (p benchProber) LivenessHandler() http.HandlerFunc { return nil }

func (p benchProber) ReadinessHandler() http.HandlerFunc { return nil }

func (p benchProber) StartupHandler() http.HandlerFunc { return nil }

// BenchmarkDashboard_HealthCheck measures the do.Healthchecker hook the DI
// container calls per container-wide health check: an atomic pointer load
// plus a pusher-state check.
func BenchmarkDashboard_HealthCheck(b *testing.B) {
	dash := New(benchProber{resp: benchMetadataChecks(60, true)})

	if err := dash.Start(context.Background()); err != nil {
		b.Fatalf("start: %v", err)
	}

	defer dash.Shutdown()

	ctx := context.Background()

	b.ReportAllocs()

	for b.Loop() {
		if err := dash.HealthCheck(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
