package dashboard

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	health "github.com/larsartmann/go-health"
)

// webhookTimeout bounds a single webhook delivery. Best-effort by design:
// the receiver (or an intermediary like Gatus) owns alert thresholds and
// retries, so a failed delivery is dropped, never retried here.
const webhookTimeout = 10 * time.Second

// webhookMaxInFlight bounds concurrent deliveries: a flapping status against
// a slow receiver must not accumulate unbounded goroutines across push ticks.
const webhookMaxInFlight = 8

// webhookNotifier pushes health-state transitions to a caller-provided HTTP
// endpoint (WithWebhook). It exists so egress-restricted deployments — NAT,
// serverless, locked-down clusters — can be observed without a scraper
// polling the health endpoints, and so JSON receivers (event ingests, n8n,
// custom handlers) can consume transitions without scraping HTML or parsing
// Prometheus text.
//
// Deliveries are change-only: a fire happens when the overall status or the
// per-check fingerprint differs from the last announced state, independent of
// PushMode (a PushAlways dashboard streams every tick over SSE but must not
// spam a webhook). The first observation is always announced so the receiver
// learns the initial state, mirroring how the SSE handler sends current state
// on connect.
//
// Each fire runs in its own goroutine bounded by webhookTimeout, so a slow
// receiver can never block or wedge the SSE push loop.
type webhookNotifier struct {
	url     string
	headers map[string]string
	public  bool
	client  *http.Client

	mu              sync.Mutex
	lastStatus      health.Status
	lastFingerprint string
	announced       bool

	inFlight atomic.Int64
}

// newWebhookNotifier builds the notifier from configuration. Returns nil when
// no webhook URL is configured — the pusher treats nil as "disabled".
func newWebhookNotifier(cfg Config) *webhookNotifier {
	if cfg.WebhookURL == "" {
		return nil
	}

	return &webhookNotifier{
		url:     cfg.WebhookURL,
		headers: cfg.WebhookHeaders,
		public:  cfg.PublicMode,
		client:  &http.Client{Timeout: webhookTimeout},
	}
}

// webhookPayload is the transition snapshot POSTed as JSON. Field naming
// mirrors the go-health wire format (snake_case). ChangedAt is the detection
// time of the transition (± one push interval).
type webhookPayload struct {
	Status       string                  `json:"status"`
	ShuttingDown bool                    `json:"shutting_down,omitempty"`
	Checks       map[string]webhookCheck `json:"checks"`
	ChangedAt    string                  `json:"changed_at"`
}

// webhookCheck is one check's transition entry.
type webhookCheck struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// fireOnChange announces resp when it differs from the last announced state.
// Called from the push loop; never blocks on network I/O. The URL is
// deliberately never logged — it may embed a bearer secret.
func (n *webhookNotifier) fireOnChange(resp health.Response) {
	fp := fingerprintChecks(resp.Checks)

	n.mu.Lock()
	if n.announced && resp.Status == n.lastStatus && fp == n.lastFingerprint {
		n.mu.Unlock()

		return
	}
	n.lastStatus = resp.Status
	n.lastFingerprint = fp

	n.announced = true
	n.mu.Unlock()

	go n.post(resp)
}

// post delivers the snapshot. Failures are silent by contract: the endpoint
// is someone else's ingest, and alerting on a failed alert path belongs to
// the operator's monitoring stack, not to this library (zero-logging policy).
func (n *webhookNotifier) post(resp health.Response) {
	if n.inFlight.Add(1) > webhookMaxInFlight {
		// Bound concurrent deliveries under a flapping receiver: a slow or
		// broken endpoint must not accumulate unbounded goroutines across
		// push ticks. Later transitions coalesce into the next fire.
		n.inFlight.Add(-1)

		return
	}
	defer n.inFlight.Add(-1)

	payload := n.buildPayload(resp)

	body, err := json.Marshal(payload, json.Deterministic(true))
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		n.url,
		strings.NewReader(string(body)),
	)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	for k, v := range n.headers {
		req.Header.Set(k, v)
	}

	httpResp, err := n.client.Do(req)
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	// Drain so the connection returns to the pool.
	_, _ = io.Copy(io.Discard, httpResp.Body)
}

// buildPayload snapshots resp for the wire. Public mode masks check names to
// check-N (sorted, the same scheme as the metrics endpoint) and strips error
// details, so the webhook can point at untrusted receivers.
func (n *webhookNotifier) buildPayload(resp health.Response) webhookPayload {
	checks := make(map[string]webhookCheck, len(resp.Checks))

	for i, name := range sortedCheckNames(resp.Checks) {
		check := resp.Checks[name]

		key := name
		errText := check.Error

		if n.public {
			key = fmt.Sprintf("check-%d", i+1)
			errText = ""
		}

		checks[key] = webhookCheck{Status: string(check.Status), Error: errText}
	}

	return webhookPayload{
		Status:       string(resp.Status),
		ShuttingDown: resp.ShuttingDown,
		Checks:       checks,
		ChangedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}
