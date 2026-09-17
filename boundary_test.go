package dashboard_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
)

// --- Trend/export boundaries (plan M80) ---

// TestWithTrend_OneSampleBoundary pins the WithTrend(1) edge: a one-slot
// ring is legal, trend/export serve exactly one sample per tick, and the
// sparkline card — which requires at least two samples — stays hidden.
func TestWithTrend_OneSampleBoundary(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t, dashboard.WithTrend(1))
	defer s.cleanup()

	waitForTrendSamples(t, s, 1)

	w := doRequest(t, s.mux, "/health/trend")
	if w.Code != http.StatusOK {
		t.Fatalf("trend status: want 200, got %d", w.Code)
	}

	if got := strings.Count(w.Body.String(), `"status"`); got != 1 {
		t.Errorf(
			"trend samples with a one-slot ring: want exactly 1 sample, got %d (%s)",
			got,
			w.Body.String(),
		)
	}

	csv := doRequestWithAccept(t, s.mux, "/health/export", "text/csv")
	if csv.Code != http.StatusOK ||
		!strings.HasPrefix(csv.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf(
			"csv export: want 200 text/csv, got %d %s",
			csv.Code,
			csv.Header().Get("Content-Type"),
		)
	}

	if rows := strings.Count(strings.TrimSpace(csv.Body.String()), "\n"); rows != 1 {
		t.Errorf(
			"csv rows with a one-slot ring: want header + 1 sample, got %d newlines (%s)",
			rows,
			csv.Body.String(),
		)
	}

	html := doRequest(t, s.mux, "/health")
	if strings.Contains(html.Body.String(), "Health Trend") {
		t.Error("one-slot ring rendered a trend card; the card requires >= 2 samples")
	}
}

// TestExportHandler_AcceptCSVWithQValue pins the q-valued Accept variant of
// the CSV negotiation: text/csv at any q-value (not just a bare header)
// selects CSV.
func TestExportHandler_AcceptCSVWithQValue(t *testing.T) {
	t.Parallel()

	s, _ := setupTrendDashboard(t)
	defer s.cleanup()

	waitForTrendSamples(t, s, 2)

	w := doRequestWithAccept(t, s.mux, "/health/export", "text/csv;q=0.8")
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content-type: want text/csv, got %s", ct)
	}

	if !strings.HasPrefix(w.Body.String(), "timestamp,value,status") {
		t.Errorf("csv header missing: %q", w.Body.String())
	}
}

// --- BasePath boundaries (plan M81) ---

// TestWithBasePath_EdgePrefixes pins the BasePath normalization: any number
// of trailing slashes is stripped, an empty result disables the prefix, and
// nested paths compose. Guards against the double-slash route class
// ("//health") that a single-suffix trim would still produce for "//".
func TestWithBasePath_EdgePrefixes(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		prefix string
		want   string
	}{
		{prefix: "", want: "/health"},
		{prefix: "/", want: "/health"},
		{prefix: "//", want: "/health"},
		{prefix: "/admin/", want: "/admin/health"},
		{prefix: "/a/b", want: "/a/b/health"},
		{prefix: "/a/b/", want: "/a/b/health"},
	} {
		dash := dashboard.New(
			newStubProber(health.Response{Status: health.StatusPass}),
			dashboard.WithBasePath(testCase.prefix),
		)

		mux := http.NewServeMux()
		dash.RegisterRoutes(mux)

		w := doRequest(t, mux, testCase.want)
		if w.Code != http.StatusOK {
			t.Errorf(
				"prefix %q: GET %s: want 200, got %d",
				testCase.prefix,
				testCase.want,
				w.Code,
			)
		}
	}
}

// --- RetryInterval boundaries (plan M81) ---

// TestWithRetryInterval_EdgeValues pins the SSE retry field at the
// boundaries: non-positive values emit no retry field (the option treats
// them as unset), a sub-millisecond value rounds to zero milliseconds and
// is therefore omitted too (go-sse skips retry: 0, restoring the browser
// default), and huge values keep their millisecond precision.
func TestWithRetryInterval_EdgeValues(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		value     time.Duration
		wantRetry string
	}{
		{name: "negative is unset", value: -time.Second, wantRetry: ""},
		{name: "zero is unset", value: 0, wantRetry: ""},
		{name: "sub-millisecond rounds away", value: 500 * time.Microsecond, wantRetry: ""},
		{name: "two seconds", value: 2 * time.Second, wantRetry: "2000"},
		{name: "huge keeps precision", value: 10000 * time.Hour, wantRetry: "36000000000"},
	} {
		svc := &toggleService{}
		svc.healthy.Store(true)

		server, _, cleanup := setupSSEServer(
			t,
			50*time.Millisecond,
			svc,
			dashboard.WithRetryInterval(testCase.value),
		)

		resp, stream := connectSSE(t, server)
		evt := stream.waitFor(t, func(string) bool { return true }, 2*time.Second)
		_ = resp.Body.Close()
		server.Close()
		cleanup()

		if testCase.wantRetry == "" {
			if strings.Contains(evt, "retry:") {
				t.Errorf("%s: event must not carry a retry field, got:\n%s", testCase.name, evt)
			}

			continue
		}

		if !strings.Contains(evt, "retry: "+testCase.wantRetry) {
			t.Errorf(
				"%s: event must carry retry: %s, got:\n%s",
				testCase.name,
				testCase.wantRetry,
				evt,
			)
		}
	}
}
