package dashboard

import (
	"encoding/json/v2"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

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

	notifier := newWebhookNotifier(Config{})

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

	notifier := newWebhookNotifier(Config{PublicMode: true})

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

func assertWebhookWireShape(t *testing.T, payload webhookPayload, wantChecks map[string]map[string]string) {
	t.Helper()

	body, err := json.Marshal(payload, json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal webhook payload: %v", err)
	}

	var decoded map[string]any

	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal webhook payload %s: %v", body, err)
	}

	for key := range decoded {
		switch key {
		case "status", "checks", "changed_at":
		default:
			t.Errorf("unexpected top-level key %q in payload %s", key, body)
		}
	}

	if _, ok := decoded["shutting_down"]; ok {
		t.Error("shutting_down must stay omitted when false (omitempty contract)")
	}

	changedAt, ok := decoded["changed_at"].(string)
	if !ok || changedAt == "" {
		t.Fatalf("changed_at must be a non-empty RFC3339 string: %s", body)
	}

	if _, err := time.Parse(time.RFC3339, changedAt); err != nil {
		t.Errorf("changed_at %q is not RFC3339: %v", changedAt, err)
	}

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
