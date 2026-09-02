package dashboard

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"

	health "github.com/larsartmann/go-health"
)

// prometheusContentType is the Content-Type of the Prometheus text
// exposition format (version 0.0.4).
const prometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

// MetricsHandler returns an http.HandlerFunc that serves dashboard metrics in
// the Prometheus text exposition format. Enable the route with WithMetrics
// and configure its path via Routes.Metrics (default /health/metrics).
//
// Exposed metrics:
//
//	dashboard_health_up              1 when overall status is pass, else 0
//	dashboard_health_status          2 pass, 1 warn, 0 fail, -1 unknown
//	dashboard_health_check{...}      1 when the check passes, else 0
//	dashboard_health_latency_ms      wall-clock time of last check batch
//	dashboard_health_shutting_down   1 when the probe is shutting down
//	dashboard_sse_connections        current SSE client count
//	dashboard_pusher_active          1 when the SSE pusher is running
//
// The exposition is hand-rolled to keep this module dependency-free; point
// Prometheus at the route and scrape as usual.
func (d *Dashboard) MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", prometheusContentType)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(d.renderMetrics()))
	}
}

// renderMetrics produces the full exposition payload for a scrape.
func (d *Dashboard) renderMetrics() string {
	resp := d.currentResponse()

	var b strings.Builder

	b.WriteString(
		"# HELP dashboard_health_up Whether the overall health status is pass (1) or not (0).\n",
	)
	b.WriteString("# TYPE dashboard_health_up gauge\n")
	fmt.Fprintf(&b, "dashboard_health_up %d\n", boolGauge(resp.Status == health.StatusPass))

	b.WriteString(
		"# HELP dashboard_health_status Overall health status encoded numerically: 2 pass, 1 warn, 0 fail, -1 unknown.\n",
	)
	b.WriteString("# TYPE dashboard_health_status gauge\n")
	fmt.Fprintf(&b, "dashboard_health_status %d\n", numericHealthStatus(resp.Status))

	b.WriteString(
		"# HELP dashboard_health_check Individual check health: 1 when the check passes, 0 otherwise.\n",
	)
	b.WriteString("# TYPE dashboard_health_check gauge\n")

	for _, name := range sortedCheckNames(resp.Checks) {
		check := resp.Checks[name]

		fmt.Fprintf(&b, "dashboard_health_check{check=\"%s\",status=\"%s\"} %d\n",
			escapeLabelValue(name),
			escapeLabelValue(string(check.Status)),
			boolGauge(check.Status == health.StatusPass),
		)
	}

	b.WriteString(
		"# HELP dashboard_health_latency_ms Wall-clock time spent running the last health-check batch, in milliseconds.\n",
	)
	b.WriteString("# TYPE dashboard_health_latency_ms gauge\n")
	fmt.Fprintf(&b, "dashboard_health_latency_ms %d\n", resp.TotalLatencyMs)

	b.WriteString(
		"# HELP dashboard_health_shutting_down Whether the probe has been marked for shutdown (1) or not (0).\n",
	)
	b.WriteString("# TYPE dashboard_health_shutting_down gauge\n")
	fmt.Fprintf(&b, "dashboard_health_shutting_down %d\n", boolGauge(resp.ShuttingDown))

	b.WriteString("# HELP dashboard_sse_connections Current number of connected SSE clients.\n")
	b.WriteString("# TYPE dashboard_sse_connections gauge\n")
	fmt.Fprintf(&b, "dashboard_sse_connections %d\n", d.SubscriberCount())

	b.WriteString(
		"# HELP dashboard_pusher_active Whether the SSE pusher goroutine is running (1) or not (0).\n",
	)
	b.WriteString("# TYPE dashboard_pusher_active gauge\n")
	fmt.Fprintf(&b, "dashboard_pusher_active %d\n", boolGauge(d.push.Load() != nil))

	if d.latency != nil {
		d.latency.renderPrometheus(&b)
	}

	return b.String()
}

// sortedCheckNames returns the check names in deterministic order so
// successive scrapes produce byte-identical output.
func sortedCheckNames(checks map[string]health.Check) []string {
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// boolGauge maps a boolean to the Prometheus gauge convention 1/0.
func boolGauge(b bool) int {
	if b {
		return 1
	}

	return 0
}

// numericHealthStatus encodes health statuses for Prometheus consumption.
// Unknown statuses map to -1 so they can never be confused with fail (0).
func numericHealthStatus(s health.Status) int {
	switch s {
	case health.StatusPass:
		return 2
	case health.StatusWarn:
		return 1
	case health.StatusFail:
		return 0
	default:
		return -1
	}
}

// escapeLabelValue escapes a string for use inside a Prometheus label value
// per the exposition format: backslash, double quote, and newline.
func escapeLabelValue(v string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
	).Replace(v)
}

// latencyBucketBounds are the cumulative histogram bucket upper bounds in
// seconds, chosen to span fast local checks through slow timeouts.
//
//nolint:gochecknoglobals // immutable bucket bounds; a global keeps the\n// histogram zero-alloc and the exposition deterministic
var latencyBucketBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// latencyHistogram is a fixed-bucket cumulative histogram of health-check
// batch durations, hand-rolled to keep the module dependency-free. The
// pusher observes once per tick; scrapes read the counters.
// microsPerSecond scales atomic-int64 accumulation of fractional seconds.
const microsPerSecond = int64(1e6)

type latencyHistogram struct {
	buckets []atomic.Uint64 // per-bucket cumulative counts
	sum     atomic.Int64    // observed seconds, scaled by 1e6 for atomicity
	count   atomic.Uint64
}

func newLatencyHistogram() *latencyHistogram {
	return &latencyHistogram{buckets: make([]atomic.Uint64, len(latencyBucketBounds))}
}

// observe records one duration in seconds.
func (h *latencyHistogram) observe(seconds float64) {
	for i, bound := range latencyBucketBounds {
		if seconds <= bound {
			h.buckets[i].Add(1)
		}
	}

	h.count.Add(1)
	h.sum.Add(int64(seconds * float64(microsPerSecond)))
}

// renderPrometheus writes the exposition lines for the histogram, including
// the +Inf bucket, _sum, and _count.
func (h *latencyHistogram) renderPrometheus(b *strings.Builder) {
	b.WriteString(
		"# HELP dashboard_health_check_duration_seconds Wall-clock duration of health-check batches.\n",
	)
	b.WriteString("# TYPE dashboard_health_check_duration_seconds histogram\n")

	for i, bound := range latencyBucketBounds {
		fmt.Fprintf(
			b,
			"dashboard_health_check_duration_seconds_bucket{le=\"%g\"} %d\n",
			bound,
			h.buckets[i].Load(),
		)
	}

	fmt.Fprintf(
		b,
		"dashboard_health_check_duration_seconds_bucket{le=\"+Inf\"} %d\n",
		h.count.Load(),
	)
	fmt.Fprintf(
		b,
		"dashboard_health_check_duration_seconds_sum %g\n",
		float64(h.sum.Load())/float64(microsPerSecond),
	)
	fmt.Fprintf(b, "dashboard_health_check_duration_seconds_count %d\n", h.count.Load())
}
