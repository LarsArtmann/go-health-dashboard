package dashboard

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// webhookPinHookURL is only a constructor prerequisite — buildPayload is
// pure and never dials.
const webhookPinHookURL = "https://webhook.invalid/hook"

// webhookPinSince is the fixed state-entry time for the payload pin.
var webhookPinSince = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

// TestWebhookPayload_PinWireShape pins the transition payload's wire shape:
// status, checks{status,error}, and changed_at — nothing else. The
// go-health v0.2.0 metadata fields (Check.Since, DurationNanos)
// deliberately do NOT transit: the webhook announces transition facts, not
// display metadata. This test fails if the shape ever grows or drops a
// field silently.
func TestWebhookPayload_PinWireShape(t *testing.T) {
	t.Parallel()

	notifier := newWebhookNotifier(Config{WebhookURL: webhookPinHookURL})

	resp := health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{
			"api/postgres": {
				Status:        health.StatusPass,
				Since:         webhookPinSince,
				DurationNanos: int64(42 * time.Millisecond),
			},
			"db": {
				Status: health.StatusFail,
				Error:  "connection refused",
				Since:  webhookPinSince,
			},
		},
	}

	assertWebhookWireShape(t, notifier.buildPayload(resp), map[string]map[string]string{
		"api/postgres": {"status": "pass"},
		"db":           {"status": "fail", "error": "connection refused"},
	})
}

// TestWebhookPayload_PublicModeMasksNames pins the public-mode payload:
// names become check-N (sorted, the metrics masking scheme), error details
// are stripped, and — same as the named shape — the v0.2.0 metadata stays
// off the wire.
func TestWebhookPayload_PublicModeMasksNames(t *testing.T) {
	t.Parallel()

	notifier := newWebhookNotifier(Config{WebhookURL: webhookPinHookURL, PublicMode: true})

	resp := health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{
			"api/postgres": {
				Status:        health.StatusPass,
				Since:         webhookPinSince,
				DurationNanos: int64(42 * time.Millisecond),
			},
			"db": {
				Status: health.StatusFail,
				Error:  "connection refused",
				Since:  webhookPinSince,
			},
		},
	}

	assertWebhookWireShape(t, notifier.buildPayload(resp), map[string]map[string]string{
		"check-1": {"status": "pass"},
		"check-2": {"status": "fail"},
	})
}

// TestWebhookPayload_ShuttingDownTrue pins the true branch of the
// shutting-down flag at wire level: the payload carries
// `"shutting_down":true` as a real JSON bool (the false branch rides the
// wire too — jsonv2 omitempty keeps false bools — and is pinned via
// assertWebhookTopLevelKeys above).
func TestWebhookPayload_ShuttingDownTrue(t *testing.T) {
	t.Parallel()

	notifier := newWebhookNotifier(Config{WebhookURL: webhookPinHookURL})

	resp := health.Response{
		Status:       health.StatusFail,
		ShuttingDown: true,
		Checks: map[string]health.Check{
			"db": {Status: health.StatusFail, Error: "connection refused"},
		},
	}

	body, err := json.Marshal(notifier.buildPayload(resp), json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal webhook payload: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal webhook payload %s: %v", body, err)
	}

	flag, ok := decoded["shutting_down"].(bool)
	if !ok {
		t.Fatalf("shutting_down missing or not a JSON bool in payload %s", body)
	}
	if !flag {
		t.Errorf(
			"shutting_down = false, want true when the response carries the shutdown overlay (%s)",
			body,
		)
	}
}

func assertWebhookWireShape(
	t *testing.T,
	payload webhookPayload,
	wantChecks map[string]map[string]string,
) {
	t.Helper()

	body, err := json.Marshal(payload, json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal webhook payload: %v", err)
	}

	var decoded map[string]any

	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal webhook payload %s: %v", body, err)
	}

	assertWebhookTopLevelKeys(t, decoded, body)
	assertWebhookCheckEntries(t, decoded, wantChecks, body)
}

// assertWebhookTopLevelKeys verifies the payload's top-level shape: exactly
// status/checks/changed_at/shutting_down, with changed_at RFC3339-formatted.
// Under encoding/json/v2 semantics `omitempty` does not drop a false bool
// (that requires omitzero), so shutting_down:false legitimately rides the
// wire — this pin documents that reality.
func assertWebhookTopLevelKeys(t *testing.T, decoded map[string]any, body []byte) {
	t.Helper()

	for key := range decoded {
		switch key {
		case "status", "checks", "changed_at", "shutting_down":
		default:
			t.Errorf("unexpected top-level key %q in payload %s", key, body)
		}
	}

	if v, ok := decoded["shutting_down"]; ok {
		if _, isBool := v.(bool); !isBool {
			t.Errorf("shutting_down must be a JSON bool: %s", body)
		}
	}

	changedAt, ok := decoded["changed_at"].(string)
	if !ok || changedAt == "" {
		t.Fatalf("changed_at must be a non-empty RFC3339 string: %s", body)
	}

	if _, err := time.Parse(time.RFC3339, changedAt); err != nil {
		t.Errorf("changed_at %q is not RFC3339: %v", changedAt, err)
	}
}

// assertWebhookCheckEntries verifies each check entry carries exactly the
// expected status/error pair — any extra key (e.g. a v0.2.0 since or
// duration_ns leaking onto the wire) fails the pin.
func assertWebhookCheckEntries(
	t *testing.T,
	decoded map[string]any,
	wantChecks map[string]map[string]string,
	body []byte,
) {
	t.Helper()

	checks, ok := decoded["checks"].(map[string]any)
	if !ok {
		t.Fatalf("checks missing or not an object: %s", body)
	}

	if len(checks) != len(wantChecks) {
		t.Fatalf("checks count = %d, want %d (%s)", len(checks), len(wantChecks), body)
	}

	for name, want := range wantChecks {
		entry, ok := checks[name].(map[string]any)
		if !ok {
			t.Fatalf("check %q missing from payload %s", name, body)
		}

		for key := range entry {
			switch key {
			case "status", "error":
			default:
				t.Errorf(
					"check %q: unexpected key %q — the v0.2.0 metadata must stay off the webhook wire (%s)",
					name,
					key,
					body,
				)
			}
		}

		for key, wantVal := range want {
			if got, ok := entry[key]; !ok || got != wantVal {
				t.Errorf("check %q %s = %v, want %q (%s)", name, key, got, wantVal, body)
			}
		}
	}
}

// TestWebhook_BoundedInFlightVsSlowReceiver pins the bounded-delivery
// claim against a receiver that accepts requests but never answers: a
// flapping status must never put more than webhookMaxInFlight deliveries
// on the wire at once, excess transitions are dropped (not queued), and
// the bound releases once the receiver answers again — no permanent
// starvation.
func TestWebhook_BoundedInFlightVsSlowReceiver(t *testing.T) {
	t.Parallel()

	var (
		release   = make(chan struct{})
		inHandler atomic.Int64
		peak      atomic.Int64
		handled   atomic.Int64
	)

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := inHandler.Add(1)

		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}

		defer inHandler.Add(-1)
		defer handled.Add(1)

		// Bounded hang: release unblocks on schedule; the timer keeps a
		// failing test from deadlocking server teardown for the full
		// webhookTimeout.
		select {
		case <-release:
		case <-time.After(webhookTimeout + time.Second):
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()

	notifier := newWebhookNotifier(Config{WebhookURL: slow.URL})

	// 40 distinct transitions against a stuck receiver: the first
	// webhookMaxInFlight deliveries occupy the server, the rest must be
	// dropped at the gate instead of piling up.
	for i := 0; i < 40; i++ {
		notifier.fireOnChange(health.Response{
			Status: health.StatusWarn,
			Checks: map[string]health.Check{
				fmt.Sprintf("svc/%d", i): {Status: health.StatusWarn},
			},
		})
	}

	waitFor(t, 3*time.Second, func() bool { return inHandler.Load() >= 1 })

	if got := peak.Load(); got > webhookMaxInFlight {
		t.Fatalf("slow receiver saw %d concurrent deliveries, bound is %d", got, webhookMaxInFlight)
	}

	// Give any would-be ninth delivery a window to (wrongly) arrive.
	time.Sleep(50 * time.Millisecond)

	if got := peak.Load(); got > webhookMaxInFlight {
		t.Fatalf("bound exceeded after settle window: %d concurrent, limit %d", got, webhookMaxInFlight)
	}

	if got := handled.Load(); got > webhookMaxInFlight {
		t.Fatalf("stuck receiver handled %d deliveries, bound is %d", got, webhookMaxInFlight)
	}

	// Release the receiver and let the bounded set drain.
	close(release)

	waitFor(t, webhookTimeout, func() bool { return inHandler.Load() == 0 })

	// The bound must recover: a fresh transition after the drain reaches
	// the receiver (entry-side counter — handled only ticks on return, so
	// waiting on it here would deadlock against the release below).
	release = make(chan struct{})

	notifier.fireOnChange(health.Response{
		Status: health.StatusFail,
		Checks: map[string]health.Check{"svc/post-drain": {Status: health.StatusFail}},
	})

	waitFor(t, 3*time.Second, func() bool { return inHandler.Load() == 1 })

	close(release)

	waitFor(t, webhookTimeout, func() bool { return inHandler.Load() == 0 })
}

// waitFor polls cond until it holds or the deadline expires, failing the
// test with the message-free verdict "condition never became true".
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for !cond() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if !cond() {
		t.Fatal("condition never became true before the deadline")
	}
}
