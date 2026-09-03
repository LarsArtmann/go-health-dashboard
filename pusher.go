package dashboard

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	dstar "github.com/larsartmann/go-datastar"
	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/go-sse"
)

// PushMode controls when the SSE pusher sends updates to connected clients.
type PushMode string

const (
	// PushOnChange broadcasts only when the overall status or any individual
	// check result changes (default). Minimises SSE traffic for NOC monitors
	// that stay connected for long periods.
	PushOnChange PushMode = "on-change"

	// PushAlways broadcasts on every tick, regardless of whether anything
	// changed. Use this when you want continuous confirmation that the
	// pusher is alive.
	PushAlways PushMode = "always"
)

const (
	// healthRegionID is the DOM element ID that Datastar patches on each update.
	healthRegionID = "health-region"
)

// pusher periodically reads the probe's cached response and broadcasts
// Datastar element patches to all connected SSE clients via a go-sse
// Broadcaster. Only one pusher goroutine runs per Dashboard instance.
type pusher struct {
	broadcaster   *sse.Broadcaster[sse.Event]
	dashboard     *Dashboard
	interval      time.Duration
	pushMode      PushMode
	heartbeat     time.Duration
	maxConns      int
	retry         time.Duration
	maxLifetime   time.Duration
	connections   atomic.Int64
	lastBroadcast atomic.Int64
	history       *historyBuffer

	mu              sync.Mutex
	lastStatus      health.Status
	lastFingerprint string
}

// newPusher creates a pusher that broadcasts at the dashboard's configured
// push interval. When TrendSamples is configured, the pusher also maintains
// a ring buffer of recent status samples for the trend sparkline.
func newPusher(d *Dashboard) *pusher {
	var history *historyBuffer
	if d.cfg.TrendSamples > 0 {
		history = newHistoryBuffer(d.cfg.TrendSamples)
	}

	return &pusher{
		broadcaster: sse.NewBroadcaster[sse.Event](),
		dashboard:   d,
		interval:    d.cfg.PushInterval,
		pushMode:    d.cfg.PushMode,
		heartbeat:   d.cfg.HeartbeatInterval,
		maxConns:    d.cfg.MaxSSEConnections,
		retry:       d.cfg.RetryInterval,
		maxLifetime: d.cfg.MaxConnectionLifetime,
		history:     history,
	}
}

// sample is one recorded health observation: the numeric value the
// sparkline plots plus the raw status and the observation time. Timestamps
// power the status timeline, the trend JSON endpoint, and the CSV export.
type sample struct {
	At     time.Time
	Value  float64
	Status string
}

// historyBuffer is a fixed-capacity ring buffer of status samples in
// chronological order. The pusher goroutine records on every tick; the SSE
// handler snapshots from other goroutines when rendering initial state, so
// all access is mutex-guarded.
type historyBuffer struct {
	mu      sync.Mutex
	samples []sample
	next    int
	full    bool
}

func newHistoryBuffer(capacity int) *historyBuffer {
	if capacity < 1 {
		capacity = 1
	}

	return &historyBuffer{samples: make([]sample, capacity)}
}

// record appends a sample, overwriting the oldest once at capacity.
func (h *historyBuffer) record(s sample) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.samples[h.next] = s
	h.next = (h.next + 1) % len(h.samples)

	if h.next == 0 {
		h.full = true
	}
}

// snapshot returns the recorded samples oldest-first, or nil when empty.
func (h *historyBuffer) snapshot() []sample {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.full && h.next == 0 {
		return nil
	}

	out := make([]sample, 0, len(h.samples))
	if h.full {
		out = append(out, h.samples[h.next:]...)
	}

	return append(out, h.samples[:h.next]...)
}

// statusTransition is one flip in the recorded status history.
type statusTransition struct {
	At   time.Time
	From string
	To   string
}

// transitions derives the status changes from the recorded samples,
// oldest-first. The first sample has no predecessor and never produces a
// transition.
func (h *historyBuffer) transitions() []statusTransition {
	samples := h.snapshot()

	var out []statusTransition

	for i := 1; i < len(samples); i++ {
		if samples[i].Status != samples[i-1].Status {
			out = append(out, statusTransition{
				At:   samples[i].At,
				From: samples[i-1].Status,
				To:   samples[i].Status,
			})
		}
	}

	return out
}

// start runs the push loop until ctx is cancelled, then closes the broadcaster.
func (p *pusher) start(ctx context.Context) {
	p.broadcast()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.broadcaster.Close()

			return
		case <-ticker.C:
			p.broadcast()
		}
	}
}

// broadcast renders the current dashboard content as a Datastar patch and
// sends it to all subscribers. Respects PushMode for change detection. Every
// tick records a trend sample, regardless of whether a patch is broadcast.
func (p *pusher) broadcast() {
	p.lastBroadcast.Store(time.Now().UnixNano())

	resp := p.dashboard.currentResponse()

	if p.history != nil {
		now := time.Now()

		p.history.record(sample{
			At:     now,
			Value:  statusValue(resp.Status),
			Status: string(resp.Status),
		})
	}

	if p.dashboard.latency != nil {
		p.dashboard.latency.observe(msToSeconds(resp.TotalLatencyMs))
	}

	if !p.shouldBroadcast(resp) {
		return
	}

	// Announce the transition to the webhook before rendering the SSE patch:
	// webhook delivery is state propagation and must not depend on rendering
	// success. fireOnChange is change-only and non-blocking.
	if n := p.dashboard.notify; n != nil {
		n.fireOnChange(resp)
	}

	evt, ok := p.renderPatch(resp)
	if !ok {
		return
	}

	p.broadcaster.Broadcast(evt)
}

// renderPatch renders the dashboard content to a Datastar ElementsPatch
// and returns the resulting sse.Event. Returns ok=false if rendering fails.
func (p *pusher) renderPatch(resp health.Response) (sse.Event, bool) {
	vm := buildViewModel(resp, p.dashboard.cfg.Title, p.dashboard.cfg.Routes.SSE)
	vm.CSSPath = p.dashboard.cfg.CSSPath
	vm.DatastarSrc = p.dashboard.cfg.DatastarSrc
	vm.ShowStatCards = !p.dashboard.cfg.HideStatCards

	if p.history != nil {
		populateHistory(&vm, p.history)
	}

	content := dashboardContent(vm)

	patch, err := dstar.ElementsFromTempl(content,
		dstar.WithModeInner(),
		dstar.WithSelectorID(healthRegionID),
	)
	if err != nil {
		return sse.Event{}, false
	}

	evt := patch.Event()
	if p.retry > 0 {
		evt.Retry = uint(p.retry.Milliseconds())
	}

	return evt, true
}

// shouldBroadcast returns true when the pusher should send an update based
// on the PushMode and whether anything changed since the last broadcast.
func (p *pusher) shouldBroadcast(resp health.Response) bool {
	if p.pushMode == PushAlways {
		return true
	}

	fp := fingerprintChecks(resp.Checks)

	p.mu.Lock()
	defer p.mu.Unlock()

	if resp.Status != p.lastStatus || fp != p.lastFingerprint {
		p.lastStatus = resp.Status
		p.lastFingerprint = fp

		return true
	}

	return false
}

// atCapacity reports (and writes) the 503 shown when the SSE connection
// limit is configured and already reached.
func (p *pusher) atCapacity(w http.ResponseWriter) bool {
	if p.maxConns <= 0 || p.connections.Load() < int64(p.maxConns) {
		return false
	}

	http.Error(w, "dashboard: too many SSE connections", http.StatusServiceUnavailable)

	return true
}

// sseHandler upgrades to an SSE connection, sends the initial state as a
// Datastar patch, then forwards broadcaster events to the client. Blocks
// until the client disconnects or the pusher shuts down.
func (d *Dashboard) sseHandler(w http.ResponseWriter, r *http.Request) {
	push := d.push.Load()
	if push == nil {
		http.Error(w, "dashboard: SSE push is not active", http.StatusServiceUnavailable)

		return
	}

	if push.atCapacity(w) {
		return
	}

	push.connections.Add(1)
	defer push.connections.Add(-1)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	// Send initial state so the client doesn't wait for the next tick.
	resp := d.currentResponse()
	if evt, ok := push.renderPatch(resp); ok {
		if err := stream.Send(evt); err != nil {
			return
		}
	}

	ch := push.broadcaster.Subscribe()
	defer push.broadcaster.Unsubscribe(ch)

	// Heartbeat goroutine prevents proxy timeouts on long-lived connections.
	go stream.Heartbeat(r.Context(), push.heartbeat)

	var lifetime <-chan time.Time

	if push.maxLifetime > 0 {
		timer := time.NewTimer(push.maxLifetime)
		defer timer.Stop()

		lifetime = timer.C
	}

	for {
		select {
		case <-stream.Context().Done():
			return
		case <-lifetime:
			// Connection lifetime cap: close the stream; the browser
			// reconnects automatically (SSE semantics) and receives fresh
			// state on connect.
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}

			if err := stream.Send(evt); err != nil {
				return
			}
		}
	}
}

// msToSeconds converts milliseconds to seconds.
func msToSeconds(ms int64) float64 {
	const millisPerSecond = 1000

	return float64(ms) / millisPerSecond
}
