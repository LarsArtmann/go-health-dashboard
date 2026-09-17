package dashboard_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// --- SSE race-stress (plan M82) ---

// TestSSE_RaceStress_SubscriberCountConsistency hammers the pusher's
// connection accounting with 50 concurrent clients under -race: the count
// must reach exactly `clients` while all streams are open and return to
// exactly zero after every body is closed — never a negative or
// stuck-in-between value.
func TestSSE_RaceStress_SubscriberCountConsistency(t *testing.T) {
	t.Parallel()

	const clients = 50

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(t, 50*time.Millisecond, svc)
	defer cleanup()

	var (
		mutex  sync.Mutex
		bodies []*http.Response
		wg     sync.WaitGroup
	)

	wg.Add(clients)

	for range clients {
		go func() {
			defer wg.Done()

			resp, err := http.Get(server.URL + "/health/sse")
			if err != nil {
				t.Errorf("client connect: %v", err)

				return
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("client connect: want 200, got %d", resp.StatusCode)
				_ = resp.Body.Close()

				return
			}

			mutex.Lock()
			//nolint:bodyclose // the body closes in the mutex-guarded sweep after wg.Wait
			bodies = append(bodies, resp)
			mutex.Unlock()
		}()
	}

	wg.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for dash.SubscriberCount() != clients {
		if time.Now().After(deadline) {
			t.Fatalf(
				"SubscriberCount with %d concurrent clients: want %d, got %d",
				clients,
				clients,
				dash.SubscriberCount(),
			)
		}

		time.Sleep(10 * time.Millisecond)
	}

	mutex.Lock()
	for _, body := range bodies {
		_ = body.Body.Close()
	}
	bodies = nil
	mutex.Unlock()

	deadline = time.Now().Add(5 * time.Second)
	for dash.SubscriberCount() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf(
				"SubscriberCount after closing all clients: want 0, got %d",
				dash.SubscriberCount(),
			)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// --- Webhook delivery under concurrent transitions (plan M83) ---

// TestWebhook_ConcurrentTransitionsDeliverExactlyOnce pins the webhook's
// delivery contract when the status flaps. Fire count is transition-based
// (the notifier announces a state only when it differs from the last
// announced one), while DELIVERY order is best-effort: one goroutine per
// fire, no retries, and the payload's second-granular changed_at makes
// same-second fires byte-indistinguishable — so the receiver-side
// assertions are set-based, never sequence-based. That asymmetry is the
// documented contract this test locks.
func TestWebhook_ConcurrentTransitionsDeliverExactlyOnce(t *testing.T) {
	t.Parallel()

	var (
		mutex     sync.Mutex
		delivered []string
	)

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		mutex.Lock()
		delivered = append(delivered, string(body))
		mutex.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	svc := &toggleService{}
	svc.healthy.Store(false)

	_, _, cleanup := setupSSEServer(t, 30*time.Millisecond, svc,
		dashboard.WithWebhook(receiver.URL),
	)
	defer cleanup()

	for i := range 8 {
		if i%2 == 0 {
			svc.healthy.Store(true)
		} else {
			svc.healthy.Store(false)
		}

		time.Sleep(60 * time.Millisecond)
	}

	// Let the last fire (one goroutine per delivery, bounded by 10s but
	// local in practice) land before auditing the receiver.
	deadline := time.Now().Add(3 * time.Second)
	for {
		mutex.Lock()
		count := len(delivered)
		mutex.Unlock()

		if count >= 4 {
			break
		}

		if time.Now().After(deadline) {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	mutex.Lock()
	payloads := append([]string(nil), delivered...)
	mutex.Unlock()

	if len(payloads) < 3 {
		t.Fatalf("deliveries: want >= 3 transitions, got %d", len(payloads))
	}

	// Transition-based fires: 8 toggles plus the initial announcement can
	// produce at most 9 fires — more means the notifier is spamming per
	// tick instead of per change.
	if len(payloads) > 9 {
		t.Errorf("deliveries: want <= 9 (initial + 8 toggles), got %d", len(payloads))
	}

	sawPass, sawFail, finalArrived := auditWebhookDeliveries(t, payloads)

	if !sawPass || !sawFail {
		t.Errorf("both toggled states must arrive (pass=%v fail=%v)", sawPass, sawFail)
	}

	if !finalArrived {
		t.Errorf(
			"the final state (fail) never arrived; delivered:\n%s",
			strings.Join(payloads, "\n"),
		)
	}
}

// auditWebhookDeliveries validates every delivered payload's structure and
// reports which states arrived. Malformed payloads fail the test inline;
// the booleans summarize the observed status set.
func auditWebhookDeliveries(t *testing.T, payloads []string) (bool, bool, bool) {
	t.Helper()

	var sawPass, sawFail, finalArrived bool

	for _, raw := range payloads {
		var payload struct {
			Status string `json:"status"`
			Checks map[string]struct {
				Status string `json:"status"`
				Since  any    `json:"since"`
			} `json:"checks"`
		}

		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("invalid delivery payload %s: %v", raw, err)
		}

		switch payload.Status {
		case "pass":
			sawPass = true
		case "fail":
			sawFail = true
		default:
			t.Errorf("unexpected delivered status %q", payload.Status)
		}

		for name, check := range payload.Checks {
			if check.Since != nil {
				t.Errorf(
					"check %q: the v0.2.0 metadata must stay off the webhook wire (since=%v)",
					name,
					check.Since,
				)
			}
		}

		if payload.Status == "fail" {
			finalArrived = true
		}
	}

	return sawPass, sawFail, finalArrived
}

// --- Heartbeat goroutine lifetime (plan M84) ---

// TestShutdown_HeartbeatGoroutinesExit proves the per-connection heartbeat
// goroutines do not leak across Shutdown: with five hot-beat connections,
// after Shutdown plus closing every client the goroutine count returns to
// its baseline (within noise) instead of staying elevated by one leaked
// ticker per connection.
func TestShutdown_HeartbeatGoroutinesExit(t *testing.T) {
	t.Parallel()

	svc := &toggleService{}
	svc.healthy.Store(true)

	server, dash, cleanup := setupSSEServer(t, 50*time.Millisecond, svc,
		dashboard.WithHeartbeatInterval(50*time.Millisecond),
	)
	defer cleanup()

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	const clients = 5

	type openClient struct {
		resp *http.Response
	}

	open := make([]openClient, 0, clients)

	defer func() {
		for _, c := range open {
			_ = c.resp.Body.Close()
		}
	}()

	for range clients {
		resp, err := http.Get(
			server.URL + "/health/sse",
		) //nolint:bodyclose // closed via the deferred open-slice sweep
		if err != nil {
			t.Fatalf("client connect: %v", err)
		}

		open = append(open, openClient{resp: resp})
	}

	deadline := time.Now().Add(5 * time.Second)
	for dash.SubscriberCount() != clients {
		if time.Now().After(deadline) {
			t.Fatalf("SubscriberCount: want %d, got %d", clients, dash.SubscriberCount())
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Every heartbeat goroutine is now beating every 50ms; a leak shows up
	// as five goroutines that survive Shutdown.
	dash.Shutdown()

	for _, c := range open {
		_ = c.resp.Body.Close()
	}

	open = nil

	deadline = time.Now().Add(3 * time.Second)
	for {
		runtime.GC()

		now := runtime.NumGoroutine()
		if dash.SubscriberCount() == 0 && now <= baseline+2 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"goroutines after shutdown: baseline %d, now %d (subscribers %d) — heartbeat goroutines leak",
				baseline,
				now,
				dash.SubscriberCount(),
			)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
