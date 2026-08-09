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
	broadcaster *sse.Broadcaster[sse.Event]
	dashboard   *Dashboard
	interval    time.Duration
	pushMode    PushMode
	heartbeat   time.Duration
	maxConns    int
	retry       time.Duration
	connections atomic.Int64

	mu              sync.Mutex
	lastStatus      health.Status
	lastFingerprint string
}

// newPusher creates a pusher that broadcasts at the dashboard's configured
// push interval.
func newPusher(d *Dashboard) *pusher {
	return &pusher{
		broadcaster: sse.NewBroadcaster[sse.Event](),
		dashboard:   d,
		interval:    d.cfg.PushInterval,
		pushMode:    d.cfg.PushMode,
		heartbeat:   d.cfg.HeartbeatInterval,
		maxConns:    d.cfg.MaxSSEConnections,
		retry:       d.cfg.RetryInterval,
	}
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
// sends it to all subscribers. Respects PushMode for change detection.
func (p *pusher) broadcast() {
	resp := p.dashboard.currentResponse()

	if !p.shouldBroadcast(resp) {
		return
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

// sseHandler upgrades to an SSE connection, sends the initial state as a
// Datastar patch, then forwards broadcaster events to the client. Blocks
// until the client disconnects or the pusher shuts down.
func (d *Dashboard) sseHandler(w http.ResponseWriter, r *http.Request) {
	push := d.push.Load()
	if push == nil {
		http.Error(w, "dashboard: SSE push is not active", http.StatusServiceUnavailable)

		return
	}

	if push.maxConns > 0 && push.connections.Load() >= int64(push.maxConns) {
		http.Error(w, "dashboard: too many SSE connections", http.StatusServiceUnavailable)

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

	for {
		select {
		case <-stream.Context().Done():
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
