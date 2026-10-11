package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/go-sse"
)

// subscriberBufferSlack broadcasts this many events past a full 64-slot
// subscriber buffer: enough drops to prove counting without a long loop.
const subscriberBufferSlack = 10

// TestPusher_DropCounter_CountsPerFullSubscriber pins the T18 fix: a stalled
// subscriber (channel never drained) makes Broadcast drop patches, and the
// drop counter reports them — visible in DroppedPatchCount and in the
// metrics exposition, where the HELP text discloses the per-subscriber
// counting semantics (N stalled clients missing the same patch add N).
func TestPusher_DropCounter_CountsPerFullSubscriber(t *testing.T) {
	t.Parallel()

	probe := &plainProber{resp: health.Response{
		Status: health.StatusPass,
		Checks: map[string]health.Check{"db": {Status: health.StatusPass}},
	}}
	dash := New(probe)
	p := newPusher(dash)
	// Start stores the pusher on the dashboard; mirror that wiring so the
	// accessor reads this pusher's counter without running the push loop.
	dash.push.Store(p)

	ch := p.broadcaster.Subscribe()
	defer p.broadcaster.Unsubscribe(ch)

	for range 64 + subscriberBufferSlack {
		p.broadcaster.Broadcast(sse.Event{Data: "<div>patch</div>"})
	}

	if got := dash.DroppedPatchCount(); got < subscriberBufferSlack {
		t.Errorf("DroppedPatchCount = %d, want >= %d (stalled subscriber must lose the overflow)",
			got, subscriberBufferSlack)
	}

	rec := httptest.NewRecorder()
	dash.MetricsHandler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/metrics", nil))

	line, ok := findMetricLine(rec.Body.String(), "dashboard_sse_drops_total ")
	if !ok {
		t.Fatal("metrics exposition missing dashboard_sse_drops_total")
	}

	if reported, err := strconv.ParseInt(
		line,
		10,
		64,
	); err != nil ||
		reported < subscriberBufferSlack {
		t.Errorf("metrics dashboard_sse_drops_total = %q, want >= %d", line, subscriberBufferSlack)
	}
}

// TestDashboard_DroppedPatchCount_ZeroBeforeStart pins the unstarted
// contract: no pusher, no counter, and the metric still renders (as 0) so
// scrapers never see a missing series.
func TestDashboard_DroppedPatchCount_ZeroBeforeStart(t *testing.T) {
	t.Parallel()

	probe := &plainProber{resp: health.Response{Status: health.StatusPass}}
	dash := New(probe)

	if got := dash.DroppedPatchCount(); got != 0 {
		t.Errorf("DroppedPatchCount before Start = %d, want 0", got)
	}

	rec := httptest.NewRecorder()
	dash.MetricsHandler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/metrics", nil))

	line, ok := findMetricLine(rec.Body.String(), "dashboard_sse_drops_total ")
	if !ok || line != "0" {
		t.Errorf("metrics dashboard_sse_drops_total = %q (found=%t), want \"0\"", line, ok)
	}
}

// findMetricLine returns the value of the first metrics line whose name
// matches (exact prefix, so dashboard_sse_drops_total never matches a
// different family member) and strips the Prometheus "name " prefix.
func findMetricLine(body, namePrefix string) (string, bool) {
	for line := range strings.SplitSeq(body, "\n") {
		if value, ok := strings.CutPrefix(line, namePrefix); ok {
			return value, true
		}
	}

	return "", false
}
