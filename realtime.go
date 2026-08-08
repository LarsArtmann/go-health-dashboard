package dashboard

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

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
	// sseHeartbeatInterval is how often the SSE handler sends a comment-line
	// keepalive to prevent proxy/load-balancer timeout.
	sseHeartbeatInterval = 15 * time.Second
	// sseEventType is the SSE event name for dashboard updates.
	sseEventType = "update"
)

// ssePusher periodically reads the probe's cached response and broadcasts
// the rendered dashboard content to all connected SSE clients via a
// Broadcaster. Only one pusher goroutine runs per Dashboard instance.
type ssePusher struct {
	broadcaster *sse.Broadcaster[sse.Event]
	dashboard   *Dashboard
	interval    time.Duration
	pushMode    PushMode

	mu          sync.Mutex
	lastStatus  health.Status
	lastChecks  string
}

// newSSEPusher creates a pusher that broadcasts at the dashboard's configured
// refresh interval.
func newSSEPusher(d *Dashboard) *ssePusher {
	return &ssePusher{
		broadcaster: sse.NewBroadcaster[sse.Event](),
		dashboard:   d,
		interval:    d.cfg.RefreshInterval,
		pushMode:    PushOnChange,
	}
}

// start runs the push loop until ctx is cancelled, then closes the broadcaster.
// Call once in a goroutine from Dashboard.Start.
func (p *ssePusher) start(ctx context.Context) {
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

// broadcast renders the current dashboard content and sends it to all
// subscribers. Respects PushMode for change detection.
func (p *ssePusher) broadcast() {
	resp := p.dashboard.currentResponse()

	if !p.shouldBroadcast(resp) {
		return
	}

	html, err := p.renderContent(resp)
	if err != nil {
		return
	}

	p.broadcaster.Broadcast(sse.Event{
		Event: sseEventType,
		Data:  html,
	})
}

// shouldBroadcast returns true when the pusher should send an update based
// on the PushMode and whether anything changed since the last broadcast.
func (p *ssePusher) shouldBroadcast(resp health.Response) bool {
	if p.pushMode == PushAlways {
		return true
	}

	checksHash := hashChecks(resp.Checks)

	p.mu.Lock()
	defer p.mu.Unlock()

	if resp.Status != p.lastStatus || checksHash != p.lastChecks {
		p.lastStatus = resp.Status
		p.lastChecks = checksHash
		return true
	}

	return false
}

// renderContent renders the dashboard inner content to an HTML string.
func (p *ssePusher) renderContent(resp health.Response) (string, error) {
	data := buildViewModel(
		resp,
		p.dashboard.cfg.Title,
		p.dashboard.cfg.Routes.Partial,
		"",
	)

	var buf bytes.Buffer
	if err := dashboardContent(data).Render(context.Background(), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// sseConnectionHandler upgrades to an SSE connection, subscribes to the
// broadcaster, and forwards events to the client. The handler blocks until
// the client disconnects or the pusher shuts down.
func (d *Dashboard) sseConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if d.push == nil {
		http.Error(w, "dashboard: SSE push mode is not active", http.StatusServiceUnavailable)
		return
	}

	stream := sse.NewStream(w, r)
	defer stream.Close()

	ch := d.push.broadcaster.Subscribe()
	defer d.push.broadcaster.Unsubscribe(ch)

	// Send initial state so the client doesn't wait for the next tick.
	initialHTML, err := d.push.renderContent(d.currentResponse())
	if err == nil {
		_ = stream.Send(sse.Event{Event: sseEventType, Data: initialHTML})
	}

	// Heartbeat goroutine prevents proxy timeouts on long-lived connections.
	go stream.Heartbeat(r.Context(), sseHeartbeatInterval)

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

// hashChecks creates a lightweight fingerprint of the checks map for change
// detection. Uses string concatenation of name+status+error rather than
// cryptographic hashing — we only need equality comparison.
func hashChecks(checks map[string]health.Check) string {
	var buf bytes.Buffer

	for name, check := range checks {
		buf.WriteString(name)
		buf.WriteByte(':')
		buf.WriteString(string(check.Status))
		buf.WriteByte(':')
		buf.WriteString(check.Error)
		buf.WriteByte(';')
	}

	return buf.String()
}
