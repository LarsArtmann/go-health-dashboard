package dashboard_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/larsartmann/go-health/aggregate"
	"github.com/samber/do/v2"
)

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

	aggregateSources := make([]aggregate.Source, 0, sourceCount)

	for i := range sourceCount {
		injector := do.New()

		for j := range checksPerSrc {
			do.ProvideNamed(
				injector,
				fmt.Sprintf("service-%d", j),
				func(_ do.Injector) (healthyService, error) {
					return healthyService{}, nil
				},
			)

			_ = do.MustInvokeNamed[healthyService](injector, fmt.Sprintf("service-%d", j))
		}

		probe := health.New(injector, health.WithRefreshInterval(100*time.Millisecond))
		if err := probe.Start(context.Background()); err != nil {
			t.Fatalf("probe %d start: %v", i, err)
		}

		defer probe.Shutdown()

		aggregateSources = append(aggregateSources, aggregate.Source{
			Name:  fmt.Sprintf("src-%02d", i),
			Probe: probe,
		})
	}

	agg, err := aggregate.New(aggregateSources...)
	if err != nil {
		t.Fatalf("aggregate.New: %v", err)
	}

	s := setupDashboardWithProber(t, agg,
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithPushInterval(100*time.Millisecond),
		dashboard.WithMetrics(true),
		dashboard.WithTrend(64),
	)
	defer s.cleanup()

	server := httptest.NewServer(s.mux)
	defer server.Close()

	client := &http.Client{
		Timeout: 0, // streams stay open for the whole run
	}

	ctx, cancel := context.WithTimeout(context.Background(), runDuration)
	defer cancel()

	var (
		eventsMu          sync.Mutex
		eventsPerClient   = make([]int, sseClients)
		firstEventElapsed time.Duration
		firstEventOnce    sync.Once
		streamWG          sync.WaitGroup
	)

	streamStart := time.Now()

	for clientIndex := range sseClients {
		streamWG.Add(1)

		go func(clientIndex int) {
			defer streamWG.Done()

			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				server.URL+"/health/sse",
				nil,
			)
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

			buf := make([]byte, 32*1024)

			for {
				n, err := resp.Body.Read(buf)
				if n > 0 && strings.Contains(string(buf[:n]), "event:") {
					eventsMu.Lock()
					eventsPerClient[clientIndex]++
					eventsMu.Unlock()

					firstEventOnce.Do(func() {
						firstEventElapsed = time.Since(streamStart)
					})
				}

				if err != nil {
					if ctx.Err() == nil {
						t.Errorf("sse read %d: %v", clientIndex, err)
					}

					return
				}

				if ctx.Err() != nil {
					return
				}
			}
		}(clientIndex)
	}

	// Concurrent scrapes while the SSE readers stream.
	var (
		scrapeLatencies []time.Duration
		scrapeMu        sync.Mutex
		scrapeWG        sync.WaitGroup
	)

	for range scrapeWorkers {
		scrapeWG.Add(1)

		go func() {
			defer scrapeWG.Done()

			for i := range scrapesEach {
				path := "/health/metrics"
				if i%2 == 1 {
					path = "/health"
				}

				start := time.Now()

				resp, err := client.Get(server.URL + path)
				if err != nil {
					t.Errorf("scrape: %v", err)

					return
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				scrapeMu.Lock()
				scrapeLatencies = append(scrapeLatencies, time.Since(start))
				scrapeMu.Unlock()
			}
		}()
	}

	scrapeWG.Wait()
	streamWG.Wait()

	totalEvents := 0

	for _, events := range eventsPerClient {
		totalEvents += events
	}

	if totalEvents == 0 {
		t.Fatal("no SSE events received by any client")
	}

	sort.Slice(
		scrapeLatencies,
		func(i, j int) bool { return scrapeLatencies[i] < scrapeLatencies[j] },
	)

	pct := func(p float64) time.Duration {
		if len(scrapeLatencies) == 0 {
			return 0
		}

		return scrapeLatencies[int(float64(len(scrapeLatencies)-1)*p)]
	}

	t.Logf(
		"load: %d sources × %d checks, %d SSE clients, %d scrapes over %s",
		sourceCount, checksPerSrc, sseClients, len(scrapeLatencies), runDuration,
	)
	t.Logf(
		"sse: %d events total (~%d/client), first event after %s",
		totalEvents, totalEvents/max(sseClients, 1), firstEventElapsed,
	)
	t.Logf(
		"scrape latency: p50=%s p95=%s max=%s",
		pct(0.5), pct(0.95), scrapeLatencies[len(scrapeLatencies)-1],
	)
}
