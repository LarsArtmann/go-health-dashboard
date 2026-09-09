package dashboard_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/larsartmann/go-health/aggregate"
	"github.com/samber/do/v2"
)

// streamBufferSize is the fixed SSE read buffer (an array, so the read
// slice is derived once per reader without allocation-shape linter noise).
const streamBufferSize = 32 * 1024

// TestLoad_Aggregate20Sources is the F8 load harness: a 20-source
// aggregate (60 checks total) under concurrent SSE readers and HTTP
// scrapes. Skipped unless LOADTEST=1 — it holds twenty streaming
// connections open for several seconds and belongs to research runs,
// not the CI suite. Results are recorded in docs/research/.
func TestLoad_Aggregate20Sources(t *testing.T) {
	if os.Getenv("LOADTEST") == "" {
		t.Skip("load test: set LOADTEST=1 to run (research harness, not CI)")
	}

	t.Parallel()

	const (
		sourceCount   = 20
		checksPerSrc  = 3
		sseClients    = 20
		scrapeWorkers = 8
		scrapesEach   = 25
		runDuration   = 5 * time.Second
	)

	agg := buildLoadAggregate(t, sourceCount, checksPerSrc)

	s := setupDashboardWithProber(t, agg,
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithPushInterval(100*time.Millisecond),
		dashboard.WithMetrics(true),
		dashboard.WithTrend(64),
	)
	defer s.cleanup()

	server := httptest.NewServer(s.mux)
	defer server.Close()

	client := &http.Client{Timeout: 0} // streams stay open for the whole run

	ctx, cancel := context.WithTimeout(context.Background(), runDuration)
	defer cancel()

	streamWG, totalEvents, firstEvent := streamLoadClients(t, ctx, server.URL, client, sseClients)

	scrapeLatencies := runLoadScrapes(t, server.URL, client, scrapeWorkers, scrapesEach)

	streamWG.Wait()

	reportLoadResults(t, sourceCount, checksPerSrc, sseClients, runDuration,
		*totalEvents, *firstEvent, scrapeLatencies)
}

// buildLoadAggregate starts sourceCount probes with checksPerSrc healthy
// checks each and merges them into one aggregate.
func buildLoadAggregate(t *testing.T, sourceCount, checksPerSrc int) *aggregate.Aggregate {
	t.Helper()

	sources := make([]aggregate.Source, 0, sourceCount)

	for sourceIndex := range sourceCount {
		injector := do.New()

		for checkIndex := range checksPerSrc {
			name := fmt.Sprintf("service-%d", checkIndex)

			do.ProvideNamed(injector, name, func(_ do.Injector) (healthyService, error) {
				return healthyService{}, nil
			})

			_ = do.MustInvokeNamed[healthyService](injector, name)
		}

		probe := health.New(injector, health.WithRefreshInterval(100*time.Millisecond))
		if err := probe.Start(context.Background()); err != nil {
			t.Fatalf("probe %d start: %v", sourceIndex, err)
		}

		t.Cleanup(probe.Shutdown)

		sources = append(sources, aggregate.Source{
			Name:  fmt.Sprintf("src-%02d", sourceIndex),
			Probe: probe,
		})
	}

	agg, err := aggregate.New(sources...)
	if err != nil {
		t.Fatalf("aggregate.New: %v", err)
	}

	return agg
}

// streamLoadClients opens clientCount SSE readers until ctx expires and
// returns their waitgroup plus the total event count and time of the
// first event.
func streamLoadClients(
	t *testing.T,
	ctx context.Context,
	baseURL string,
	client *http.Client,
	clientCount int,
) (*sync.WaitGroup, *int, *time.Duration) {
	t.Helper()

	var (
		eventsMu       sync.Mutex
		totalEvents    int
		firstEvent     time.Duration
		firstEventOnce sync.Once
		streamWG       sync.WaitGroup
	)

	streamStart := time.Now()

	for clientIndex := range clientCount {
		streamWG.Go(func() {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health/sse", nil)
			if err != nil {
				t.Errorf("sse request: %v", err)

				return
			}

			resp, err := client.Do(req)
			if err != nil {
				if ctx.Err() == nil {
					t.Errorf("sse connect %d: %v", clientIndex, err)
				}

				return
			}

			defer resp.Body.Close()

			var buf [streamBufferSize]byte

			for ctx.Err() == nil {
				n, err := resp.Body.Read(buf[:])
				if n > 0 && strings.Contains(string(buf[:n]), "event:") {
					eventsMu.Lock()
					totalEvents++
					eventsMu.Unlock()

					firstEventOnce.Do(func() { firstEvent = time.Since(streamStart) })
				}

				if err != nil {
					if ctx.Err() == nil {
						t.Errorf("sse read %d: %v", clientIndex, err)
					}

					return
				}
			}
		})
	}

	return &streamWG, &totalEvents, &firstEvent
}

// runLoadScrapes hammers /health and /health/metrics from scrapeWorkers
// concurrent workers and returns all observed latencies.
func runLoadScrapes(
	t *testing.T,
	baseURL string,
	client *http.Client,
	workers, scrapesEach int,
) []time.Duration {
	t.Helper()

	var (
		latencies []time.Duration
		latencyMu sync.Mutex
		scrapeWG  sync.WaitGroup
	)

	for range workers {
		scrapeWG.Go(func() {
			for scrapeIndex := range scrapesEach {
				path := "/health/metrics"
				if scrapeIndex%2 == 1 {
					path = "/health"
				}

				start := time.Now()

				resp, err := client.Get(baseURL + path)
				if err != nil {
					t.Errorf("scrape: %v", err)

					return
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				latencyMu.Lock()
				latencies = append(latencies, time.Since(start))
				latencyMu.Unlock()
			}
		})
	}

	scrapeWG.Wait()

	return latencies
}

// reportLoadResults logs the comparison baseline recorded in
// docs/research/2026-09-10_aggregate-load-test.md.
func reportLoadResults(
	t *testing.T,
	sources, checksPerSrc, sseClients int,
	runDuration time.Duration,
	totalEvents int,
	firstEvent time.Duration,
	latencies []time.Duration,
) {
	t.Helper()

	if totalEvents == 0 {
		t.Fatal("no SSE events received by any client")
	}

	slices.Sort(latencies)

	pct := func(p float64) time.Duration {
		if len(latencies) == 0 {
			return 0
		}

		return latencies[int(float64(len(latencies)-1)*p)]
	}

	t.Logf(
		"load: %d sources × %d checks, %d SSE clients, %d scrapes over %s",
		sources, checksPerSrc, sseClients, len(latencies), runDuration,
	)
	t.Logf(
		"sse: %d events total (~%d/client), first event after %s",
		totalEvents, totalEvents/max(sseClients, 1), firstEvent,
	)
	t.Logf(
		"scrape latency: p50=%s p95=%s max=%s",
		pct(0.5), pct(0.95), latencies[len(latencies)-1],
	)
}
